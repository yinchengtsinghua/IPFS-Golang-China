
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package pin

import (
	"context"
	"testing"
	"time"

	mdag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	bs "gx/ipfs/QmYPZzd9VqmJDwxUnThfeSbV1Y5o53aVPDijTB7j7rS9Ep/go-blockservice"

	util "gx/ipfs/QmNohiVssaPw3KVLZik59DBVGTSm2dGvYT9eoXt5DQ36Yz/go-ipfs-util"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	blockstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	offline "gx/ipfs/QmYZwey1thDTynSrvd6qQkX24UpTka6TFhQ2v569UpoqxD/go-ipfs-exchange-offline"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
	dssync "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/sync"
)

var rand = util.NewTimeSeededRand()

func randNode() (*mdag.ProtoNode, cid.Cid) {
	nd := new(mdag.ProtoNode)
	nd.SetData(make([]byte, 32))
	rand.Read(nd.Data())
	k := nd.Cid()
	return nd, k
}

func assertPinned(t *testing.T, p Pinner, c cid.Cid, failmsg string) {
	_, pinned, err := p.IsPinned(c)
	if err != nil {
		t.Fatal(err)
	}

	if !pinned {
		t.Fatal(failmsg)
	}
}

func assertUnpinned(t *testing.T, p Pinner, c cid.Cid, failmsg string) {
	_, pinned, err := p.IsPinned(c)
	if err != nil {
		t.Fatal(err)
	}

	if pinned {
		t.Fatal(failmsg)
	}
}

func TestPinnerBasic(t *testing.T) {
	ctx := context.Background()

	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))

	dserv := mdag.NewDAGService(bserv)

//TODO Pinner是否需要与BlockService共享数据存储？
	p := NewPinner(dstore, dserv, dserv)

	a, ak := randNode()
	err := dserv.Add(ctx, a)
	if err != nil {
		t.Fatal(err)
	}

//引脚A {}
	err = p.Pin(ctx, a, false)
	if err != nil {
		t.Fatal(err)
	}

	assertPinned(t, p, ak, "Failed to find key")

//创建新节点c，通过b间接固定
	c, _ := randNode()
	err = dserv.Add(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	ck := c.Cid()

//创建新节点B，作为A和C的父节点
	b, _ := randNode()
	err = b.AddNodeLink("child", a)
	if err != nil {
		t.Fatal(err)
	}

	err = b.AddNodeLink("otherchild", c)
	if err != nil {
		t.Fatal(err)
	}

	err = dserv.Add(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	bk := b.Cid()

//递归插脚B A，C
	err = p.Pin(ctx, b, true)
	if err != nil {
		t.Fatal(err)
	}

	assertPinned(t, p, ck, "child of recursively pinned node not found")

	assertPinned(t, p, bk, "Recursively pinned node not found..")

	d, _ := randNode()
	d.AddNodeLink("a", a)
	d.AddNodeLink("c", c)

	e, _ := randNode()
	d.AddNodeLink("e", e)

//必须在dagserv中才能运行unpin
	err = dserv.Add(ctx, e)
	if err != nil {
		t.Fatal(err)
	}
	err = dserv.Add(ctx, d)
	if err != nil {
		t.Fatal(err)
	}

//添加d {a，c，e}
	err = p.Pin(ctx, d, true)
	if err != nil {
		t.Fatal(err)
	}

	dk := d.Cid()
	assertPinned(t, p, dk, "pinned node not found.")

//测试递归取消固定
	err = p.Unpin(ctx, dk, true)
	if err != nil {
		t.Fatal(err)
	}

	err = p.Flush()
	if err != nil {
		t.Fatal(err)
	}

	np, err := LoadPinner(dstore, dserv, dserv)
	if err != nil {
		t.Fatal(err)
	}

//直接固定测试
	assertPinned(t, np, ak, "Could not find pinned node!")

//递归固定测试
	assertPinned(t, np, bk, "could not find recursively pinned node")
}

func TestIsPinnedLookup(t *testing.T) {
//我们将测试查找是否在共享的pin中工作
//同样的分支。为此，我们将构建这棵树：
//
//a5->a4->a3->a2->a1->a0
/// /
//B-------/
//\/
//C-----------
//
//我们将确保ispinned在所有对象
//是固定的，一旦松开。
	aBranchLen := 6
	if aBranchLen < 3 {
		t.Fatal("set aBranchLen to at least 3")
	}

	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))

	dserv := mdag.NewDAGService(bserv)

//TODO Pinner是否需要与BlockService共享数据存储？
	p := NewPinner(dstore, dserv, dserv)

	aNodes := make([]*mdag.ProtoNode, aBranchLen)
	aKeys := make([]cid.Cid, aBranchLen)
	for i := 0; i < aBranchLen; i++ {
		a, _ := randNode()
		if i >= 1 {
			err := a.AddNodeLink("child", aNodes[i-1])
			if err != nil {
				t.Fatal(err)
			}
		}

		err := dserv.Add(ctx, a)
		if err != nil {
			t.Fatal(err)
		}
//T.logf（“A[%d]是%s”，I，AK）
		aNodes[i] = a
		aKeys[i] = a.Cid()
	}

