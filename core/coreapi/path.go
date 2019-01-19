
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package coreapi

import (
	"context"
	"fmt"
	gopath "path"

	"github.com/ipfs/go-ipfs/core"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"

	ipfspath "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	"gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path/resolver"
	uio "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/io"
	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
)

//resolvenode使用unixfs resolver解析路径“p”，获取并返回
//解析节点。
func (api *CoreAPI) ResolveNode(ctx context.Context, p coreiface.Path) (ipld.Node, error) {
	rp, err := api.ResolvePath(ctx, p)
	if err != nil {
		return nil, err
	}

	node, err := api.dag.Get(ctx, rp.Cid())
	if err != nil {
		return nil, err
	}
	return node, nil
}

//resolvepath使用unixfs resolver解析路径“p”，返回
//解决路径。
func (api *CoreAPI) ResolvePath(ctx context.Context, p coreiface.Path) (coreiface.ResolvedPath, error) {
	if _, ok := p.(coreiface.ResolvedPath); ok {
		return p.(coreiface.ResolvedPath), nil
	}

	ipath := ipfspath.Path(p.String())
	ipath, err := core.ResolveIPNS(ctx, api.namesys, ipath)
	if err == core.ErrNoNamesys {
		return nil, coreiface.ErrOffline
	} else if err != nil {
		return nil, err
	}

	var resolveOnce resolver.ResolveOnce

	switch ipath.Segments()[0] {
	case "ipfs":
		resolveOnce = uio.ResolveUnixfsOnce
	case "ipld":
		resolveOnce = resolver.ResolveSingle
	default:
		return nil, fmt.Errorf("unsupported path namespace: %s", p.Namespace())
	}

	r := &resolver.Resolver{
		DAG:         api.dag,
		ResolveOnce: resolveOnce,
	}

	node, rest, err := r.ResolveToLastNode(ctx, ipath)
	if err != nil {
		return nil, err
	}

	root, err := cid.Parse(ipath.Segments()[1])
	if err != nil {
		return nil, err
	}

	return coreiface.NewResolvedPath(ipath, node, root, gopath.Join(rest...)), nil
}
