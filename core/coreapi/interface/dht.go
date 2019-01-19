
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

	"github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	pstore "gx/ipfs/QmPiemjiKBC9VA7vZF82m4x1oygtg2c2YVqag8PX7dN1BD/go-libp2p-peerstore"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
)

//DHTAPI指定DHT的接口
//注意：此API在不久的将来可能会被弃用，请参阅
//https://github.com/ipfs/interface-ipfs-core/issues/249了解更多上下文。
type DhtAPI interface {
//findpeer查询DHT中与
//对等体ID
	FindPeer(context.Context, peer.ID) (pstore.PeerInfo, error)

//findproviders在DHT中查找能够提供特定值的对等方
//给出了一个键。
	FindProviders(context.Context, Path, ...options.DhtFindProvidersOption) (<-chan pstore.PeerInfo, error)

//向网络宣布您正在提供给定值
	Provide(context.Context, Path, ...options.DhtProvideOption) error
}
