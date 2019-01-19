
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package tests

import (
	"context"
	"github.com/ipfs/go-ipfs/core/coreapi/interface"
	"strings"
	"testing"

	opt "github.com/ipfs/go-ipfs/core/coreapi/interface/options"
)

func (tp *provider) TestPin(t *testing.T) {
	tp.hasApi(t, func(api iface.CoreAPI) error {
		if api.Pin() == nil {
			return apiNotImplemented
		}
		return nil
	})

	t.Run("TestPinAdd", tp.TestPinAdd)
	t.Run("TestPinSimple", tp.TestPinSimple)
	t.Run("TestPinRecursive", tp.TestPinRecursive)
}

func (tp *provider) TestPinAdd(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api, err := tp.makeAPI(ctx)
	if err != nil {
		t.Error(err)
	}

	p, err := api.Unixfs().Add(ctx, strFile("foo")())
	if err != nil {
		t.Error(err)
	}

	err = api.Pin().Add(ctx, p)
	if err != nil {
		t.Error(err)
	}
}

func (tp *provider) TestPinSimple(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api, err := tp.makeAPI(ctx)
	if err != nil {
		t.Error(err)
	}

	p, err := api.Unixfs().Add(ctx, strFile("foo")())
	if err != nil {
		t.Error(err)
	}

	err = api.Pin().Add(ctx, p)
	if err != nil {
		t.Error(err)
	}

	list, err := api.Pin().Ls(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 1 {
		t.Errorf("unexpected pin list len: %d", len(list))
	}

	if list[0].Path().Cid().String() != p.Cid().String() {
		t.Error("paths don't match")
	}

	if list[0].Type() != "recursive" {
		t.Error("unexpected pin type")
	}

	err = api.Pin().Rm(ctx, p)
	if err != nil {
		t.Fatal(err)
	}

	list, err = api.Pin().Ls(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 0 {
		t.Errorf("unexpected pin list len: %d", len(list))
	}
}

func (tp *provider) TestPinRecursive(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	api, err := tp.makeAPI(ctx)
	if err != nil {
		t.Error(err)
	}

	p0, err := api.Unixfs().Add(ctx, strFile("foo")())
	if err != nil {
		t.Error(err)
	}

	p1, err := api.Unixfs().Add(ctx, strFile("bar")())
	if err != nil {
		t.Error(err)
	}

	p2, err := api.Dag().Put(ctx, strings.NewReader(`{"lnk": {"/": "`+p0.Cid().String()+`"}}`))
	if err != nil {
		t.Error(err)
	}

	p3, err := api.Dag().Put(ctx, strings.NewReader(`{"lnk": {"/": "`+p1.Cid().String()+`"}}`))
	if err != nil {
		t.Error(err)
	}

	err = api.Pin().Add(ctx, p2)
	if err != nil {
		t.Error(err)
	}

	err = api.Pin().Add(ctx, p3, opt.Pin.Recursive(false))
	if err != nil {
		t.Error(err)
	}

	list, err := api.Pin().Ls(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 3 {
		t.Errorf("unexpected pin list len: %d", len(list))
	}

	list, err = api.Pin().Ls(ctx, opt.Pin.Type.Direct())
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 1 {
		t.Errorf("unexpected pin list len: %d", len(list))
	}

	if list[0].Path().String() != p3.String() {
		t.Error("unexpected path")
	}

	list, err = api.Pin().Ls(ctx, opt.Pin.Type.Recursive())
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 1 {
		t.Errorf("unexpected pin list len: %d", len(list))
	}

	if list[0].Path().String() != p2.String() {
		t.Error("unexpected path")
	}

	list, err = api.Pin().Ls(ctx, opt.Pin.Type.Indirect())
	if err != nil {
		t.Fatal(err)
	}

	if len(list) != 1 {
		t.Errorf("unexpected pin list len: %d", len(list))
	}

	if list[0].Path().Cid().String() != p0.Cid().String() {
		t.Error("unexpected path")
	}

	res, err := api.Pin().Verify(ctx)
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for r := range res {
		if !r.Ok() {
			t.Error("expected pin to be ok")
		}
		n++
	}

	if n != 1 {
		t.Errorf("unexpected verify result count: %d", n)
	}

//TODO:找出一种在不接触ipfsnode的情况下测试验证的方法
 /*
  err=api.block（）.rm（ctx，p0，opt.block.force（true））。
  如果犯错！= nIL{
   致死性（Err）
  }

  res，err=api.pin（）。验证（ctx）
  如果犯错！= nIL{
   致死性（Err）
  }
  n＝0
  对于r：=范围res
   若r o（）{
    T.错误（“预期管脚不正常”）
   }

   如果len（r.badnodes（））！= 1 {
    t.fatalf（“意外的badnodes len”）
   }

   如果r.badnodes（）[0].path（）.cid（）.string（）！=p0.cid（）.string（）
    t.error（“意外的badnode路径”）
   }

   如果r.badnodes（）[0].err（）.error（）！=“未找到Merkledag”
    t.errorf（“意外的badnode错误：%s”，r.badnode s（）[0].err（）.error（））
   }
   n++
  }

  如果是N！= 1 {
   t.errorf（“意外验证结果计数：%d”，n）
  }
 **/

}
