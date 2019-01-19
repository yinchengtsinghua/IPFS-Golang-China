
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
package core实现ipfsnode对象和相关方法。

核心下的软件包/提供（相对）稳定的低级API
执行大多数与IPF相关的任务。有关另一个的详细信息
接口和核心技术。适合更大的IPF图片，请参见：

  $godoc github.com/ipfs/go-ipfs
**/

package core

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	version "github.com/ipfs/go-ipfs"
	rp "github.com/ipfs/go-ipfs/exchange/reprovide"
	filestore "github.com/ipfs/go-ipfs/filestore"
	mount "github.com/ipfs/go-ipfs/fuse/mount"
	namesys "github.com/ipfs/go-ipfs/namesys"
	ipnsrp "github.com/ipfs/go-ipfs/namesys/republisher"
	p2p "github.com/ipfs/go-ipfs/p2p"
	pin "github.com/ipfs/go-ipfs/pin"
	repo "github.com/ipfs/go-ipfs/repo"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	resolver "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path/resolver"
	ic "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	dht "gx/ipfs/QmNoNExMdWrYSPZDiJJTVmxSh6uKLN26xYVzbLzBLedRcv/go-libp2p-kad-dht"
	dhtopts "gx/ipfs/QmNoNExMdWrYSPZDiJJTVmxSh6uKLN26xYVzbLzBLedRcv/go-libp2p-kad-dht/opts"
	u "gx/ipfs/QmNohiVssaPw3KVLZik59DBVGTSm2dGvYT9eoXt5DQ36Yz/go-ipfs-util"
	exchange "gx/ipfs/QmP2g3VxmC7g7fyRJDj1VJ72KHZbJ9UW24YjSWEj1XTb4H/go-ipfs-exchange-interface"
	mfs "gx/ipfs/QmP9eu5X5Ax8169jNWqAJcc42mdZgzLR1aKCEzqhNoBLKk/go-mfs"
	pstore "gx/ipfs/QmPiemjiKBC9VA7vZF82m4x1oygtg2c2YVqag8PX7dN1BD/go-libp2p-peerstore"
	ft "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs"
	mafilter "gx/ipfs/QmQgSnRC74nHoXrN9CShvfWUUSrgAMJ4unjbnuBVsxk2mw/go-maddr-filter"
	quic "gx/ipfs/QmR1g19UeP13BrVPCeEJm6R1J1E5yCdueiKpQJfPdnWC9z/go-libp2p-quic-transport"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	psrouter "gx/ipfs/QmReFccdPXitZc73LpfC299f9uQzMYnooAGsHJBGS5Mc4h/go-libp2p-pubsub-router"
	autonat "gx/ipfs/QmRmMbeY5QC5iMsuW16wchtFt8wmYTv2suWb8t9MV8dsxm/go-libp2p-autonat-svc"
	bstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	goprocess "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	ifconnmgr "gx/ipfs/QmSFo2QrMF4M1mKdB291ZqNtsie4NfwXCRdWgDU3inw4Ff/go-libp2p-interface-connmgr"
	mamask "gx/ipfs/QmSMZwvs3n4GBikZ7hKzT17c3bk65FmyZo2JqtJ16swqCv/multiaddr-filter"
	merkledag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	routing "gx/ipfs/QmTiRqrF5zkdZyrdsL5qndG1UbeWi8k8N2pYxCtXWrahR2/go-libp2p-routing"
	pubsub "gx/ipfs/QmVRxA4J3UPQpw74dLrQ6NJkfysCA1H4GU28gVpXQt9zMU/go-libp2p-pubsub"
	nilrouting "gx/ipfs/QmVZ6cQXHoTQja4oo9GhhHZi7dThi4x98mRKgGtKnTy37u/go-ipfs-routing/none"
	circuit "gx/ipfs/QmWuMW6UKZMJo9bFFDwnjg8tW3AtKisMHHrXEutQdmJ19N/go-libp2p-circuit"
	pnet "gx/ipfs/QmY4Q5JC4vxLEi8EpVxJM4rcRryEVtH1zRKVTAm6BKV1pg/go-libp2p-pnet"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	smux "gx/ipfs/QmY9JXR3FupnYAYJWK9aMr9bCpqWKcToQ1tz8DVGTrHpHw/go-stream-muxer"
	connmgr "gx/ipfs/QmYB44VSn76PMvefjvcKxdhnHtZxB36zrToCSh6u4H9U7M/go-libp2p-connmgr"
	bserv "gx/ipfs/QmYPZzd9VqmJDwxUnThfeSbV1Y5o53aVPDijTB7j7rS9Ep/go-blockservice"
	bitswap "gx/ipfs/QmYoGLuLwTUv1SYBmsw1EVNC9MyLVUxwxzXYtKgAGHyEfw/go-bitswap"
	bsnet "gx/ipfs/QmYoGLuLwTUv1SYBmsw1EVNC9MyLVUxwxzXYtKgAGHyEfw/go-bitswap/network"
	libp2p "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p"
	discovery "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p/p2p/discovery"
	p2pbhost "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p/p2p/host/basic"
	rhost "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p/p2p/host/routed"
	identify "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p/p2p/protocol/identify"
	mplex "gx/ipfs/QmZsejKNkeFSQe5TcmYXJ8iq6qPL1FpsP4eAA8j7RfE7xg/go-smux-multiplex"
	p2phost "gx/ipfs/QmaoXrM4Z41PD48JY36YqQGKQpLGjyLA2cKcLsES7YddAq/go-libp2p-host"
	metrics "gx/ipfs/QmbYN6UmTJn5UUQdi5CTsU86TXVBSrTcRk5UmyA36Qx2J6/go-libp2p-metrics"
	rhelpers "gx/ipfs/QmbYV2PXQVQnqerMBfuoNtzvBYnfzTRn9FZMGw6r3MHLDE/go-libp2p-routing-helpers"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	yamux "gx/ipfs/Qmdps3CYh5htGQSrPvzg5PHouVexLmtpbuLCqc4vuej8PC/go-smux-yamux"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
	record "gx/ipfs/QmfARXVCzpwFXQdepAJZuqyNDgV9doEsMnVCo1ssmuSe1U/go-libp2p-record"
)

