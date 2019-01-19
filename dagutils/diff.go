
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
	"fmt"
	"path"

	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"

	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
)

//这些常量定义了可以应用于DAG的更改。
const (
	Add = iota
	Remove
	Mod
)

//更改表示对DAG的更改，并包含对旧和
//新的CID。
type Change struct {
	Type   coreiface.ChangeType
	Path   string
	Before cid.Cid
	After  cid.Cid
}

//字符串打印了一条关于更改的人性化行。
func (c *Change) String() string {
	switch c.Type {
	case Add:
		return fmt.Sprintf("Added %s at %s", c.After.String(), c.Path)
	case Remove:
		return fmt.Sprintf("Removed %s from %s", c.Before.String(), c.Path)
	case Mod:
		return fmt.Sprintf("Changed %s to %s at %s", c.Before.String(), c.After.String(), c.Path)
	default:
		panic("nope")
	}
}

//ApplyChange将请求的更改应用于给定DAG中的给定节点。
func ApplyChange(ctx context.Context, ds ipld.DAGService, nd *dag.ProtoNode, cs []*Change) (*dag.ProtoNode, error) {
	e := NewDagEditor(nd, ds)
	for _, c := range cs {
		switch c.Type {
		case Add:
			child, err := ds.Get(ctx, c.After)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*dag.ProtoNode)
			if !ok {
				return nil, dag.ErrNotProtobuf
			}

			err = e.InsertNodeAtPath(ctx, c.Path, childpb, nil)
			if err != nil {
				return nil, err
			}

		case Remove:
			err := e.RmLink(ctx, c.Path)
			if err != nil {
				return nil, err
			}

		case Mod:
			err := e.RmLink(ctx, c.Path)
			if err != nil {
				return nil, err
			}
			child, err := ds.Get(ctx, c.After)
			if err != nil {
				return nil, err
			}

			childpb, ok := child.(*dag.ProtoNode)
			if !ok {
				return nil, dag.ErrNotProtobuf
			}

			err = e.InsertNodeAtPath(ctx, c.Path, childpb, nil)
			if err != nil {
				return nil, err
			}
		}
	}

	return e.Finalize(ctx, ds)
}

//diff返回一组将节点“a”转换为节点“b”的更改。
//它仅在以下情况下遍历链接：
//1。两个节点的链接数大于0。
//2。两个节点都是原节点。
//否则，它会比较cid并发出mod change对象。
func Diff(ctx context.Context, ds ipld.DAGService, a, b ipld.Node) ([]*Change, error) {
//两个节点都是叶的基本情况，只需比较
//他们的CIID。
	if len(a.Links()) == 0 && len(b.Links()) == 0 {
		return getChange(a, b)
	}

	var out []*Change
	cleanA, okA := a.Copy().(*dag.ProtoNode)
	cleanB, okB := b.Copy().(*dag.ProtoNode)
	if !okA || !okB {
		return getChange(a, b)
	}

//去掉不变的东西
	for _, lnk := range a.Links() {
		l, _, err := b.ResolveLink([]string{lnk.Name})
		if err == nil {
			if l.Cid.Equals(lnk.Cid) {
//没有变化…忽略它
			} else {
				anode, err := lnk.GetNode(ctx, ds)
				if err != nil {
					return nil, err
				}

				bnode, err := l.GetNode(ctx, ds)
				if err != nil {
					return nil, err
				}

				sub, err := Diff(ctx, ds, anode, bnode)
				if err != nil {
					return nil, err
				}

				for _, subc := range sub {
					subc.Path = path.Join(lnk.Name, subc.Path)
					out = append(out, subc)
				}
			}
			cleanA.RemoveNodeLink(l.Name)
			cleanB.RemoveNodeLink(l.Name)
		}
	}

	for _, lnk := range cleanA.Links() {
		out = append(out, &Change{
			Type:   Remove,
			Path:   lnk.Name,
			Before: lnk.Cid,
		})
	}
	for _, lnk := range cleanB.Links() {
		out = append(out, &Change{
			Type:  Add,
			Path:  lnk.Name,
			After: lnk.Cid,
		})
	}

	return out, nil
}

//冲突表示两个不兼容的更改，由mergediff（）返回。
type Conflict struct {
	A *Change
	B *Change
}

//MergeDiff接受两个更改切片，并将它们添加到单个切片中。
//当从B变更到A中现有变更的相同路径时，
//将创建一个冲突，B不会添加到合并切片中。
//返回冲突切片并包含指向
//涉及的更改（共享同一路径）。
func MergeDiffs(a, b []*Change) ([]*Change, []Conflict) {
	var out []*Change
	var conflicts []Conflict
	paths := make(map[string]*Change)
	for _, c := range a {
		paths[c.Path] = c
	}

	for _, c := range b {
		if ca, ok := paths[c.Path]; ok {
			conflicts = append(conflicts, Conflict{
				A: ca,
				B: c,
			})
		} else {
			out = append(out, c)
		}
	}
	for _, c := range paths {
		out = append(out, c)
	}
	return out, conflicts
}

func getChange(a, b ipld.Node) ([]*Change, error) {
	if a.Cid().Equals(b.Cid()) {
		return []*Change{}, nil
	}
	return []*Change{
		{
			Type:   Mod,
			Before: a.Cid(),
			After:  b.Cid(),
		},
	}, nil
}
