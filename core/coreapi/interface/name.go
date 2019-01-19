
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package iface

import (
	"context"
	"errors"

	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"
)

var ErrResolveFailed = errors.New("could not resolve name")

//ipnsentry指定ipnsentries的接口
type IpnsEntry interface {
//name返回ipnsentry名称
	Name() string
//值返回ipnsentry值
	Value() Path
}

type IpnsResult struct {
	Path
	Err error
}

//nameapi指定IPN的接口。
//
//IPN是一个pki命名空间，其中名称是公钥的散列值，并且
//私钥允许发布新的（签名的）值。在发布和
//解析，使用的默认名称是节点自己的peerID，它是
//它的公钥。
//
//可以使用.key API列出并生成更多名称及其各自的键。
type NameAPI interface {
//发布宣布新的IPN名称
	Publish(ctx context.Context, path Path, opts ...options.NamePublishOption) (IpnsEntry, error)

//解析尝试解析指定名称的最新版本
	Resolve(ctx context.Context, name string, opts ...options.NameResolveOption) (Path, error)

//搜索是解析的一个版本，它在发现路径时输出路径，
//缩短第一次进入的时间
//
//注意：默认情况下，从通道读取的所有路径都被认为是不安全的，
//除了最新的（通道读取缓冲区中的最后一个路径）。
	Search(ctx context.Context, name string, opts ...options.NameResolveOption) (<-chan IpnsResult, error)
}
