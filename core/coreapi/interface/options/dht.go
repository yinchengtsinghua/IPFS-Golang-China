
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

type DhtProvideSettings struct {
	Recursive bool
}

type DhtFindProvidersSettings struct {
	NumProviders int
}

type DhtProvideOption func(*DhtProvideSettings) error
type DhtFindProvidersOption func(*DhtFindProvidersSettings) error

func DhtProvideOptions(opts ...DhtProvideOption) (*DhtProvideSettings, error) {
	options := &DhtProvideSettings{
		Recursive: false,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

func DhtFindProvidersOptions(opts ...DhtFindProvidersOption) (*DhtFindProvidersSettings, error) {
	options := &DhtFindProvidersSettings{
		NumProviders: 20,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

type dhtOpts struct{}

var Dht dhtOpts

//递归是dht.provide的一个选项，指定是否提供
//给定路径递归
func (dhtOpts) Recursive(recursive bool) DhtProvideOption {
	return func(settings *DhtProvideSettings) error {
		settings.Recursive = recursive
		return nil
	}
}

//numProviders是dht.findproviders的一个选项，它指定
//要查找的对等数。默认值为20
func (dhtOpts) NumProviders(numProviders int) DhtFindProvidersOption {
	return func(settings *DhtFindProvidersSettings) error {
		settings.NumProviders = numProviders
		return nil
	}
}
