
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
	"io"

	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
)

//pubsubscription是一个活动的pubsubscription
type PubSubSubscription interface {
	io.Closer

//下一步返回下一条传入消息
	Next(context.Context) (PubSubMessage, error)
}

//pubsub message是单个pubsub消息
type PubSubMessage interface {
//From返回消息到达的对等方的ID
	From() peer.ID

//数据返回消息体
	Data() []byte

//seq返回消息标识符
	Seq() []byte

//主题返回此邮件设置为的主题列表
	Topics() []string
}

//pubsubapi指定到pubsub的接口
type PubSubAPI interface {
//LS按名称列出订阅的主题
	Ls(context.Context) ([]string, error)

//对等方列出我们当前发布的对等方
	Peers(context.Context, ...options.PubSubPeersOption) ([]peer.ID, error)

//将消息发布到给定的pubsub主题
	Publish(context.Context, string, []byte) error

//订阅给定主题的消息
	Subscribe(context.Context, string, ...options.PubSubSubscribeOption) (PubSubSubscription, error)
}
