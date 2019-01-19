
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package namesys

import (
	"context"
	"strings"
	"time"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"

	opts "github.com/ipfs/go-ipfs/namesys/opts"

	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	lru "gx/ipfs/QmQjMHF8ptRgx4E57UFMiT4YM6kqaJeYxZ1MCDX23aw4rK/golang-lru"
	routing "gx/ipfs/QmTiRqrF5zkdZyrdsL5qndG1UbeWi8k8N2pYxCtXWrahR2/go-libp2p-routing"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
)

//MPN（多协议命名系统）实现通用IPF命名。
//
//使用多个冲突解决程序：
//（a）ipfs路由命名：与pki名称类似的sfs。
//（B）DNS域：使用DNS TXT记录中的链接解析
//（C）ProQuints：将字符串解释为原始字节数据。
//
//它只能发布到：（a）IPFS路由命名。
//
type mpns struct {
	dnsResolver, proquintResolver, ipnsResolver resolver
	ipnsPublisher                               Publisher

	cache *lru.Cache
}

//NewNameSystem将基于路由构建IPFS命名系统
func NewNameSystem(r routing.ValueStore, ds ds.Datastore, cachesize int) NameSystem {
	var cache *lru.Cache
	if cachesize > 0 {
		cache, _ = lru.New(cachesize)
	}

	return &mpns{
		dnsResolver:      NewDNSResolver(),
		proquintResolver: new(ProquintResolver),
		ipnsResolver:     NewIpnsResolver(r),
		ipnsPublisher:    NewIpnsPublisher(r, ds),
		cache:            cache,
	}
}

const DefaultResolverCacheTTL = time.Minute

//解析实现解析程序。
func (ns *mpns) Resolve(ctx context.Context, name string, options ...opts.ResolveOpt) (path.Path, error) {
	if strings.HasPrefix(name, "/ipfs/") {
		return path.ParsePath(name)
	}

	if !strings.HasPrefix(name, "/") {
		return path.ParsePath("/ipfs/" + name)
	}

	return resolve(ctx, ns, name, opts.ProcessOpts(options))
}

func (ns *mpns) ResolveAsync(ctx context.Context, name string, options ...opts.ResolveOpt) <-chan Result {
	res := make(chan Result, 1)
	if strings.HasPrefix(name, "/ipfs/") {
		p, err := path.ParsePath(name)
		res <- Result{p, err}
		return res
	}

	if !strings.HasPrefix(name, "/") {
		p, err := path.ParsePath("/ipfs/" + name)
		res <- Result{p, err}
		return res
	}

	return resolveAsync(ctx, ns, name, opts.ProcessOpts(options))
}

//resolveonce实现resolver。
func (ns *mpns) resolveOnceAsync(ctx context.Context, name string, options opts.ResolveOpts) <-chan onceResult {
	out := make(chan onceResult, 1)

	if !strings.HasPrefix(name, ipnsPrefix) {
		name = ipnsPrefix + name
	}
	segments := strings.SplitN(name, "/", 4)
	if len(segments) < 3 || segments[0] != "" {
		log.Debugf("invalid name syntax for %s", name)
		out <- onceResult{err: ErrResolveFailed}
		close(out)
		return out
	}

	key := segments[2]

	if p, ok := ns.cacheGet(key); ok {
		if len(segments) > 3 {
			var err error
			p, err = path.FromSegments("", strings.TrimRight(p.String(), "/"), segments[3])
			if err != nil {
				emitOnceResult(ctx, out, onceResult{value: p, err: err})
			}
		}

		out <- onceResult{value: p}
		close(out)
		return out
	}

//解析器选择：
//1。如果是通过“IPN”进行多哈希解析。
//2。如果是域名，通过“dns”解析
//三。否则通过“Proquint”分解器解决

	var res resolver
	if _, err := mh.FromB58String(key); err == nil {
		res = ns.ipnsResolver
	} else if isd.IsDomain(key) {
		res = ns.dnsResolver
	} else {
		res = ns.proquintResolver
	}

	resCh := res.resolveOnceAsync(ctx, key, options)
	var best onceResult
	go func() {
		defer close(out)
		for {
			select {
			case res, ok := <-resCh:
				if !ok {
					if best != (onceResult{}) {
						ns.cacheSet(key, best.value, best.ttl)
					}
					return
				}
				if res.err == nil {
					best = res
				}
				p := res.value

//附加路径的其余部分
				if len(segments) > 3 {
					var err error
					p, err = path.FromSegments("", strings.TrimRight(p.String(), "/"), segments[3])
					if err != nil {
						emitOnceResult(ctx, out, onceResult{value: p, ttl: res.ttl, err: err})
					}
				}

				emitOnceResult(ctx, out, onceResult{value: p, ttl: res.ttl, err: res.err})
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}

func emitOnceResult(ctx context.Context, outCh chan<- onceResult, r onceResult) {
	select {
	case outCh <- r:
	case <-ctx.Done():
	}
}

//发布实现发布服务器
func (ns *mpns) Publish(ctx context.Context, name ci.PrivKey, value path.Path) error {
	return ns.PublishWithEOL(ctx, name, value, time.Now().Add(DefaultRecordTTL))
}

func (ns *mpns) PublishWithEOL(ctx context.Context, name ci.PrivKey, value path.Path, eol time.Time) error {
	id, err := peer.IDFromPrivateKey(name)
	if err != nil {
		return err
	}
	if err := ns.ipnsPublisher.PublishWithEOL(ctx, name, value, eol); err != nil {
		return err
	}
	ttl := DefaultResolverCacheTTL
	if ttEol := eol.Sub(time.Now()); ttEol < ttl {
		ttl = ttEol
	}
	ns.cacheSet(peer.IDB58Encode(id), value, ttl)
	return nil
}