//插脚A5递归
	if err := p.Pin(ctx, aNodes[aBranchLen-1], true); err != nil {
		t.Fatal(err)
	}

//创建节点B并将A3添加为子节点
	b, _ := randNode()
	if err := b.AddNodeLink("mychild", aNodes[3]); err != nil {
		t.Fatal(err)
	}

//创建C节点
	c, _ := randNode()
//将a0添加为c的子级
	if err := c.AddNodeLink("child", aNodes[0]); err != nil {
		t.Fatal(err)
	}

//添加C
	err := dserv.Add(ctx, c)
	if err != nil {
		t.Fatal(err)
	}
	ck := c.Cid()
//t.logf（“c是%s”，ck）

//加C到B再加B
	if err := b.AddNodeLink("myotherchild", c); err != nil {
		t.Fatal(err)
	}
	err = dserv.Add(ctx, b)
	if err != nil {
		t.Fatal(err)
	}
	bk := b.Cid()
//t.logf（“b是%s”，bk）

//插脚C递归

	if err := p.Pin(ctx, c, true); err != nil {
		t.Fatal(err)
	}

//插脚B递归

	if err := p.Pin(ctx, b, true); err != nil {
		t.Fatal(err)
	}

	assertPinned(t, p, aKeys[0], "A0 should be pinned")
	assertPinned(t, p, aKeys[1], "A1 should be pinned")
	assertPinned(t, p, ck, "C should be pinned")
	assertPinned(t, p, bk, "B should be pinned")

//递归解锁A5
	if err := p.Unpin(ctx, aKeys[5], true); err != nil {
		t.Fatal(err)
	}

	assertPinned(t, p, aKeys[0], "A0 should still be pinned through B")
	assertUnpinned(t, p, aKeys[4], "A4 should be unpinned")

//递归地取消固定B
	if err := p.Unpin(ctx, bk, true); err != nil {
		t.Fatal(err)
	}
	assertUnpinned(t, p, bk, "B should be unpinned")
	assertUnpinned(t, p, aKeys[1], "A1 should be unpinned")
	assertPinned(t, p, aKeys[0], "A0 should still be pinned through C")
}

func TestDuplicateSemantics(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))

	dserv := mdag.NewDAGService(bserv)

//TODO Pinner是否需要与BlockService共享数据存储？
	p := NewPinner(dstore, dserv, dserv)

	a, _ := randNode()
	err := dserv.Add(ctx, a)
	if err != nil {
		t.Fatal(err)
	}

//
	err = p.Pin(ctx, a, true)
	if err != nil {
		t.Fatal(err)
	}

//直接固定应失败
	err = p.Pin(ctx, a, false)
	if err == nil {
		t.Fatal("expected direct pin to fail")
	}

//再次递归固定应该成功
	err = p.Pin(ctx, a, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFlush(t *testing.T) {
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))

	dserv := mdag.NewDAGService(bserv)
	p := NewPinner(dstore, dserv, dserv)
	_, k := randNode()

	p.PinWithMode(k, Recursive)
	if err := p.Flush(); err != nil {
		t.Fatal(err)
	}
	assertPinned(t, p, k, "expected key to still be pinned")
}

func TestPinRecursiveFail(t *testing.T) {
	ctx := context.Background()
	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))
	dserv := mdag.NewDAGService(bserv)

	p := NewPinner(dstore, dserv, dserv)

	a, _ := randNode()
	b, _ := randNode()
	err := a.AddNodeLink("child", b)
	if err != nil {
		t.Fatal(err)
	}

//注意：这不是基于时间的测试，我们希望pin失败
	mctx, cancel := context.WithTimeout(ctx, time.Millisecond)
	defer cancel()

	err = p.Pin(mctx, a, true)
	if err == nil {
		t.Fatal("should have failed to pin here")
	}

	err = dserv.Add(ctx, b)
	if err != nil {
		t.Fatal(err)
	}

	err = dserv.Add(ctx, a)
	if err != nil {
		t.Fatal(err)
	}

//这是基于时间的…但不应该引起任何问题
	mctx, cancel = context.WithTimeout(ctx, time.Second)
	defer cancel()
	err = p.Pin(mctx, a, true)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPinUpdate(t *testing.T) {
	ctx := context.Background()

	dstore := dssync.MutexWrap(ds.NewMapDatastore())
	bstore := blockstore.NewBlockstore(dstore)
	bserv := bs.New(bstore, offline.Exchange(bstore))

	dserv := mdag.NewDAGService(bserv)
	p := NewPinner(dstore, dserv, dserv)
	n1, c1 := randNode()
	n2, c2 := randNode()

	dserv.Add(ctx, n1)
	dserv.Add(ctx, n2)

	if err := p.Pin(ctx, n1, true); err != nil {
		t.Fatal(err)
	}

	if err := p.Update(ctx, c1, c2, true); err != nil {
		t.Fatal(err)
	}

	assertPinned(t, p, c2, "c2 should be pinned now")
	assertUnpinned(t, p, c1, "c1 should no longer be pinned")

	if err := p.Update(ctx, c2, c1, false); err != nil {
		t.Fatal(err)
	}

	assertPinned(t, p, c2, "c2 should be pinned still")
	assertPinned(t, p, c1, "c1 should be pinned now")
}
