
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

import (
	"errors"
	"fmt"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

type Layout int

const (
	BalancedLayout Layout = iota
	TrickleLayout
)

type UnixfsAddSettings struct {
	CidVersion int
	MhType     uint64

	Inline       bool
	InlineLimit  int
	RawLeaves    bool
	RawLeavesSet bool

	Chunker string
	Layout  Layout

	Pin      bool
	OnlyHash bool
	FsCache  bool
	NoCopy   bool

	Wrap      bool
	Hidden    bool
	StdinName string

	Events   chan<- interface{}
	Silent   bool
	Progress bool
}

type UnixfsAddOption func(*UnixfsAddSettings) error

func UnixfsAddOptions(opts ...UnixfsAddOption) (*UnixfsAddSettings, cid.Prefix, error) {
	options := &UnixfsAddSettings{
		CidVersion: -1,
		MhType:     mh.SHA2_256,

		Inline:       false,
		InlineLimit:  32,
		RawLeaves:    false,
		RawLeavesSet: false,

		Chunker: "size-262144",
		Layout:  BalancedLayout,

		Pin:      false,
		OnlyHash: false,
		FsCache:  false,
		NoCopy:   false,

		Wrap:      false,
		Hidden:    false,
		StdinName: "",

		Events:   nil,
		Silent:   false,
		Progress: false,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, cid.Prefix{}, err
		}
	}

//无复制->rawblock
	if options.NoCopy && !options.RawLeaves {
//固定的？
		if options.RawLeavesSet {
			return nil, cid.Prefix{}, fmt.Errorf("nocopy option requires '--raw-leaves' to be enabled as well")
		}

//不，满足强制约束。
		options.RawLeaves = true
	}

//（散列！=“sha2-256”）->cidv1
	if options.MhType != mh.SHA2_256 {
		switch options.CidVersion {
		case 0:
			return nil, cid.Prefix{}, errors.New("CIDv0 only supports sha2-256")
		case 1, -1:
			options.CidVersion = 1
		default:
			return nil, cid.Prefix{}, fmt.Errorf("unknown CID version: %d", options.CidVersion)
		}
	} else {
		if options.CidVersion < 0 {
//默认为cidv0
			options.CidVersion = 0
		}
	}

//cidv1->原始块（默认）
	if options.CidVersion > 0 && !options.RawLeavesSet {
		options.RawLeaves = true
	}

	prefix, err := dag.PrefixForCidVersion(options.CidVersion)
	if err != nil {
		return nil, cid.Prefix{}, err
	}

	prefix.MhType = options.MhType
	prefix.MhLength = -1

	return options, prefix, nil
}

type unixfsOpts struct{}

var Unixfs unixfsOpts

//cidVersion指定要使用的cid版本。默认为0，除非选项
//这取决于CIDV1是否通过。
func (unixfsOpts) CidVersion(version int) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.CidVersion = version
		return nil
	}
}

//要使用的哈希函数。如果未设置为sha2-256，则表示cidv1（默认）。
//
//函数表在https://github.com/multiformats/go-multihash/blob/master/multihash.go中声明
func (unixfsOpts) Hash(mhtype uint64) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.MhType = mhtype
		return nil
	}
}

//raw leaves指定是否对叶使用原始块（不带
//链接）而不是用unixfs结构包装它们。
func (unixfsOpts) RawLeaves(enable bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.RawLeaves = enable
		settings.RawLeavesSet = true
		return nil
	}
}

//inline告诉加法器将小的块内联到CID中
func (unixfsOpts) Inline(enable bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Inline = enable
		return nil
	}
}

//inlinelimit设置将在下面对块进行编码的字节数
//直接进入cid，而不是由散列存储和寻址。
//指定此选项不会启用块内联。用于“inline”
//选择权。默认值：32字节
//
//注意，虽然字节数没有硬限制，但应该是
//保持在一个相当低的值，例如64；实现可能选择
//拒绝任何更大的。
func (unixfsOpts) InlineLimit(limit int) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.InlineLimit = limit
		return nil
	}
}

//chunker指定要使用的分块算法的设置。
//
//默认值：262144，格式：
//大小-[字节]-简单的分块器，将数据分为n个字节的块
//拉宾-[Min]-[Avg]-[Max]-拉宾Chunker
func (unixfsOpts) Chunker(chunker string) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Chunker = chunker
		return nil
	}
}

//布局告诉加法器如何平衡叶之间的数据。
//options.balancedlayout是默认值，它针对静态可查找性进行了优化
//文件夹。
//options.tricklelayout针对流式数据进行了优化，
func (unixfsOpts) Layout(layout Layout) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Layout = layout
		return nil
	}
}

//pin告诉加法器在添加后递归地固定文件根目录
func (unixfsOpts) Pin(pin bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Pin = pin
		return nil
	}
}

//hashOnly将使加法器计算数据哈希而不将其存储在
//BlockStore或向网络公布
func (unixfsOpts) HashOnly(hashOnly bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.OnlyHash = hashOnly
		return nil
	}
}

//wrap告诉加法器用附加的
//目录。
func (unixfsOpts) Wrap(wrap bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Wrap = wrap
		return nil
	}
}

//隐藏允许添加隐藏文件（以“.”为前缀的文件）
func (unixfsOpts) Hidden(hidden bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Hidden = hidden
		return nil
	}
}

//stdinname是为不指定filepath为的文件设置的名称
//OS.STDIN。名称（）
func (unixfsOpts) StdinName(name string) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.StdinName = name
		return nil
	}
}

//事件指定将用于报告有关正在进行的事件的通道
//添加操作。
//
//注意，如果这个通道阻塞，可能会减慢加法器的速度。
func (unixfsOpts) Events(sink chan<- interface{}) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Events = sink
		return nil
	}
}

//静默减少事件输出
func (unixfsOpts) Silent(silent bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Silent = silent
		return nil
	}
}

//进度告诉加法器是否启用进度事件
func (unixfsOpts) Progress(enable bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.Progress = enable
		return nil
	}
}

//fscache告诉加法器检查文件存储是否存在预先存在的块
//
//实验
func (unixfsOpts) FsCache(enable bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.FsCache = enable
		return nil
	}
}

//nocopy告诉加法器使用文件存储添加文件。意味着劳利夫斯。
//
//实验
func (unixfsOpts) Nocopy(enable bool) UnixfsAddOption {
	return func(settings *UnixfsAddSettings) error {
		settings.NoCopy = enable
		return nil
	}
}
