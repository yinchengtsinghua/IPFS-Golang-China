
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

import (
	"fmt"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

type BlockPutSettings struct {
	Codec    string
	MhType   uint64
	MhLength int
}

type BlockRmSettings struct {
	Force bool
}

type BlockPutOption func(*BlockPutSettings) error
type BlockRmOption func(*BlockRmSettings) error

func BlockPutOptions(opts ...BlockPutOption) (*BlockPutSettings, cid.Prefix, error) {
	options := &BlockPutSettings{
		Codec:    "",
		MhType:   mh.SHA2_256,
		MhLength: -1,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, cid.Prefix{}, err
		}
	}

	var pref cid.Prefix
	pref.Version = 1

	if options.Codec == "" {
		if options.MhType != mh.SHA2_256 || (options.MhLength != -1 && options.MhLength != 32) {
			options.Codec = "protobuf"
		} else {
			options.Codec = "v0"
		}
	}

	if options.Codec == "v0" && options.MhType == mh.SHA2_256 {
		pref.Version = 0
	}

	formatval, ok := cid.Codecs[options.Codec]
	if !ok {
		return nil, cid.Prefix{}, fmt.Errorf("unrecognized format: %s", options.Codec)
	}

	if options.Codec == "v0" {
		if options.MhType != mh.SHA2_256 || (options.MhLength != -1 && options.MhLength != 32) {
			return nil, cid.Prefix{}, fmt.Errorf("only sha2-255-32 is allowed with CIDv0")
		}
	}

	pref.Codec = formatval

	pref.MhType = options.MhType
	pref.MhLength = options.MhLength

	return options, pref, nil
}

func BlockRmOptions(opts ...BlockRmOption) (*BlockRmSettings, error) {
	options := &BlockRmSettings{
		Force: false,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

type blockOpts struct{}

var Block blockOpts

//FORMAT是block.put的一个选项，它指定要使用的multicodec
//序列化对象。默认为“V0”
func (blockOpts) Format(codec string) BlockPutOption {
	return func(settings *BlockPutSettings) error {
		settings.Codec = codec
		return nil
	}
}

//哈希是块的选项。Put指定要使用的多哈希设置
//散列对象时。默认值为mh.sha2_256（0x12）。
//如果mhlen设置为-1，则将使用哈希的默认长度。
func (blockOpts) Hash(mhType uint64, mhLen int) BlockPutOption {
	return func(settings *BlockPutSettings) error {
		settings.MhType = mhType
		settings.MhLength = mhLen
		return nil
	}
}

//force是block.rm的一个选项，当设置为true时，它将忽略
//不存在的块
func (blockOpts) Force(force bool) BlockRmOption {
	return func(settings *BlockRmSettings) error {
		settings.Force = force
		return nil
	}
}
