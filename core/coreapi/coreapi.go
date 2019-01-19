
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
包coreapi提供对ipfs中核心命令的直接访问。如果你是
将IPF直接嵌入Go程序中，此包是公共的
您应该用来读写文件或控制IPF的接口。

如果将IPF作为单独的进程运行，则应使用“go ipfs api”to
通过HTTP使用它。当我们在这里完成接口时，`go ipfs api`将
透明地采用它们，这样您就可以对任何一个包使用相同的代码。

**注：此软件包是实验性的。**主要开发了“Go IPF”
作为一个独立的应用程序和库样式，这个包的使用仍然是新的。
这里的接口还不完全稳定。
**/

package coreapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/ipfs/go-ipfs/core"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	"github.com/ipfs/go-ipfs/core/coreapi/interface/options"
	"github.com/ipfs/go-ipfs/namesys"
	"github.com/ipfs/go-ipfs/pin"
	"github.com/ipfs/go-ipfs/repo"

	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	"gx/ipfs/QmP2g3VxmC7g7fyRJDj1VJ72KHZbJ9UW24YjSWEj1XTb4H/go-ipfs-exchange-interface"
	pstore "gx/ipfs/QmPiemjiKBC9VA7vZF82m4x1oygtg2c2YVqag8PX7dN1BD/go-libp2p-peerstore"
	"gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	"gx/ipfs/QmTiRqrF5zkdZyrdsL5qndG1UbeWi8k8N2pYxCtXWrahR2/go-libp2p-routing"
	pubsub "gx/ipfs/QmVRxA4J3UPQpw74dLrQ6NJkfysCA1H4GU28gVpXQt9zMU/go-libp2p-pubsub"
	offlineroute "gx/ipfs/QmVZ6cQXHoTQja4oo9GhhHZi7dThi4x98mRKgGtKnTy37u/go-ipfs-routing/offline"
	"gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	bserv "gx/ipfs/QmYPZzd9VqmJDwxUnThfeSbV1Y5o53aVPDijTB7j7rS9Ep/go-blockservice"
	offlinexch "gx/ipfs/QmYZwey1thDTynSrvd6qQkX24UpTka6TFhQ2v569UpoqxD/go-ipfs-exchange-offline"
	p2phost "gx/ipfs/QmaoXrM4Z41PD48JY36YqQGKQpLGjyLA2cKcLsES7YddAq/go-libp2p-host"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	record "gx/ipfs/QmfARXVCzpwFXQdepAJZuqyNDgV9doEsMnVCo1ssmuSe1U/go-libp2p-record"
)

var log = logging.Logger("core/coreapi")

type CoreAPI struct {
	nctx context.Context

	identity   peer.ID
	privateKey ci.PrivKey

	repo       repo.Repo
	blockstore blockstore.GCBlockstore
	baseBlocks blockstore.Blockstore
	pinning    pin.Pinner

	blocks bserv.BlockService
	dag    ipld.DAGService

	peerstore       pstore.Peerstore
	peerHost        p2phost.Host
	recordValidator record.Validator
	exchange        exchange.Interface

	namesys namesys.NameSystem
	routing routing.IpfsRouting

	pubSub *pubsub.PubSub

	checkPublishAllowed func() error
	checkOnline         func(allowOffline bool) error

//仅用于在WITHOPTIONS中重新应用选项，不要在其他任何地方使用
	nd         *core.IpfsNode
	parentOpts options.ApiSettings
}

//newcarapi创建由go ipfs节点支持的ipfs coreapi的新实例。
func NewCoreAPI(n *core.IpfsNode, opts ...options.ApiOption) (coreiface.CoreAPI, error) {
	parentOpts, err := options.ApiOptions()
	if err != nil {
		return nil, err
	}

	return (&CoreAPI{nd: n, parentOpts: *parentOpts}).WithOptions(opts...)
}

//unixfs返回由go ipfs节点支持的unixfsapi接口实现
func (api *CoreAPI) Unixfs() coreiface.UnixfsAPI {
	return (*UnixfsAPI)(api)
}

//block返回由go ipfs节点支持的blockapi接口实现
func (api *CoreAPI) Block() coreiface.BlockAPI {
	return (*BlockAPI)(api)
}

//DAG返回由go ipfs节点支持的DAGAPI接口实现
func (api *CoreAPI) Dag() coreiface.DagAPI {
	return (*DagAPI)(api)
}

