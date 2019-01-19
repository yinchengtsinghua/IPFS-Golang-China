
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corehttp

import (
	"context"
	"testing"
	"time"

	core "github.com/ipfs/go-ipfs/core"

	inet "gx/ipfs/QmNgLg1NTw37iWbYPKcyK85YJ9Whs1MkPtJwhfqbNYAyKg/go-libp2p-net"
	bhost "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p/p2p/host/basic"
	swarmt "gx/ipfs/QmegQFxhr1J6yZ1vDQuDmJi5jntmj6BL96S11HVtXNCaHb/go-libp2p-swarm/testing"
)

//此测试基于go-libp2p/p2p/net/swarm.testconnestednesscorrect
//它构建4个节点并连接它们，其中一个是唯一的中心。
//然后检查中心是否报告了正确的对等数。
func TestPeersTotal(t *testing.T) {
	ctx := context.Background()

	hosts := make([]*bhost.BasicHost, 4)
	for i := 0; i < 4; i++ {
		hosts[i] = bhost.New(swarmt.GenSwarm(t, ctx))
	}

	dial := func(a, b inet.Network) {
		swarmt.DivulgeAddresses(b, a)
		if _, err := a.DialPeer(ctx, b.LocalPeer()); err != nil {
			t.Fatalf("Failed to dial: %s", err)
		}
	}

	dial(hosts[0].Network(), hosts[1].Network())
	dial(hosts[0].Network(), hosts[2].Network())
	dial(hosts[0].Network(), hosts[3].Network())

//我想拨号盘有问题。还没完呢
//完全。一定有一些异步的东西。
	<-time.After(100 * time.Millisecond)

	node := &core.IpfsNode{PeerHost: hosts[0]}
	collector := IpfsNodeCollector{Node: node}
	actual := collector.PeersTotalValues()
	if len(actual) != 1 {
		t.Fatalf("expected 1 peers transport, got %d", len(actual))
	}
	if actual["/ip4/tcp"] != float64(3) {
		t.Fatalf("expected 3 peers, got %f", actual["/ip4/tcp"])
	}
}
