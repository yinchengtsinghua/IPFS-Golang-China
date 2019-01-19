
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

type PubSubPeersSettings struct {
	Topic string
}

type PubSubSubscribeSettings struct {
	Discover bool
}

type PubSubPeersOption func(*PubSubPeersSettings) error
type PubSubSubscribeOption func(*PubSubSubscribeSettings) error

func PubSubPeersOptions(opts ...PubSubPeersOption) (*PubSubPeersSettings, error) {
	options := &PubSubPeersSettings{
		Topic: "",
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

func PubSubSubscribeOptions(opts ...PubSubSubscribeOption) (*PubSubSubscribeSettings, error) {
	options := &PubSubSubscribeSettings{
		Discover: false,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}
	return options, nil
}

type pubsubOpts struct{}

var PubSub pubsubOpts

func (pubsubOpts) Topic(topic string) PubSubPeersOption {
	return func(settings *PubSubPeersSettings) error {
		settings.Topic = topic
		return nil
	}
}

func (pubsubOpts) Discover(discover bool) PubSubSubscribeOption {
	return func(settings *PubSubSubscribeSettings) error {
		settings.Discover = discover
		return nil
	}
}
