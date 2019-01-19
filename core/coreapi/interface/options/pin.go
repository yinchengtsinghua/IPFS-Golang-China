
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

type PinAddSettings struct {
	Recursive bool
}

type PinLsSettings struct {
	Type string
}

type PinUpdateSettings struct {
	Unpin bool
}

type PinAddOption func(*PinAddSettings) error
type PinLsOption func(settings *PinLsSettings) error
type PinUpdateOption func(*PinUpdateSettings) error

func PinAddOptions(opts ...PinAddOption) (*PinAddSettings, error) {
	options := &PinAddSettings{
		Recursive: true,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}

	return options, nil
}

func PinLsOptions(opts ...PinLsOption) (*PinLsSettings, error) {
	options := &PinLsSettings{
		Type: "all",
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}

	return options, nil
}

func PinUpdateOptions(opts ...PinUpdateOption) (*PinUpdateSettings, error) {
	options := &PinUpdateSettings{
		Unpin: true,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}

	return options, nil
}

type pinType struct{}

type pinOpts struct {
	Type pinType
}

var Pin pinOpts

//all是pin.ls的选项，它将返回所有pin。它是
//默认值
func (pinType) All() PinLsOption {
	return Pin.pinType("all")
}

//recursive是pin.ls的一个选项，它只返回recursive
//插脚
func (pinType) Recursive() PinLsOption {
	return Pin.pinType("recursive")
}

//直接是pin.ls的一个选项，它将使它只返回直接（非
//递归）引脚
func (pinType) Direct() PinLsOption {
	return Pin.pinType("direct")
}

//间接是pin.ls的一个选项，它将使它只返回间接pin
//（其他递归固定对象引用的对象）
func (pinType) Indirect() PinLsOption {
	return Pin.pinType("indirect")
}

//recursive是pin.add的一个选项，它指定是否固定整个
//对象树或仅一个对象。默认值：真
func (pinOpts) Recursive(recursive bool) PinAddOption {
	return func(settings *PinAddSettings) error {
		settings.Recursive = recursive
		return nil
	}
}

//type是pin.ls的一个选项，它允许指定应该使用哪个pin类型
//归还
//
//支持的值：
//*“直接”-直接固定对象
//*“recursive”-递归管脚的根
//*“间接”-间接固定的对象（由递归固定引用）
//对象）
//*“all”-所有固定对象（默认）
func (pinOpts) pinType(t string) PinLsOption {
	return func(settings *PinLsSettings) error {
		settings.Type = t
		return nil
	}
}

//unpin是pin.update的一个选项，用于指定是否删除旧的pin。
//默认值为true。
func (pinOpts) Unpin(unpin bool) PinUpdateOption {
	return func(settings *PinUpdateSettings) error {
		settings.Unpin = unpin
		return nil
	}
}
