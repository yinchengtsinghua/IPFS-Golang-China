
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package core

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	math2 "github.com/ipfs/go-ipfs/thirdparty/math2"
	lgbl "gx/ipfs/QmWZnkipEJbsdUDhgTGBnKAN1iHM2WDMNqsXHyAuaAiCgo/go-libp2p-loggables"

	inet "gx/ipfs/QmNgLg1NTw37iWbYPKcyK85YJ9Whs1MkPtJwhfqbNYAyKg/go-libp2p-net"
	pstore "gx/ipfs/QmPiemjiKBC9VA7vZF82m4x1oygtg2c2YVqag8PX7dN1BD/go-libp2p-peerstore"
	goprocess "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	procctx "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/context"
	periodicproc "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/periodic"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	host "gx/ipfs/QmaoXrM4Z41PD48JY36YqQGKQpLGjyLA2cKcLsES7YddAq/go-libp2p-host"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
)

//errnotenoughbootstrappers表示我们没有足够的引导
//对等机正确引导。
var ErrNotEnoughBootstrapPeers = errors.New("not enough bootstrap peers to bootstrap")

//bootstrapconfig指定在ipfsnode网络中使用的参数
//引导过程。
type BootstrapConfig struct {

//MinPeerThreshold控制是否引导更多连接。如果
//节点的打开连接数少于此数字，它将打开连接
//到引导程序节点。从那里，路由系统应该能够
//使用到引导节点的连接来连接到更多
//同龄人。像ipfsdht这样的路由系统在它们自己的引导程序中这样做。
//进程，它发出随机查询以查找更多对等方。
	MinPeerThreshold int

//周期控制节点将
//尝试引导。引导过程不是很昂贵，所以
//这个阈值可以很小（<=30s）。
	Period time.Duration

//ConnectionTimeout确定等待引导的时间
//取消前尝试连接。
	ConnectionTimeout time.Duration

//引导程序是返回一组引导程序对等端的函数。
//用于引导进程。这使客户有可能
//控制进程在任何时刻使用的对等机。
	BootstrapPeers func() []pstore.PeerInfo
}

//defaultbootstrapconfig为引导指定默认的SANE参数。
var DefaultBootstrapConfig = BootstrapConfig{
	MinPeerThreshold:  4,
	Period:            30 * time.Second,
ConnectionTimeout: (30 * time.Second) / 3, //PiOD/ 3
}

func BootstrapConfigWithPeers(pis []pstore.PeerInfo) BootstrapConfig {
	cfg := DefaultBootstrapConfig
	cfg.BootstrapPeers = func() []pstore.PeerInfo {
		return pis
	}
	return cfg
}

//引导启动ipfsnode引导。此功能将定期
//检查打开的连接数，如果连接数太少，则启动
//连接到著名的引导程序对等端。它还启动子系统
//引导（即路由）。
func Bootstrap(n *IpfsNode, cfg BootstrapConfig) (io.Closer, error) {

//发出信号等待一轮引导完成。
	doneWithRound := make(chan struct{})

	if len(cfg.BootstrapPeers()) == 0 {
//我们*需要*来引导，但我们没有引导对等机
//已配置*，通知用户。
		log.Error("no bootstrap nodes configured: go-ipfs may have difficulty connecting to the network")
	}

//周期性引导函数——连接管理器
	periodic := func(worker goprocess.Process) {
		ctx := procctx.OnClosingContext(worker)
		defer log.EventBegin(ctx, "periodicBootstrap", n.Identity).Done()

		if err := bootstrapRound(ctx, n.PeerHost, cfg); err != nil {
			log.Event(ctx, "bootstrapError", n.Identity, lgbl.Error(err))
			log.Debugf("%s bootstrap error: %s", n.Identity, err)
		}

		<-doneWithRound
	}

//启动节点的定期引导
	proc := periodicproc.Tick(cfg.Period, periodic)
proc.Go(periodic) //立即运行一个。

//启动路由。引导
	if n.Routing != nil {
		ctx := procctx.OnClosingContext(proc)
		if err := n.Routing.Bootstrap(ctx); err != nil {
			proc.Close()
			return nil, err
		}
	}

	doneWithRound <- struct{}{}
close(doneWithRound) //它不再阻止周期性
	return proc, nil
}

