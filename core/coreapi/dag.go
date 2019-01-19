
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
	"io"
	"sync"

	gopath "path"

	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	caopts "github.com/ipfs/go-ipfs/core/coreapi/interface/options"
	coredag "github.com/ipfs/go-ipfs/core/coredag"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
)

type DagAPI CoreAPI

type dagBatch struct {
	api   *DagAPI
	toPut []ipld.Node

	lk sync.Mutex
}

//使用指定的格式和输入编码放置插入数据。除非用于
//“withcodes”或“withhash”，使用默认值“dag cbor”和“sha256”。
//返回插入数据的路径。
func (api *DagAPI) Put(ctx context.Context, src io.Reader, opts ...caopts.DagPutOption) (coreiface.ResolvedPath, error) {
	nd, err := getNode(src, opts...)
	if err != nil {
		return nil, err
	}

	err = api.dag.Add(ctx, nd)
	if err != nil {
		return nil, err
	}

	return coreiface.IpldPath(nd.Cid()), nil
}

//get使用unixfs resolver解析“path”，返回解析的节点。
func (api *DagAPI) Get(ctx context.Context, path coreiface.Path) (ipld.Node, error) {
	return api.core().ResolveNode(ctx, path)
}

//树返回由路径“p”指定的节点内的路径列表。
func (api *DagAPI) Tree(ctx context.Context, p coreiface.Path, opts ...caopts.DagTreeOption) ([]coreiface.Path, error) {
	settings, err := caopts.DagTreeOptions(opts...)
	if err != nil {
		return nil, err
	}

	n, err := api.Get(ctx, p)
	if err != nil {
		return nil, err
	}
	paths := n.Tree("", settings.Depth)
	out := make([]coreiface.Path, len(paths))
	for n, p2 := range paths {
		out[n], err = coreiface.ParsePath(gopath.Join(p.String(), p2))
		if err != nil {
			return nil, err
		}
	}

	return out, nil
}

//批量创建新的dagbatch
func (api *DagAPI) Batch(ctx context.Context) coreiface.DagBatch {
	return &dagBatch{api: api}
}

//使用指定的格式和输入编码放置插入数据。除非用于
//“withcodes”或“withhash”，使用默认值“dag cbor”和“sha256”。
//返回插入数据的路径。
func (b *dagBatch) Put(ctx context.Context, src io.Reader, opts ...caopts.DagPutOption) (coreiface.ResolvedPath, error) {
	nd, err := getNode(src, opts...)
	if err != nil {
		return nil, err
	}

	b.lk.Lock()
	b.toPut = append(b.toPut, nd)
	b.lk.Unlock()

	return coreiface.IpldPath(nd.Cid()), nil
}

//提交将节点提交到数据存储并向网络公布它们
func (b *dagBatch) Commit(ctx context.Context) error {
	b.lk.Lock()
	defer b.lk.Unlock()
	defer func() {
		b.toPut = nil
	}()

	return b.api.dag.AddMany(ctx, b.toPut)
}

func getNode(src io.Reader, opts ...caopts.DagPutOption) (ipld.Node, error) {
	settings, err := caopts.DagPutOptions(opts...)
	if err != nil {
		return nil, err
	}

	codec, ok := cid.CodecToStr[settings.Codec]
	if !ok {
		return nil, fmt.Errorf("invalid codec %d", settings.Codec)
	}

	nds, err := coredag.ParseInputs(settings.InputEnc, codec, src, settings.MhType, settings.MhLength)
	if err != nil {
		return nil, err
	}
	if len(nds) == 0 {
		return nil, fmt.Errorf("no node returned from ParseInputs")
	}
	if len(nds) != 1 {
		return nil, fmt.Errorf("got more that one node from ParseInputs")
	}

	return nds[0], nil
}

func (api *DagAPI) core() coreiface.CoreAPI {
	return (*CoreAPI)(api)
}