const IpnsValidatorTag = "ipns"

const kReprovideFrequency = time.Hour * 12
const discoveryConnTimeout = time.Second * 30
const DefaultIpnsCacheSize = 128

var log = logging.Logger("core")

type mode int

const (
//零值不是有效模式，必须显式设置
	localMode mode = iota
	offlineMode
	onlineMode
)

func init() {
	identify.ClientVersion = "go-ipfs/" + version.CurrentVersionNumber + "/" + version.CurrentCommit
}

//ipfsnode是ipfs核心模块。它表示一个IPF实例。
type IpfsNode struct {

//自我
Identity peer.ID //本地节点的标识

	Repo repo.Repo

//局部节点
Pinning         pin.Pinner //固定管理器
Mounts          Mounts     //当前安装状态（如果有）。
PrivateKey      ic.PrivKey //本地节点的私钥
PNetFingerprint []byte     //专用网络指纹

//服务
Peerstore       pstore.Peerstore     //其他对等实例的存储
Blockstore      bstore.GCBlockstore  //街区商店（下层）
Filestore       *filestore.Filestore //文件存储块存储
BaseBlocks      bstore.Blockstore    //原始块存储，没有文件存储包装
GCLocker        bstore.GCLocker      //在GC期间用于保护BlockStore的储物柜
Blocks          bserv.BlockService   //块服务，获取/添加块。
DAG             ipld.DAGService      //Merkle DAG服务，获取/添加对象。
Resolver        *resolver.Resolver   //路径分辨系统
	Reporter        metrics.Reporter
	Discovery       discovery.Service
	FilesRoot       *mfs.Root
	RecordValidator record.Validator

//在线的
PeerHost     p2phost.Host        //网络主机（服务器+客户端）
Bootstrapper io.Closer           //周期性引导程序
Routing      routing.IpfsRouting //路由系统。推荐IPFS DHT
Exchange     exchange.Interface  //块交换+策略（位交换）
Namesys      namesys.NameSystem  //名称系统，解析哈希的路径
Reprovider   *rp.Reprovider      //价值再分系统
	IpnsRepub    *ipnsrp.Republisher

	AutoNAT  *autonat.AutoNATService
	PubSub   *pubsub.PubSub
	PSRouter *psrouter.PubsubValueStore
	DHT      *dht.IpfsDHT
	P2P      *p2p.P2P

	proc goprocess.Process
	ctx  context.Context

	mode         mode
	localModeSet bool
}