func bootstrapRound(ctx context.Context, host host.Host, cfg BootstrapConfig) error {

	ctx, cancel := context.WithTimeout(ctx, cfg.ConnectionTimeout)
	defer cancel()
	id := host.ID()

//从配置中获取引导程序对等。在这里检索可以使
//当然，我们仍然关注客户机配置的更改。
	peers := cfg.BootstrapPeers()
//确定要打开多少引导连接
	connected := host.Network().Peers()
	if len(connected) >= cfg.MinPeerThreshold {
		log.Event(ctx, "bootstrapSkip", id)
		log.Debugf("%s core bootstrap skipped -- connected to %d (> %d) nodes",
			id, len(connected), cfg.MinPeerThreshold)
		return nil
	}
	numToDial := cfg.MinPeerThreshold - len(connected)

//筛选出已连接到的引导程序节点
	var notConnected []pstore.PeerInfo
	for _, p := range peers {
		if host.Network().Connectedness(p.ID) != inet.Connected {
			notConnected = append(notConnected, p)
		}
	}

//如果连接到所有引导对等候选，则退出
	if len(notConnected) < 1 {
		log.Debugf("%s no more bootstrap peers to create %d connections", id, numToDial)
		return ErrNotEnoughBootstrapPeers
	}

//连接到一个随机的候选引导集
	randSubset := randomSubsetOfPeers(notConnected, numToDial)

	defer log.EventBegin(ctx, "bootstrapStart", id).Done()
	log.Debugf("%s bootstrapping to %d nodes: %s", id, numToDial, randSubset)
	return bootstrapConnect(ctx, host, randSubset)
}

func bootstrapConnect(ctx context.Context, ph host.Host, peers []pstore.PeerInfo) error {
	if len(peers) < 1 {
		return ErrNotEnoughBootstrapPeers
	}

	errs := make(chan error, len(peers))
	var wg sync.WaitGroup
	for _, p := range peers {

//异步执行，因为当同步执行时，如果
//一个“connect”调用挂起，后续调用更可能
//由于上下文过期而失败/中止。
//此外，还异步执行拨号速度。

		wg.Add(1)
		go func(p pstore.PeerInfo) {
			defer wg.Done()
			defer log.EventBegin(ctx, "bootstrapDial", ph.ID(), p.ID).Done()
			log.Debugf("%s bootstrapping to %s", ph.ID(), p.ID)

			ph.Peerstore().AddAddrs(p.ID, p.Addrs, pstore.PermanentAddrTTL)
			if err := ph.Connect(ctx, p); err != nil {
				log.Event(ctx, "bootstrapDialFailed", p.ID)
				log.Debugf("failed to bootstrap with %v: %s", p.ID, err)
				errs <- err
				return
			}
			log.Event(ctx, "bootstrapDialSuccess", p.ID)
			log.Infof("bootstrapped with %v", p.ID)
		}(p)
	}
	wg.Wait()

//我们的失败条件是没有成功的连接尝试。
//因此，清空errs通道，计算结果。
	close(errs)
	count := 0
	var err error
	for err = range errs {
		if err != nil {
			count++
		}
	}
	if count == len(peers) {
		return fmt.Errorf("failed to bootstrap. %s", err)
	}
	return nil
}

func toPeerInfos(bpeers []config.BootstrapPeer) []pstore.PeerInfo {
	pinfos := make(map[peer.ID]*pstore.PeerInfo)
	for _, bootstrap := range bpeers {
		pinfo, ok := pinfos[bootstrap.ID()]
		if !ok {
			pinfo = new(pstore.PeerInfo)
			pinfos[bootstrap.ID()] = pinfo
			pinfo.ID = bootstrap.ID()
		}

		pinfo.Addrs = append(pinfo.Addrs, bootstrap.Transport())
	}

	var peers []pstore.PeerInfo
	for _, pinfo := range pinfos {
		peers = append(peers, *pinfo)
	}

	return peers
}

func randomSubsetOfPeers(in []pstore.PeerInfo, max int) []pstore.PeerInfo {
	n := math2.IntMin(max, len(in))
	var out []pstore.PeerInfo
	for _, val := range rand.Perm(len(in)) {
		out = append(out, in[val])
		if len(out) >= n {
			break
		}
	}
	return out
}
