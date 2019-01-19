
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

	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	"gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
)

//key指定keyapi keystore中键的接口
type Key interface {
//键返回键名称
	Name() string

//路径返回键路径
	Path() Path

//id返回密钥peerid
	ID() peer.ID
}

//keyapi指定到keystore的接口
type KeyAPI interface {
//生成生成新密钥，并将其存储在指定的密钥库中
//名称并返回其公钥的base58编码多哈希
	Generate(ctx context.Context, name string, opts ...options.KeyGenerateOption) (Key, error)

//重命名将oldname键重命名为newname。返回键以及是否另一个键
//密钥被覆盖，或出现错误
	Rename(ctx context.Context, oldName string, newName string, opts ...options.KeyRenameOption) (Key, bool, error)

//列表列出存储在密钥库中的密钥
	List(ctx context.Context) ([]Key, error)

//self返回“main”节点键
	Self(ctx context.Context) (Key, error)

//移除从密钥库中移除密钥。返回已删除密钥的IPN路径
	Remove(ctx context.Context, name string) (Key, error)
}