//mount s定义节点的mount状态是什么。这应该
//可能被移动到守护进程或挂载。它在这里是因为
//它需要跨守护进程请求进行访问。
type Mounts struct {
	Ipfs mount.Mount
	Ipns mount.Mount
}

func (n *IpfsNode) startOnlineServices(ctx context.Context, routingOption RoutingOption, hostOption HostOption, do DiscoveryOption, pubsub, ipnsps, mplex bool) error {
if n.PeerHost != nil { //已经上线了。
		return errors.New("node already online")
	}

	if n.PrivateKey == nil {
		return fmt.Errorf("private key not available")
	}

//从配置中获取无法解析的加法器
	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	var libp2pOpts []libp2p.Option
	for _, s := range cfg.Swarm.AddrFilters {
		f, err := mamask.NewMask(s)
		if err != nil {
			return fmt.Errorf("incorrectly formatted address filter in config: %s", s)
		}
		libp2pOpts = append(libp2pOpts, libp2p.FilterAddresses(f))
	}

	if !cfg.Swarm.DisableBandwidthMetrics {
//设置记者
		n.Reporter = metrics.NewBandwidthCounter()
		libp2pOpts = append(libp2pOpts, libp2p.BandwidthReporter(n.Reporter))
	}

	swarmkey, err := n.Repo.SwarmKey()
	if err != nil {
		return err
	}

	if swarmkey != nil {
		protec, err := pnet.NewProtector(bytes.NewReader(swarmkey))
		if err != nil {
			return fmt.Errorf("failed to configure private network: %s", err)
		}
		n.PNetFingerprint = protec.Fingerprint()
		go func() {
			t := time.NewTicker(30 * time.Second)
<-t.C //吞下一滴答
			for {
				select {
				case <-t.C:
					if ph := n.PeerHost; ph != nil {
						if len(ph.Network().Peers()) == 0 {
							log.Warning("We are in private network and have no peers.")
							log.Warning("This might be configuration mistake.")
						}
					}
				case <-n.Process().Closing():
					t.Stop()
					return
				}
			}
		}()

		libp2pOpts = append(libp2pOpts, libp2p.PrivateNetwork(protec))
	}

	addrsFactory, err := makeAddrsFactory(cfg.Addresses)
	if err != nil {
		return err
	}
	if !cfg.Swarm.DisableRelay {
		addrsFactory = composeAddrsFactory(addrsFactory, filterRelayAddrs)
	}
	libp2pOpts = append(libp2pOpts, libp2p.AddrsFactory(addrsFactory))

	connm, err := constructConnMgr(cfg.Swarm.ConnMgr)
	if err != nil {
		return err
	}
	libp2pOpts = append(libp2pOpts, libp2p.ConnectionManager(connm))

	libp2pOpts = append(libp2pOpts, makeSmuxTransportOption(mplex))

	if !cfg.Swarm.DisableNatPortMap {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap())
	}

//禁用默认侦听加法器
	libp2pOpts = append(libp2pOpts, libp2p.NoListenAddrs)

	if cfg.Swarm.DisableRelay {
//默认启用。
		libp2pOpts = append(libp2pOpts, libp2p.DisableRelay())
	} else {
		relayOpts := []circuit.RelayOpt{circuit.OptDiscovery}
		if cfg.Swarm.EnableRelayHop {
			relayOpts = append(relayOpts, circuit.OptHop)
		}
		libp2pOpts = append(libp2pOpts, libp2p.EnableRelay(relayOpts...))
	}

//显式启用默认传输
	libp2pOpts = append(libp2pOpts, libp2p.DefaultTransports)

	if cfg.Experimental.QUIC {
		libp2pOpts = append(libp2pOpts, libp2p.Transport(quic.NewTransport))
	}

//允许路由
	libp2pOpts = append(libp2pOpts, libp2p.Routing(func(h p2phost.Host) (routing.PeerRouting, error) {
		r, err := routingOption(ctx, h, n.Repo.Datastore(), n.RecordValidator)
		n.Routing = r
		return r, err
	}))

//启用自动播放
	if cfg.Swarm.EnableAutoRelay {
		libp2pOpts = append(libp2pOpts, libp2p.EnableAutoRelay())
	}

	peerhost, err := hostOption(ctx, n.Identity, n.Peerstore, libp2pOpts...)

	if err != nil {
		return err
	}

	n.PeerHost = peerhost

	if err := n.startOnlineServicesWithHost(ctx, routingOption, pubsub, ipnsps); err != nil {
		return err
	}