//name返回由go ipfs节点支持的nameapi接口实现
func (api *CoreAPI) Name() coreiface.NameAPI {
	return (*NameAPI)(api)
}

//key返回由go ipfs节点支持的keyapi接口实现
func (api *CoreAPI) Key() coreiface.KeyAPI {
	return (*KeyAPI)(api)
}

//对象返回由go ipfs节点支持的objectapi接口实现
func (api *CoreAPI) Object() coreiface.ObjectAPI {
	return (*ObjectAPI)(api)
}

//pin返回由go ipfs节点支持的pinapi接口实现
func (api *CoreAPI) Pin() coreiface.PinAPI {
	return (*PinAPI)(api)
}

//DHT返回由go ipfs节点支持的DHTAPI接口实现
func (api *CoreAPI) Dht() coreiface.DhtAPI {
	return (*DhtAPI)(api)
}

//swarm返回由go ipfs节点支持的swarmapi接口实现
func (api *CoreAPI) Swarm() coreiface.SwarmAPI {
	return (*SwarmAPI)(api)
}

//pubsub返回由go ipfs节点支持的pubsubapi接口实现
func (api *CoreAPI) PubSub() coreiface.PubSubAPI {
	return (*PubSubAPI)(api)
}

//WITHOPTIONS返回应用了全局选项的API
func (api *CoreAPI) WithOptions(opts ...options.ApiOption) (coreiface.CoreAPI, error) {
settings := api.parentOpts //一定要复制
	_, err := options.ApiOptionsTo(&settings, opts...)
	if err != nil {
		return nil, err
	}

	if api.nd == nil {
		return nil, errors.New("cannot apply options to api without node")
	}

	n := api.nd

	subApi := &CoreAPI{
		nctx: n.Context(),

		identity:   n.Identity,
		privateKey: n.PrivateKey,

		repo:       n.Repo,
		blockstore: n.Blockstore,
		baseBlocks: n.BaseBlocks,
		pinning:    n.Pinning,

		blocks: n.Blocks,
		dag:    n.DAG,

		peerstore:       n.Peerstore,
		peerHost:        n.PeerHost,
		namesys:         n.Namesys,
		recordValidator: n.RecordValidator,
		exchange:        n.Exchange,
		routing:         n.Routing,

		pubSub: n.PubSub,

		nd:         n,
		parentOpts: settings,
	}

	subApi.checkOnline = func(allowOffline bool) error {
		if !n.OnlineMode() && !allowOffline {
			return coreiface.ErrOffline
		}
		return nil
	}

	subApi.checkPublishAllowed = func() error {
		if n.Mounts.Ipns != nil && n.Mounts.Ipns.IsActive() {
			return errors.New("cannot manually publish while IPNS is mounted")
		}
		return nil
	}

	if settings.Offline {
		cfg, err := n.Repo.Config()
		if err != nil {
			return nil, err
		}

		cs := cfg.Ipns.ResolveCacheSize
		if cs == 0 {
			cs = core.DefaultIpnsCacheSize
		}
		if cs < 0 {
			return nil, fmt.Errorf("cannot specify negative resolve cache size")
		}

		subApi.routing = offlineroute.NewOfflineRouter(subApi.repo.Datastore(), subApi.recordValidator)
		subApi.namesys = namesys.NewNameSystem(subApi.routing, subApi.repo.Datastore(), cs)

		subApi.peerstore = nil
		subApi.peerHost = nil
		subApi.recordValidator = nil
	}

	if settings.Offline || !settings.FetchBlocks {
		subApi.exchange = offlinexch.Exchange(subApi.blockstore)
		subApi.blocks = bserv.New(subApi.blockstore, subApi.exchange)
		subApi.dag = dag.NewDAGService(subApi.blocks)
	}

	return subApi, nil
}

//GetSession返回由具有只读会话DAG的同一节点支持的新API
func (api *CoreAPI) getSession(ctx context.Context) *CoreAPI {
	sesApi := *api

//TODO:我们也可以将此应用于api.blocks，并组合成可写的api，
//但这需要对blockservice/merkledag进行一些更改
	sesApi.dag = dag.NewReadOnlyDagService(dag.NewSession(ctx, api.dag))

	return &sesApi
}
