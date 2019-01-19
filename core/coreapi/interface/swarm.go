
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
	"time"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	net "gx/ipfs/QmNgLg1NTw37iWbYPKcyK85YJ9Whs1MkPtJwhfqbNYAyKg/go-libp2p-net"
	pstore "gx/ipfs/QmPiemjiKBC9VA7vZF82m4x1oygtg2c2YVqag8PX7dN1BD/go-libp2p-peerstore"
	"gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
)

var (
	ErrNotConnected = errors.New("not connected")
	ErrConnNotFound = errors.New("conn not found")
)

//ConnectionInfo包含有关对等机的信息
type ConnectionInfo interface {
//id返回peerid
	ID() peer.ID

//地址返回与对等机连接的多地址
	Address() ma.Multiaddr

//方向返回建立连接的方式
	Direction() net.Direction

//延迟向对等端返回上次已知的往返时间
	Latency() (time.Duration, error)

//流返回与对等方建立的流的列表
	Streams() ([]protocol.ID, error)
}

//swarmapi指定libp2p swarm的接口
type SwarmAPI interface {
//
	Connect(context.Context, pstore.PeerInfo) error

//断开与给定地址的连接
	Disconnect(context.Context, ma.Multiaddr) error

//peers返回我们连接到的对等机列表
	Peers(context.Context) ([]ConnectionInfo, error)

//knownaddrs返回此节点知道的所有地址的列表
	KnownAddrs(context.Context) (map[peer.ID][]ma.Multiaddr, error)

//localaddrs返回已宣布的侦听地址列表
	LocalAddrs(context.Context) ([]ma.Multiaddr, error)

//listenaddrs返回所有侦听地址的列表
	ListenAddrs(context.Context) ([]ma.Multiaddr, error)
}