//好了，现在我们准备好倾听了。
	if err := startListening(n.PeerHost, cfg); err != nil {
		return err
	}

	n.P2P = p2p.NewP2P(n.Identity, n.PeerHost, n.Peerstore)

//设置本地发现
	if do != nil {
		service, err := do(ctx, n.PeerHost)
		if err != nil {
			log.Error("mdns error: ", err)
		} else {
			service.RegisterNotifee(n)
			n.Discovery = service
		}
	}

	return n.Bootstrap(DefaultBootstrapConfig)
}

func constructConnMgr(cfg config.ConnMgr) (ifconnmgr.ConnManager, error) {
	switch cfg.Type {
	case "":
//“默认”值是基本连接管理器
		return connmgr.NewConnManager(config.DefaultConnMgrLowWater, config.DefaultConnMgrHighWater, config.DefaultConnMgrGracePeriod), nil
	case "none":
		return nil, nil
	case "basic":
		grace, err := time.ParseDuration(cfg.GracePeriod)
		if err != nil {
			return nil, fmt.Errorf("parsing Swarm.ConnMgr.GracePeriod: %s", err)
		}

		return connmgr.NewConnManager(cfg.LowWater, cfg.HighWater, grace), nil
	default:
		return nil, fmt.Errorf("unrecognized ConnMgr.Type: %q", cfg.Type)
	}
}

func (n *IpfsNode) startLateOnlineServices(ctx context.Context) error {
	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	var keyProvider rp.KeyChanFunc

	switch cfg.Reprovider.Strategy {
	case "all":
		fallthrough
	case "":
		keyProvider = rp.NewBlockstoreProvider(n.Blockstore)
	case "roots":
		keyProvider = rp.NewPinnedProvider(n.Pinning, n.DAG, true)
	case "pinned":
		keyProvider = rp.NewPinnedProvider(n.Pinning, n.DAG, false)
	default:
		return fmt.Errorf("unknown reprovider strategy '%s'", cfg.Reprovider.Strategy)
	}
	n.Reprovider = rp.NewReprovider(ctx, n.Routing, keyProvider)

	reproviderInterval := kReprovideFrequency
	if cfg.Reprovider.Interval != "" {
		dur, err := time.ParseDuration(cfg.Reprovider.Interval)
		if err != nil {
			return err
		}

		reproviderInterval = dur
	}

	go n.Reprovider.Run(reproviderInterval)

	return nil
}

func makeAddrsFactory(cfg config.Addresses) (p2pbhost.AddrsFactory, error) {
	var annAddrs []ma.Multiaddr
	for _, addr := range cfg.Announce {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		annAddrs = append(annAddrs, maddr)
	}

	filters := mafilter.NewFilters()
	noAnnAddrs := map[string]bool{}
	for _, addr := range cfg.NoAnnounce {
		f, err := mamask.NewMask(addr)
		if err == nil {
			filters.AddDialFilter(f)
			continue
		}
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, err
		}
		noAnnAddrs[maddr.String()] = true
	}

	return func(allAddrs []ma.Multiaddr) []ma.Multiaddr {
		var addrs []ma.Multiaddr
		if len(annAddrs) > 0 {
			addrs = annAddrs
		} else {
			addrs = allAddrs
		}

		var out []ma.Multiaddr
		for _, maddr := range addrs {
//检查是否完全匹配
			ok, _ := noAnnAddrs[maddr.String()]
//检查/ipcidr匹配项
			if !ok && !filters.AddrBlocked(maddr) {
				out = append(out, maddr)
			}
		}
		return out
	}, nil
}

func makeSmuxTransportOption(mplexExp bool) libp2p.Option {
	const yamuxID = "/yamux/1.0.0"
	const mplexID = "/mplex/6.7.0"

	ymxtpt := &yamux.Transport{
		AcceptBacklog:          512,
		ConnectionWriteTimeout: time.Second * 10,
		KeepAliveInterval:      time.Second * 30,
		EnableKeepAlive:        true,
		MaxStreamWindowSize:    uint32(1024 * 512),
		LogOutput:              ioutil.Discard,
	}

	if os.Getenv("YAMUX_DEBUG") != "" {
		ymxtpt.LogOutput = os.Stderr
	}

	muxers := map[string]smux.Transport{yamuxID: ymxtpt}
	if mplexExp {
		muxers[mplexID] = mplex.DefaultTransport
	}

//允许覆盖muxer首选项顺序
	order := []string{yamuxID, mplexID}
	if prefs := os.Getenv("LIBP2P_MUX_PREFS"); prefs != "" {
		order = strings.Fields(prefs)
	}

	opts := make([]libp2p.Option, 0, len(order))
	for _, id := range order {
		tpt, ok := muxers[id]
		if !ok {
			log.Warning("unknown or duplicate muxer in LIBP2P_MUX_PREFS: %s", id)
			continue
		}
		delete(muxers, id)
		opts = append(opts, libp2p.Muxer(id, tpt))
	}

	return libp2p.ChainOptions(opts...)
}

