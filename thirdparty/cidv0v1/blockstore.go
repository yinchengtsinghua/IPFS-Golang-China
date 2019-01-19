
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package cidv0v1

import (
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	bs "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	blocks "gx/ipfs/QmWoXtvgC8inqFkAATB7cp2Dax7XBi9VDvSg9RCCZufmRk/go-block-format"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

type blockstore struct {
	bs.Blockstore
}

func NewBlockstore(b bs.Blockstore) bs.Blockstore {
	return &blockstore{b}
}

func (b *blockstore) Has(c cid.Cid) (bool, error) {
	have, err := b.Blockstore.Has(c)
	if have || err != nil {
		return have, err
	}
	c1 := tryOtherCidVersion(c)
	if !c1.Defined() {
		return false, nil
	}
	return b.Blockstore.Has(c1)
}

func (b *blockstore) Get(c cid.Cid) (blocks.Block, error) {
	block, err := b.Blockstore.Get(c)
	if err == nil {
		return block, nil
	}
	if err != bs.ErrNotFound {
		return nil, err
	}
	c1 := tryOtherCidVersion(c)
	if !c1.Defined() {
		return nil, bs.ErrNotFound
	}
	block, err = b.Blockstore.Get(c1)
	if err != nil {
		return nil, err
	}
//修改块，使其具有原始CID
	block, err = blocks.NewBlockWithCid(block.RawData(), c)
	if err != nil {
		return nil, err
	}
//使用原始CID插入块以避免出现问题
//钉扎
	err = b.Blockstore.Put(block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (b *blockstore) GetSize(c cid.Cid) (int, error) {
	size, err := b.Blockstore.GetSize(c)
	if err == nil {
		return size, nil
	}
	if err != bs.ErrNotFound {
		return -1, err
	}
	c1 := tryOtherCidVersion(c)
	if !c1.Defined() {
		return -1, bs.ErrNotFound
	}
	return b.Blockstore.GetSize(c1)
}

func tryOtherCidVersion(c cid.Cid) cid.Cid {
	prefix := c.Prefix()
	if prefix.Codec != cid.DagProtobuf || prefix.MhType != mh.SHA2_256 || prefix.MhLength != 32 {
		return cid.Undef
	}
	var c1 cid.Cid
	if prefix.Version == 0 {
		c1 = cid.NewCidV1(cid.DagProtobuf, c.Hash())
	} else {
		c1 = cid.NewCidV0(c.Hash())
	}
	return c1
}
