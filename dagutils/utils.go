
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package dagutils

import (
	"context"
	"errors"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	bserv "gx/ipfs/QmYPZzd9VqmJDwxUnThfeSbV1Y5o53aVPDijTB7j7rS9Ep/go-blockservice"

	bstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	offline "gx/ipfs/QmYZwey1thDTynSrvd6qQkX24UpTka6TFhQ2v569UpoqxD/go-ipfs-exchange-offline"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
	syncds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/sync"
)

//编辑器表示协议节点树编辑器，并提供
//修改它。
type Editor struct {
	root *dag.ProtoNode

//tmp是所有
//要存储的中间节点
	tmp ipld.DAGService

//SRC是包含*全部*数据的数据存储，用于
//用于修改的节点（nil是有效值）
	src ipld.DAGService
}

//NewMemoryDagService返回一个新的线程安全内存中的DagService。
func NewMemoryDagService() ipld.DAGService {
//为编辑器的中间节点构建MEM数据存储
	bs := bstore.NewBlockstore(syncds.MutexWrap(ds.NewMapDatastore()))
	bsrv := bserv.New(bs, offline.Exchange(bs))
	return dag.NewDAGService(bsrv)
}

//NewDagEditor返回一个ProtoNode编辑器。
//
//*根是要修改的节点
//*source是从中提取节点的数据存储（可选）
func NewDagEditor(root *dag.ProtoNode, source ipld.DAGService) *Editor {
	return &Editor{
		root: root,
		tmp:  NewMemoryDagService(),
		src:  source,
	}
}

//getnode返回正在编辑的根节点的副本。
func (e *Editor) GetNode() *dag.ProtoNode {
	return e.root.Copy().(*dag.ProtoNode)
}

//GetDagService返回此编辑器使用的DagService。
func (e *Editor) GetDagService() ipld.DAGService {
	return e.tmp
}

func addLink(ctx context.Context, ds ipld.DAGService, root *dag.ProtoNode, childname string, childnd ipld.Node) (*dag.ProtoNode, error) {
	if childname == "" {
		return nil, errors.New("cannot create link with no name")
	}

//确保我们正在添加的节点在DAGService中
	err := ds.Add(ctx, childnd)
	if err != nil {
		return nil, err
	}

	_ = ds.Remove(ctx, root.Cid())

//确保不存在具有该名称的链接
_ = root.RemoveNodeLink(childname) //忽略错误，只有选项是errnotfound

	if err := root.AddNodeLink(childname, childnd); err != nil {
		return nil, err
	}

	if err := ds.Add(ctx, root); err != nil {
		return nil, err
	}
	return root, nil
}

//insertnodeatpath在树中插入一个新节点，并将当前根目录替换为新根目录。
func (e *Editor) InsertNodeAtPath(ctx context.Context, pth string, toinsert ipld.Node, create func() *dag.ProtoNode) error {
	splpath := path.SplitList(pth)
	nd, err := e.insertNodeAtPath(ctx, e.root, splpath, toinsert, create)
	if err != nil {
		return err
	}
	e.root = nd
	return nil
}

func (e *Editor) insertNodeAtPath(ctx context.Context, root *dag.ProtoNode, path []string, toinsert ipld.Node, create func() *dag.ProtoNode) (*dag.ProtoNode, error) {
	if len(path) == 1 {
		return addLink(ctx, e.tmp, root, path[0], toinsert)
	}

	nd, err := root.GetLinkedProtoNode(ctx, e.tmp, path[0])
	if err != nil {
//如果“create”为真，我们将根据需要在向下的路上创建目录。
		if err == dag.ErrLinkNotFound && create != nil {
			nd = create()
err = nil //不再是错误案例
		} else if err == ipld.ErrNotFound {
//试着在我们的源数据库中找到它
			nd, err = root.GetLinkedProtoNode(ctx, e.src, path[0])
		}

//如果我们收到errnotfound，则第二个“getLinkedNode”调用
//也失败了，我们想出错
		if err != nil {
			return nil, err
		}
	}

	ndprime, err := e.insertNodeAtPath(ctx, nd, path[1:], toinsert, create)
	if err != nil {
		return nil, err
	}

	_ = e.tmp.Remove(ctx, root.Cid())

	_ = root.RemoveNodeLink(path[0])
	err = root.AddNodeLink(path[0], ndprime)
	if err != nil {
		return nil, err
	}

	err = e.tmp.Add(ctx, root)
	if err != nil {
		return nil, err
	}

	return root, nil
}

//rmlink删除具有给定名称的链接并更新的根节点
//编辑。
func (e *Editor) RmLink(ctx context.Context, pth string) error {
	splpath := path.SplitList(pth)
	nd, err := e.rmLink(ctx, e.root, splpath)
	if err != nil {
		return err
	}
	e.root = nd
	return nil
}

func (e *Editor) rmLink(ctx context.Context, root *dag.ProtoNode, path []string) (*dag.ProtoNode, error) {
	if len(path) == 1 {
//基本情况，删除有问题的节点
		err := root.RemoveNodeLink(path[0])
		if err != nil {
			return nil, err
		}

		err = e.tmp.Add(ctx, root)
		if err != nil {
			return nil, err
		}

		return root, nil
	}

//在tmp dagstore和source dagstore中搜索节点
	nd, err := root.GetLinkedProtoNode(ctx, e.tmp, path[0])
	if err == ipld.ErrNotFound {
		nd, err = root.GetLinkedProtoNode(ctx, e.src, path[0])
	}

	if err != nil {
		return nil, err
	}

	nnode, err := e.rmLink(ctx, nd, path[1:])
	if err != nil {
		return nil, err
	}

	e.tmp.Remove(ctx, root.Cid())

	_ = root.RemoveNodeLink(path[0])
	err = root.AddNodeLink(path[0], nnode)
	if err != nil {
		return nil, err
	}

	err = e.tmp.Add(ctx, root)
	if err != nil {
		return nil, err
	}

	return root, nil
}

//Finalize将新DAG写入给定的DAG服务并返回修改后的
//根节点。
func (e *Editor) Finalize(ctx context.Context, ds ipld.DAGService) (*dag.ProtoNode, error) {
	nd := e.GetNode()
	err := copyDag(ctx, nd, e.tmp, ds)
	return nd, err
}

func copyDag(ctx context.Context, nd ipld.Node, from, to ipld.DAGService) error {
//TODO（4609）：制造这批。
	err := to.Add(ctx, nd)
	if err != nil {
		return err
	}

	for _, lnk := range nd.Links() {
		child, err := lnk.GetNode(ctx, from)
		if err != nil {
			if err == ipld.ErrNotFound {
//找不到意味着我们没有修改它，它应该
//已经在目标数据存储中
				continue
			}
			return err
		}

		err = copyDag(ctx, child, from, to)
		if err != nil {
			return err
		}
	}
	return nil
}
