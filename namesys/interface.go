
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
包名称sys为IPF实现解析程序和发布服务器
命名系统（IPN）。

IPF的核心是一个不可变的、内容可寻址的Merkle图。
这对许多用例都很有效，但不允许您回答
像“爱丽丝现在的主页是什么？”.可变名称
系统允许Alice发布以下信息：

  alice.example.com的当前主页是
  /ipfs/qmcqtw8ffrvsbarmbwwhxt3auysbhjlcvmfyi3lbc4xnwj

或：

  节点的当前主页
  qmatme9mssfkxofffphwnlkgwzg8et9bud6yopab52vpy
  是
  /ipfs/qmcqtw8ffrvsbarmbwwhxt3auysbhjlcvmfyi3lbc4xnwj

可变名称系统还允许用户解析这些引用
查找给定对象当前引用的不可变IPFS对象
可变名称

有关此功能的命令行绑定，请参阅：

  IPFS名称
  IPFS DNS
  IPFS解决方案
**/

package namesys

import (
	"errors"
	"time"

	context "context"

	opts "github.com/ipfs/go-ipfs/namesys/opts"
	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"

	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
)

//errResolveFailed在尝试解决时发出错误信号。
var ErrResolveFailed = errors.New("could not resolve name")

//errResolverExecution发出递归深度限制信号。
var ErrResolveRecursion = errors.New(
	"could not resolve name (recursion limit exceeded)")

//errpublishfailed在尝试发布时发出错误信号。
var ErrPublishFailed = errors.New("could not publish name")

//namesys表示一个内聚的名称发布和解析系统。
//
//发布名称是建立映射、键值的过程
//根据命名规则和数据库配对。
//
//解析名称是查找与
//密钥（名称）。
type NameSystem interface {
	Resolver
	Publisher
}

//result是resolver.resolveasync的返回类型。
type Result struct {
	Path path.Path
	Err  error
}

//解析器是一个能够解析名称的对象。
type Resolver interface {

//解析执行递归查找，返回未引用的
//路径。例如，如果ipfs.io有一个指向
///ipns/qmatme9mssfkxofffphwnlkgwzg8et9bud6yopab52vpy
//还有一个DHT IPN条目
//qmatme9mssfkxofffphwnlkgwzg8et9bud6yopab52vpy
//->/ipfs/qmcqtw8ffrvsbarmbwwhxt3auysbhjlcvmfyi3lbc4xnwj
//然后
//解析（ctx，“/ipns/ipfs.io”）
//将解析两个名称，返回
///ipfs/qmcqtw8ffrvsbarmbwwhxt3auysbhjlcvmfyi3lbc4xnwj
//
//有一个默认的深度限制来避免无限递归。大多数
//用户可以使用此默认限制，但如果需要
//调整可以指定为选项的限制。
	Resolve(ctx context.Context, name string, options ...opts.ResolveOpt) (value path.Path, err error)

//resolveasync执行递归名称查找，如resolve，但它返回
//在DHT中发现的条目。保证每次返回结果
//比前一个更好（通常意味着更新）。
	ResolveAsync(ctx context.Context, name string, options ...opts.ResolveOpt) <-chan Result
}

//Publisher是一个能够发布特定名称的对象。
type Publisher interface {

//发布建立名称值映射。
//要使其不特定于私钥。
	Publish(ctx context.Context, name ci.PrivKey, value path.Path) error

//TODO:将被更通用的“publishewithvalidity”类型替换
//实现记录规范后调用
	PublishWithEOL(ctx context.Context, name ci.PrivKey, value path.Path, eol time.Time) error
}
