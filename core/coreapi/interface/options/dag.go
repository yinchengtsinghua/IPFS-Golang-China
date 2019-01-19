
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

import (
	"math"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
)

type DagPutSettings struct {
	InputEnc string
	Codec    uint64
	MhType   uint64
	MhLength int
}

type DagTreeSettings struct {
	Depth int
}

type DagPutOption func(*DagPutSettings) error
type DagTreeOption func(*DagTreeSettings) error

func DagPutOptions(opts ...DagPutOption) (*DagPutSettings, error) {
	options := &DagPutSettings{
		InputEnc: "json",
		Codec:    cid.DagCBOR,
		MhType:   math.MaxUint64,
		MhLength: -1,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

func DagTreeOptions(opts ...DagTreeOption) (*DagTreeSettings, error) {
	options := &DagTreeSettings{
		Depth: -1,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

type dagOpts struct{}

var Dag dagOpts

//inputenc是dag.put的一个选项，它指定
//数据。默认为“json”，大多数格式/编解码器支持“raw”
func (dagOpts) InputEnc(enc string) DagPutOption {
	return func(settings *DagPutSettings) error {
		settings.InputEnc = enc
		return nil
	}
}

//codec是dag.put的一个选项，它指定要使用的multicodec
//序列化对象。默认为cid.dagcbor（0x71）
func (dagOpts) Codec(codec uint64) DagPutOption {
	return func(settings *DagPutSettings) error {
		settings.Codec = codec
		return nil
	}
}

//hash是dag.put的一个选项，它指定要使用的多哈希设置
//散列对象时。默认值基于使用的编解码器
//（对于DAGCBOR，MH.SHA2U 256（0x12））。如果mhlen设置为-1，则默认长度为
//将使用哈希
func (dagOpts) Hash(mhType uint64, mhLen int) DagPutOption {
	return func(settings *DagPutSettings) error {
		settings.MhType = mhType
		settings.MhLength = mhLen
		return nil
	}
}

//深度是dag.tree的一个选项，它指定
//返回树。默认值为-1（无深度限制）
func (dagOpts) Depth(depth int) DagTreeOption {
	return func(settings *DagTreeSettings) error {
		settings.Depth = depth
		return nil
	}
}
