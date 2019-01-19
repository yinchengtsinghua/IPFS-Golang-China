
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
	"errors"
	"net"
	"strings"

	opts "github.com/ipfs/go-ipfs/namesys/opts"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	isd "gx/ipfs/QmZmmuAXgX73UQmX1jRKjTGmjzq24Jinqkq8vzkBtno4uX/go-is-domain"
)

type LookupTXTFunc func(name string) (txt []string, err error)

//dns resolver在DNS域上实现解析程序
type DNSResolver struct {
	lookupTXT LookupTXTFunc
//托多：也许是某种缓存？
//缓存需要超时
}

//newdnsresolver使用DNS TXT记录构造名称冲突解决程序。
func NewDNSResolver() *DNSResolver {
	return &DNSResolver{lookupTXT: net.LookupTXT}
}

//解析实现解析程序。
func (r *DNSResolver) Resolve(ctx context.Context, name string, options ...opts.ResolveOpt) (path.Path, error) {
	return resolve(ctx, r, name, opts.ProcessOpts(options))
}

//resolveasync实现resolver。
func (r *DNSResolver) ResolveAsync(ctx context.Context, name string, options ...opts.ResolveOpt) <-chan Result {
	return resolveAsync(ctx, r, name, opts.ProcessOpts(options))
}

type lookupRes struct {
	path  path.Path
	error error
}

//resolveonce实现resolver。
//给定域名的txt记录应包含b58
//编码的多哈希。
func (r *DNSResolver) resolveOnceAsync(ctx context.Context, name string, options opts.ResolveOpts) <-chan onceResult {
	out := make(chan onceResult, 1)
	segments := strings.SplitN(name, "/", 2)
	domain := segments[0]

	if !isd.IsDomain(domain) {
		out <- onceResult{err: errors.New("not a valid domain name")}
		close(out)
		return out
	}
	log.Debugf("DNSResolver resolving %s", domain)

	rootChan := make(chan lookupRes, 1)
	go workDomain(r, domain, rootChan)

	subChan := make(chan lookupRes, 1)
	go workDomain(r, "_dnslink."+domain, subChan)

	appendPath := func(p path.Path) (path.Path, error) {
		if len(segments) > 1 {
			return path.FromSegments("", strings.TrimRight(p.String(), "/"), segments[1])
		}
		return p, nil
	}

	go func() {
		defer close(out)
		for {
			select {
			case subRes, ok := <-subChan:
				if !ok {
					subChan = nil
					break
				}
				if subRes.error == nil {
					p, err := appendPath(subRes.path)
					emitOnceResult(ctx, out, onceResult{value: p, err: err})
					return
				}
			case rootRes, ok := <-rootChan:
				if !ok {
					rootChan = nil
					break
				}
				if rootRes.error == nil {
					p, err := appendPath(rootRes.path)
					emitOnceResult(ctx, out, onceResult{value: p, err: err})
				}
			case <-ctx.Done():
				return
			}
			if subChan == nil && rootChan == nil {
				return
			}
		}
	}()

	return out
}

func workDomain(r *DNSResolver, name string, res chan lookupRes) {
	defer close(res)

	txt, err := r.lookupTXT(name)
	if err != nil {
//错误是！=零
		res <- lookupRes{"", err}
		return
	}

	for _, t := range txt {
		p, err := parseEntry(t)
		if err == nil {
			res <- lookupRes{p, nil}
			return
		}
	}
	res <- lookupRes{"", ErrResolveFailed}
}

func parseEntry(txt string) (path.Path, error) {
p, err := path.ParseCidToPath(txt) //裸IPF多哈希
	if err == nil {
		return p, nil
	}

	return tryParseDnsLink(txt)
}

func tryParseDnsLink(txt string) (path.Path, error) {
	parts := strings.SplitN(txt, "=", 2)
	if len(parts) == 2 && parts[0] == "dnslink" {
		return path.ParsePath(parts[1])
	}

	return "", errors.New("not a valid dnslink entry")
}
