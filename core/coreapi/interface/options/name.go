
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package options

import (
	"time"

	ropts "github.com/ipfs/go-ipfs/namesys/opts"
)

const (
	DefaultNameValidTime = 24 * time.Hour
)

type NamePublishSettings struct {
	ValidTime time.Duration
	Key       string

	TTL *time.Duration

	AllowOffline bool
}

type NameResolveSettings struct {
	Cache bool

	ResolveOpts []ropts.ResolveOpt
}

type NamePublishOption func(*NamePublishSettings) error
type NameResolveOption func(*NameResolveSettings) error

func NamePublishOptions(opts ...NamePublishOption) (*NamePublishSettings, error) {
	options := &NamePublishSettings{
		ValidTime: DefaultNameValidTime,
		Key:       "self",

		AllowOffline: false,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}

	return options, nil
}

func NameResolveOptions(opts ...NameResolveOption) (*NameResolveSettings, error) {
	options := &NameResolveSettings{
		Cache: true,
	}

	for _, opt := range opts {
		err := opt(options)
		if err != nil {
			return nil, err
		}
	}

	return options, nil
}

type nameOpts struct{}

var Name nameOpts

//validTime是name.publish的一个选项，它指定
//条目将保持有效。默认值为24小时
func (nameOpts) ValidTime(validTime time.Duration) NamePublishOption {
	return func(settings *NamePublishSettings) error {
		settings.ValidTime = validTime
		return nil
	}
}

//key是name.publish的一个选项，它指定用于
//出版业。默认值是“self”，它是节点自己的peerID。
//密钥参数必须是peerID或keystore密钥别名。
//
//可以使用keyapi列出并生成更多的名称及其各自的键。
func (nameOpts) Key(key string) NamePublishOption {
	return func(settings *NamePublishSettings) error {
		settings.Key = key
		return nil
	}
}

//allowoffline是name.publish的一个选项，用于指定是否允许
//节点脱机时发布。默认值为假
func (nameOpts) AllowOffline(allow bool) NamePublishOption {
	return func(settings *NamePublishSettings) error {
		settings.AllowOffline = allow
		return nil
	}
}

//TTL是name.publish的一个选项，它指定
//应为缓存已发布的记录（注意：实验）。
func (nameOpts) TTL(ttl time.Duration) NamePublishOption {
	return func(settings *NamePublishSettings) error {
		settings.TTL = &ttl
		return nil
	}
}

//缓存是名称的一个选项。请解析哪个选项指定是否应使用缓存。
//默认值为真
func (nameOpts) Cache(cache bool) NameResolveOption {
	return func(settings *NameResolveSettings) error {
		settings.Cache = cache
		return nil
	}
}

//
func (nameOpts) ResolveOption(opt ropts.ResolveOpt) NameResolveOption {
	return func(settings *NameResolveSettings) error {
		settings.ResolveOpts = append(settings.ResolveOpts, opt)
		return nil
	}
}