func setupDiscoveryOption(d config.Discovery) DiscoveryOption {
	if d.MDNS.Enabled {
		return func(ctx context.Context, h p2phost.Host) (discovery.Service, error) {
			if d.MDNS.Interval == 0 {
				d.MDNS.Interval = 5
			}
			return discovery.NewMdnsService(ctx, h, time.Duration(d.MDNS.Interval)*time.Second, discovery.ServiceTag)
		}
	}
	return nil
}

//handlepeerfound尝试从“peerinfo”连接到对等机，如果失败
//记录警告日志。
func (n *IpfsNode) HandlePeerFound(p pstore.PeerInfo) {
	log.Warning("trying peer info: ", p)
	ctx, cancel := context.WithTimeout(n.Context(), discoveryConnTimeout)
	defer cancel()
	if err := n.PeerHost.Connect(ctx, p); err != nil {
		log.Warning("Failed to connect to peer found by discovery: ", err)
	}
}

//StartOnlineServicesWithHost是需要
//在开始监听之前用主机初始化。
func (n *IpfsNode) startOnlineServicesWithHost(ctx context.Context, routingOption RoutingOption, enablePubsub bool, enableIpnsps bool) error {
	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	if cfg.Swarm.EnableAutoNATService {
		var opts []libp2p.Option
		if cfg.Experimental.QUIC {
			opts = append(opts, libp2p.DefaultTransports, libp2p.Transport(quic.NewTransport))
		}

		svc, err := autonat.NewAutoNATService(ctx, n.PeerHost, opts...)
		if err != nil {
			return err
		}
		n.AutoNAT = svc
	}

	if enablePubsub || enableIpnsps {
		var service *pubsub.PubSub

		var pubsubOptions []pubsub.Option
		if cfg.Pubsub.DisableSigning {
			pubsubOptions = append(pubsubOptions, pubsub.WithMessageSigning(false))
		}

		if cfg.Pubsub.StrictSignatureVerification {
			pubsubOptions = append(pubsubOptions, pubsub.WithStrictSignatureVerification(true))
		}

		switch cfg.Pubsub.Router {
		case "":
			fallthrough
		case "floodsub":
			service, err = pubsub.NewFloodSub(ctx, n.PeerHost, pubsubOptions...)

		case "gossipsub":
			service, err = pubsub.NewGossipSub(ctx, n.PeerHost, pubsubOptions...)

		default:
			err = fmt.Errorf("Unknown pubsub router %s", cfg.Pubsub.Router)
		}

		if err != nil {
			return err
		}
		n.PubSub = service
	}

//此代码仅用于测试：模拟网络结构
//忽略实际构造路由的libp2p构造函数选项！
	if n.Routing == nil {
		r, err := routingOption(ctx, n.PeerHost, n.Repo.Datastore(), n.RecordValidator)
		if err != nil {
			return err
		}
		n.Routing = r
		n.PeerHost = rhost.Wrap(n.PeerHost, n.Routing)
	}

//托多：我不喜欢这种类型的断言，但是
//`routingoption`系统当前不提供对
//IpfsNode。
//
//理想情况下，我们会做如下的事情：
//
//1。在分层路由器中添加一些有趣的方法进行自省以提取
//像Pubsub路由器或DHT（复杂，混乱，
//可能不值得）。
//2。将ipfsnode传递到routingoption（还将删除
//下面是PSRouter案例。
//三。介绍某种服务经理？（我个人最喜欢的但是
//这需要大量的工作）。
	if dht, ok := n.Routing.(*dht.IpfsDHT); ok {
		n.DHT = dht
	}

	if enableIpnsps {
		n.PSRouter = psrouter.NewPubsubValueStore(
			ctx,
			n.PeerHost,
			n.Routing,
			n.PubSub,
			n.RecordValidator,
		)
		n.Routing = rhelpers.Tiered{
			Routers: []routing.IpfsRouting{
//务必先检查Pubsub。
				&rhelpers.Compose{
					ValueStore: &rhelpers.LimitedValueStore{
						ValueStore: n.PSRouter,
						Namespaces: []string{"ipns"},
					},
				},
				n.Routing,
			},
			Validator: n.RecordValidator,
		}
	}

//设置Exchange服务
	bitswapNetwork := bsnet.NewFromIpfsHost(n.PeerHost, n.Routing)
	n.Exchange = bitswap.New(ctx, bitswapNetwork, n.Blockstore)

	size, err := n.getCacheSize()
	if err != nil {
		return err
	}

//安装名称系统
	n.Namesys = namesys.NewNameSystem(n.Routing, n.Repo.Datastore(), size)

//安装IPN重新发布
	return n.setupIpnsRepublisher()
}

