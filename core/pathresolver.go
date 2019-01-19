
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package core

import (
	"context"
	"errors"
	"strings"

	namesys "github.com/ipfs/go-ipfs/namesys"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	resolver "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path/resolver"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

//errnamesys是一个显式错误，当IPFS节点没有
//（然而）有一个名字系统
var ErrNoNamesys = errors.New(
	"core/resolve: no Namesys on IpfsNode - can't resolve ipns entry")

//resolveipns解析/ipns路径
func ResolveIPNS(ctx context.Context, nsys namesys.NameSystem, p path.Path) (path.Path, error) {
	if strings.HasPrefix(p.String(), "/ipns/") {
		evt := log.EventBegin(ctx, "resolveIpnsPath")
		defer evt.Done()
//解析IPN路径

//TODO（cryptix）：我们应该能够查询本地缓存中的路径
		if nsys == nil {
			evt.Append(logging.LoggableMap{"error": ErrNoNamesys.Error()})
			return "", ErrNoNamesys
		}

		seg := p.Segments()

if len(seg) < 2 || seg[1] == "" { //只是“/<protocol/>”没有更多的段
			evt.Append(logging.LoggableMap{"error": path.ErrNoComponents.Error()})
			return "", path.ErrNoComponents
		}

		extensions := seg[2:]
		resolvable, err := path.FromSegments("/", seg[0], seg[1])
		if err != nil {
			evt.Append(logging.LoggableMap{"error": err.Error()})
			return "", err
		}

		respath, err := nsys.Resolve(ctx, resolvable.String())
		if err != nil {
			evt.Append(logging.LoggableMap{"error": err.Error()})
			return "", err
		}

		segments := append(respath.Segments(), extensions...)
		p, err = path.FromSegments("/", segments...)
		if err != nil {
			evt.Append(logging.LoggableMap{"error": err.Error()})
			return "", err
		}
	}
	return p, nil
}

//解析通过解析出特定于协议的路径来解析给定路径
//条目（例如/ipns/<node key>）然后通过/ipfs/
//
func Resolve(ctx context.Context, nsys namesys.NameSystem, r *resolver.Resolver, p path.Path) (ipld.Node, error) {
	p, err := ResolveIPNS(ctx, nsys, p)
	if err != nil {
		return nil, err
	}

//好的，我们现在有一个IPF路径（或者我们将作为一个路径处理的路径）
	return r.ResolvePath(ctx, p)
}
