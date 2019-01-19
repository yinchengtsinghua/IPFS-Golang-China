
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//包iface定义ipfs核心api，这是一组用于
//与IPFS节点交互。
package iface

import (
	"context"

	"github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
)

//coreapi为go程序定义到ipfs的统一接口
type CoreAPI interface {
//unixfs返回unixfs API的实现
	Unixfs() UnixfsAPI

//block返回block api的实现
	Block() BlockAPI

//DAG返回DAG API的实现
	Dag() DagAPI

//name返回name api的实现
	Name() NameAPI

//key返回key api的实现
	Key() KeyAPI

//pin返回pin api的实现
	Pin() PinAPI

//object api返回对象api的实现
	Object() ObjectAPI

//DHT返回DHT API的实现
	Dht() DhtAPI

//Swarm返回Swarm API的实现
	Swarm() SwarmAPI

//pubsub返回pubsub API的实现
	PubSub() PubSubAPI

//resolvepath使用unixfs resolver解析路径
	ResolvePath(context.Context, Path) (ResolvedPath, error)

//resolvenode使用unixfs解析路径（如果尚未解析）
//解析程序，获取并返回解析的节点
	ResolveNode(context.Context, Path) (ipld.Node, error)

//WITHOPTIONS基于此实例创建coreapi的新实例
//应用的一组选项
	WithOptions(...options.ApiOption) (CoreAPI, error)
}