//getcachesize返回缓存寿命和缓存大小
func (n *IpfsNode) getCacheSize() (int, error) {
	cfg, err := n.Repo.Config()
	if err != nil {
		return 0, err
	}

	cs := cfg.Ipns.ResolveCacheSize
	if cs == 0 {
		cs = DefaultIpnsCacheSize
	}
	if cs < 0 {
		return 0, fmt.Errorf("cannot specify negative resolve cache size")
	}
	return cs, nil
}

func (n *IpfsNode) setupIpnsRepublisher() error {
	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	n.IpnsRepub = ipnsrp.NewRepublisher(n.Namesys, n.Repo.Datastore(), n.PrivateKey, n.Repo.Keystore())

	if cfg.Ipns.RepublishPeriod != "" {
		d, err := time.ParseDuration(cfg.Ipns.RepublishPeriod)
		if err != nil {
			return fmt.Errorf("failure to parse config setting IPNS.RepublishPeriod: %s", err)
		}

		if !u.Debug && (d < time.Minute || d > (time.Hour*24)) {
			return fmt.Errorf("config setting IPNS.RepublishPeriod is not between 1min and 1day: %s", d)
		}

		n.IpnsRepub.Interval = d
	}

	if cfg.Ipns.RecordLifetime != "" {
		d, err := time.ParseDuration(cfg.Ipns.RecordLifetime)
		if err != nil {
			return fmt.Errorf("failure to parse config setting IPNS.RecordLifetime: %s", err)
		}

		n.IpnsRepub.RecordLifetime = d
	}

	n.Process().Go(n.IpnsRepub.Run)

	return nil
}

//process返回process对象
func (n *IpfsNode) Process() goprocess.Process {
	return n.proc
}

//close对进程对象调用close（）。
func (n *IpfsNode) Close() error {
	return n.proc.Close()
}

//context返回ipfsnode上下文
func (n *IpfsNode) Context() context.Context {
	if n.ctx == nil {
		n.ctx = context.TODO()
	}
	return n.ctx
}

