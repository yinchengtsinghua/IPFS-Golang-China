
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

	mdag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
)

//differNumerate获取图表中由“to”指向的每个对象，即
//不在“从”。如果可以的话，这可以用来更有效地获取图形
//保证您已经拥有“发件人”的全部内容
func DiffEnumerate(ctx context.Context, dserv ipld.NodeGetter, from, to cid.Cid) error {
	fnd, err := dserv.Get(ctx, from)
	if err != nil {
		return fmt.Errorf("get %s: %s", from, err)
	}

	tnd, err := dserv.Get(ctx, to)
	if err != nil {
		return fmt.Errorf("get %s: %s", to, err)
	}

	diff := getLinkDiff(fnd, tnd)

	sset := cid.NewSet()
	for _, c := range diff {
//既然我们已经假设“从”图中包含了所有内容，
//将所有这些CID添加到我们的“已看到”集合中，以避免潜在的
//稍后枚举它们
		if c.bef.Defined() {
			sset.Add(c.bef)
		}
	}
	for _, c := range diff {
		if !c.bef.Defined() {
			if sset.Has(c.aft) {
				continue
			}
			err := mdag.EnumerateChildrenAsync(ctx, mdag.GetLinksDirect(dserv), c.aft, sset.Visit)
			if err != nil {
				return err
			}
		} else {
			err := DiffEnumerate(ctx, dserv, c.bef, c.aft)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

//如果bef和aft不是零，则表示bef被aft替换。
//如果bef为零而aft不是，则表示aft是新添加的。
//如果aft为零，bef不为零，则表示bef已被删除。
type diffpair struct {
	bef, aft cid.Cid
}

//GetLinkDiff返回节点“a”和“b”之间的变更集。目前确实
//不记录删除，因为我们的用例不需要这样做。
func getLinkDiff(a, b ipld.Node) []diffpair {
	ina := make(map[string]*ipld.Link)
	inb := make(map[string]*ipld.Link)
	var aonly []cid.Cid
	for _, l := range b.Links() {
		inb[l.Cid.KeyString()] = l
	}
	for _, l := range a.Links() {
		var key = l.Cid.KeyString()
		ina[key] = l
		if inb[key] == nil {
			aonly = append(aonly, l.Cid)
		}
	}

	var out []diffpair
	var aindex int

	for _, l := range b.Links() {
		if ina[l.Cid.KeyString()] != nil {
			continue
		}

		if aindex < len(aonly) {
			out = append(out, diffpair{bef: aonly[aindex], aft: l.Cid})
			aindex++
		} else {
			out = append(out, diffpair{aft: l.Cid})
			continue
		}
	}
	return out
}
