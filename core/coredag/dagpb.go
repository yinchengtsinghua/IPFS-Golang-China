
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package coredag

import (
	"io"
	"io/ioutil"
	"math"

	"gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

func dagpbJSONParser(r io.Reader, mhType uint64, mhLen int) ([]ipld.Node, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	nd := &merkledag.ProtoNode{}

	err = nd.UnmarshalJSON(data)
	if err != nil {
		return nil, err
	}

	nd.SetCidBuilder(cidPrefix(mhType, mhLen))

	return []ipld.Node{nd}, nil
}

func dagpbRawParser(r io.Reader, mhType uint64, mhLen int) ([]ipld.Node, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	nd, err := merkledag.DecodeProtobuf(data)
	if err != nil {
		return nil, err
	}

	nd.SetCidBuilder(cidPrefix(mhType, mhLen))

	return []ipld.Node{nd}, nil
}

func cidPrefix(mhType uint64, mhLen int) *cid.Prefix {
	if mhType == math.MaxUint64 {
		mhType = mh.SHA2_256
	}

	prefix := &cid.Prefix{
		MhType:   mhType,
		MhLength: mhLen,
		Version:  1,
		Codec:    cid.DagProtobuf,
	}

	if mhType == mh.SHA2_256 {
		prefix.Version = 0
	}

	return prefix
}