//Teardown关闭了所有的孩子。如果发生任何错误，此函数将返回
//第一个错误。
func (n *IpfsNode) teardown() error {
	log.Debug("core is shutting down...")
//拥有的对象在此拆卸中关闭，以确保它们已关闭
//无论使用哪个构造函数将它们添加到节点。
	var closers []io.Closer

//注意：对象添加（关闭）的顺序很重要，如果对象
//需要在关闭/清理过程中使用另一个，它应该
//在其他对象之前关闭

	if n.FilesRoot != nil {
		closers = append(closers, n.FilesRoot)
	}

	if n.Exchange != nil {
		closers = append(closers, n.Exchange)
	}

	if n.Mounts.Ipfs != nil && !n.Mounts.Ipfs.IsActive() {
		closers = append(closers, mount.Closer(n.Mounts.Ipfs))
	}
	if n.Mounts.Ipns != nil && !n.Mounts.Ipns.IsActive() {
		closers = append(closers, mount.Closer(n.Mounts.Ipns))
	}

	if n.DHT != nil {
		closers = append(closers, n.DHT.Process())
	}

	if n.Blocks != nil {
		closers = append(closers, n.Blocks)
	}

	if n.Bootstrapper != nil {
		closers = append(closers, n.Bootstrapper)
	}

	if n.PeerHost != nil {
		closers = append(closers, n.PeerHost)
	}

//回购最后一次结束，大部分事情都需要在这里保持状态
	closers = append(closers, n.Repo)

	var errs []error
	for _, closer := range closers {
		if err := closer.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

//onlinemode返回ipfsnode是否处于onlinemode。
func (n *IpfsNode) OnlineMode() bool {
	return n.mode == onlineMode
}

//set local将把ipfsnode设置为本地模式
func (n *IpfsNode) SetLocal(isLocal bool) {
	if isLocal {
		n.mode = localMode
	}
	n.localModeSet = true
}

//localmode返回ipfsnode是否处于localmode
func (n *IpfsNode) LocalMode() bool {
	if !n.localModeSet {
//不应发生程序员错误
		panic("local mode not set")
	}
	return n.mode == localMode
}

//引导程序将设置并调用ipfsnodes引导程序函数。
func (n *IpfsNode) Bootstrap(cfg BootstrapConfig) error {
//TODO在脱机模式下应返回什么值？
	if n.Routing == nil {
		return nil
	}

	if n.Bootstrapper != nil {
n.Bootstrapper.Close() //停止上一个引导进程。
	}

//如果调用方未指定引导对等函数，则获取
//配置中最新的引导程序对等。这将响应实时更改。
	if cfg.BootstrapPeers == nil {
		cfg.BootstrapPeers = func() []pstore.PeerInfo {
			ps, err := n.loadBootstrapPeers()
			if err != nil {
				log.Warning("failed to parse bootstrap peers from config")
				return nil
			}
			return ps
		}
	}

	var err error
	n.Bootstrapper, err = Bootstrap(n, cfg)
	return err
}

func (n *IpfsNode) loadID() error {
	if n.Identity != "" {
		return errors.New("identity already loaded")
	}

	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	cid := cfg.Identity.PeerID
	if cid == "" {
		return errors.New("identity was not set in config (was 'ipfs init' run?)")
	}
	if len(cid) == 0 {
		return errors.New("no peer ID in config! (was 'ipfs init' run?)")
	}

	id, err := peer.IDB58Decode(cid)
	if err != nil {
		return fmt.Errorf("peer ID invalid: %s", err)
	}

	n.Identity = id
	return nil
}

//getkey将从名为“name”的密钥库返回一个密钥。
func (n *IpfsNode) GetKey(name string) (ic.PrivKey, error) {
	if name == "self" {
		if n.PrivateKey == nil {
			return nil, fmt.Errorf("private key not available")
		}
		return n.PrivateKey, nil
	} else {
		return n.Repo.Keystore().Get(name)
	}
}

//loadprivatekey加载私钥*如果*可用
func (n *IpfsNode) loadPrivateKey() error {
	if n.Identity == "" || n.Peerstore == nil {
		return errors.New("loaded private key out of order")
	}

	if n.PrivateKey != nil {
		log.Warning("private key already loaded")
		return nil
	}

	cfg, err := n.Repo.Config()
	if err != nil {
		return err
	}

	if cfg.Identity.PrivKey == "" {
		return nil
	}

	sk, err := loadPrivateKey(&cfg.Identity, n.Identity)
	if err != nil {
		return err
	}

	n.PrivateKey = sk
	n.Peerstore.AddPrivKey(n.Identity, n.PrivateKey)
	n.Peerstore.AddPubKey(n.Identity, sk.GetPublic())
	return nil
}

func (n *IpfsNode) loadBootstrapPeers() ([]pstore.PeerInfo, error) {
	cfg, err := n.Repo.Config()
	if err != nil {
		return nil, err
	}

	parsed, err := cfg.BootstrapPeers()
	if err != nil {
		return nil, err
	}
	return toPeerInfos(parsed), nil
}

func (n *IpfsNode) loadFilesRoot() error {
	dsk := ds.NewKey("/local/filesroot")
	pf := func(ctx context.Context, c cid.Cid) error {
		return n.Repo.Datastore().Put(dsk, c.Bytes())
	}

	var nd *merkledag.ProtoNode
	val, err := n.Repo.Datastore().Get(dsk)

	switch {
	case err == ds.ErrNotFound || val == nil:
		nd = ft.EmptyDirNode()
		err := n.DAG.Add(n.Context(), nd)
		if err != nil {
			return fmt.Errorf("failure writing to dagstore: %s", err)
		}
	case err == nil:
		c, err := cid.Cast(val)
		if err != nil {
			return err
		}

		rnd, err := n.DAG.Get(n.Context(), c)
		if err != nil {
			return fmt.Errorf("error loading filesroot from DAG: %s", err)
		}

		pbnd, ok := rnd.(*merkledag.ProtoNode)
		if !ok {
			return merkledag.ErrNotProtobuf
		}

		nd = pbnd
	default:
		return err
	}

	mr, err := mfs.NewRoot(n.Context(), n.DAG, nd, pf)
	if err != nil {
		return err
	}

	n.FilesRoot = mr
	return nil
}

func loadPrivateKey(cfg *config.Identity, id peer.ID) (ic.PrivKey, error) {
	sk, err := cfg.DecodePrivateKey("passphrase todo!")
	if err != nil {
		return nil, err
	}

	id2, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		return nil, err
	}

	if id2 != id {
		return nil, fmt.Errorf("private key in config does not match id: %s != %s", id, id2)
	}

	return sk, nil
}

func listenAddresses(cfg *config.Config) ([]ma.Multiaddr, error) {
	var listen []ma.Multiaddr
	for _, addr := range cfg.Addresses.Swarm {
		maddr, err := ma.NewMultiaddr(addr)
		if err != nil {
			return nil, fmt.Errorf("failure to parse config.Addresses.Swarm: %s", cfg.Addresses.Swarm)
		}
		listen = append(listen, maddr)
	}

	return listen, nil
}

type ConstructPeerHostOpts struct {
	AddrsFactory      p2pbhost.AddrsFactory
	DisableNatPortMap bool
	DisableRelay      bool
	EnableRelayHop    bool
	ConnectionManager ifconnmgr.ConnManager
}

type HostOption func(ctx context.Context, id peer.ID, ps pstore.Peerstore, options ...libp2p.Option) (p2phost.Host, error)

var DefaultHostOption HostOption = constructPeerHost

//隔离复杂的初始化步骤
func constructPeerHost(ctx context.Context, id peer.ID, ps pstore.Peerstore, options ...libp2p.Option) (p2phost.Host, error) {
	pkey := ps.PrivKey(id)
	if pkey == nil {
		return nil, fmt.Errorf("missing private key for node ID: %s", id.Pretty())
	}
	options = append([]libp2p.Option{libp2p.Identity(pkey), libp2p.Peerstore(ps)}, options...)
	return libp2p.New(ctx, options...)
}

func filterRelayAddrs(addrs []ma.Multiaddr) []ma.Multiaddr {
	var raddrs []ma.Multiaddr
	for _, addr := range addrs {
		_, err := addr.ValueForProtocol(circuit.P_CIRCUIT)
		if err == nil {
			continue
		}
		raddrs = append(raddrs, addr)
	}
	return raddrs
}

func composeAddrsFactory(f, g p2pbhost.AddrsFactory) p2pbhost.AddrsFactory {
	return func(addrs []ma.Multiaddr) []ma.Multiaddr {
		return f(g(addrs))
	}
}

//网络地址令人吃惊
func startListening(host p2phost.Host, cfg *config.Config) error {
	listenAddrs, err := listenAddresses(cfg)
	if err != nil {
		return err
	}

//实际上开始聆听：
	if err := host.Network().Listen(listenAddrs...); err != nil {
		return err
	}

//列出我们的地址
	addrs, err := host.Network().InterfaceListenAddresses()
	if err != nil {
		return err
	}
	log.Infof("Swarm listening at: %s", addrs)
	return nil
}

func constructDHTRouting(ctx context.Context, host p2phost.Host, dstore ds.Batching, validator record.Validator) (routing.IpfsRouting, error) {
	return dht.New(
		ctx, host,
		dhtopts.Datastore(dstore),
		dhtopts.Validator(validator),
	)
}

func constructClientDHTRouting(ctx context.Context, host p2phost.Host, dstore ds.Batching, validator record.Validator) (routing.IpfsRouting, error) {
	return dht.New(
		ctx, host,
		dhtopts.Client(true),
		dhtopts.Datastore(dstore),
		dhtopts.Validator(validator),
	)
}

type RoutingOption func(context.Context, p2phost.Host, ds.Batching, record.Validator) (routing.IpfsRouting, error)

type DiscoveryOption func(context.Context, p2phost.Host) (discovery.Service, error)

var DHTOption RoutingOption = constructDHTRouting
var DHTClientOption RoutingOption = constructClientDHTRouting
var NilRouterOption RoutingOption = nilrouting.ConstructNilRouting
