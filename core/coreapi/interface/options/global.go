
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

type ApiSettings struct {
	Offline     bool
	FetchBlocks bool
}

type ApiOption func(*ApiSettings) error

func ApiOptions(opts ...ApiOption) (*ApiSettings, error) {
	options := &ApiSettings{
		Offline:     false,
		FetchBlocks: true,
	}

	return ApiOptionsTo(options, opts...)
}

func ApiOptionsTo(options *ApiSettings, opts ...ApiOption) (*ApiSettings, error) {
	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

type apiOpts struct{}

var Api apiOpts

func (apiOpts) Offline(offline bool) ApiOption {
	return func(settings *ApiSettings) error {
		settings.Offline = offline
		return nil
	}
}

//如果设置为false，则fetchBlocks会阻止API从
//网络，同时允许其他服务（如IPN）仍然在线
func (apiOpts) FetchBlocks(fetch bool) ApiOption {
	return func(settings *ApiSettings) error {
		settings.FetchBlocks = fetch
		return nil
	}
}
