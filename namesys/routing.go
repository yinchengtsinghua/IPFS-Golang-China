
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

	dht "gx/ipfs/QmNoNExMdWrYSPZDiJJTVmxSh6uKLN26xYVzbLzBLedRcv/go-libp2p-kad-dht"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	routing "gx/ipfs/QmTiRqrF5zkdZyrdsL5qndG1UbeWi8k8N2pYxCtXWrahR2/go-libp2p-routing"
	ipns "gx/ipfs/QmWPFehHmySCdaGttQ48iwF7M6mBRrGE5GSPWKCuMWqJDR/go-ipns"
	pb "gx/ipfs/QmWPFehHmySCdaGttQ48iwF7M6mBRrGE5GSPWKCuMWqJDR/go-ipns/pb"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	proto "gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

var log = logging.Logger("namesys")

//ipsresolver为主要的ipfs sfs（如命名）实现nsresolver
type IpnsResolver struct {
	routing routing.ValueStore
}

//newipsresolver使用ipfs路由系统构造名称解析程序
//在顶部实现类似于命名的SFS。
func NewIpnsResolver(route routing.ValueStore) *IpnsResolver {
	if route == nil {
		panic("attempt to create resolver with nil routing system")
	}
	return &IpnsResolver{
		routing: route,
	}
}

//解析实现解析程序。
func (r *IpnsResolver) Resolve(ctx context.Context, name string, options ...opts.ResolveOpt) (path.Path, error) {
	return resolve(ctx, r, name, opts.ProcessOpts(options))
}

//resolveasync实现resolver。
func (r *IpnsResolver) ResolveAsync(ctx context.Context, name string, options ...opts.ResolveOpt) <-chan Result {
	return resolveAsync(ctx, r, name, opts.ProcessOpts(options))
}

//resolveonce实现resolver。使用IPFS路由系统
//解析类似于sfs的名称。
func (r *IpnsResolver) resolveOnceAsync(ctx context.Context, name string, options opts.ResolveOpts) <-chan onceResult {
	out := make(chan onceResult, 1)
	log.Debugf("RoutingResolver resolving %s", name)
	cancel := func() {}

	if options.DhtTimeout != 0 {
//解决方案必须在超时内完成
		ctx, cancel = context.WithTimeout(ctx, options.DhtTimeout)
	}

	name = strings.TrimPrefix(name, "/ipns/")
	pid, err := peer.IDB58Decode(name)
	if err != nil {
		log.Debugf("RoutingResolver: could not convert public key hash %s to peer ID: %s\n", name, err)
		out <- onceResult{err: err}
		close(out)
		cancel()
		return out
	}

//名称应该是可从IPF检索的公钥的哈希。
//我们在这里检索公钥以确保它在对等机中
//在对DHT调用getValue（）之前存储-DHT将调用
//IPN验证器，它反过来从对等端获取公钥
//存储以验证记录签名
	_, err = routing.GetPublicKey(r.routing, ctx, pid)
	if err != nil {
		log.Debugf("RoutingResolver: could not retrieve public key %s: %s\n", name, err)
		out <- onceResult{err: err}
		close(out)
		cancel()
		return out
	}

//使用路由系统获取名称。
//请注意，DHT在检索时将调用IPN验证器
//值，该值依次验证IPN记录签名
	ipnsKey := ipns.RecordKey(pid)

	vals, err := r.routing.SearchValue(ctx, ipnsKey, dht.Quorum(int(options.DhtRecordCount)))
	if err != nil {
		log.Debugf("RoutingResolver: dht get for name %s failed: %s", name, err)
		out <- onceResult{err: err}
		close(out)
		cancel()
		return out
	}

	go func() {
		defer cancel()
		defer close(out)
		for {
			select {
			case val, ok := <-vals:
				if !ok {
					return
				}

				entry := new(pb.IpnsEntry)
				err = proto.Unmarshal(val, entry)
				if err != nil {
					log.Debugf("RoutingResolver: could not unmarshal value for name %s: %s", name, err)
					emitOnceResult(ctx, out, onceResult{err: err})
					return
				}

				var p path.Path
//检查旧样式记录：
				if valh, err := mh.Cast(entry.GetValue()); err == nil {
//
					log.Debugf("encountered CIDv0 ipns entry: %s", valh)
					p = path.FromCid(cid.NewCidV0(valh))
				} else {
//不是多哈希，可能是新的样式记录
					p, err = path.ParsePath(string(entry.GetValue()))
					if err != nil {
						emitOnceResult(ctx, out, onceResult{err: err})
						return
					}
				}

				ttl := DefaultResolverCacheTTL
				if entry.Ttl != nil {
					ttl = time.Duration(*entry.Ttl)
				}
				switch eol, err := ipns.GetEOL(entry); err {
				case ipns.ErrUnrecognizedValidity:
//没有EOL。
				case nil:
					ttEol := eol.Sub(time.Now())
					if ttEol < 0 {
//当我们第一次解决它时它是有效的。
						ttl = 0
					} else if ttEol < ttl {
						ttl = ttEol
					}
				default:
					log.Errorf("encountered error when parsing EOL: %s", err)
					emitOnceResult(ctx, out, onceResult{err: err})
					return
				}

				emitOnceResult(ctx, out, onceResult{value: p, ttl: ttl})
			case <-ctx.Done():
				return
			}
		}
	}()

	return out
}
