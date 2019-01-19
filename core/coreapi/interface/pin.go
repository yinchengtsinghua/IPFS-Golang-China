
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
)

//pin保存有关pinned资源的信息
type Pin interface {
//固定对象的路径
	Path() ResolvedPath

//引脚类型
	Type() string
}

//pinstatus保存有关pin运行状况的信息
type PinStatus interface {
//OK指示是否已验证PIN正确。
	Ok() bool

//bad nodes从pin返回任何坏（通常丢失）节点
	BadNodes() []BadPinNode
}

//bad pin node是一个被pin标记为坏的节点。请验证
type BadPinNode interface {
//路径是节点的路径
	Path() ResolvedPath

//err是将节点标记为坏节点的原因
	Err() error
}

//pinapi指定Pining的接口
type PinAPI interface {
//添加创建新的pin，默认为递归-固定整个引用的pin
//树
	Add(context.Context, Path, ...options.PinAddOption) error

//ls返回此节点上固定对象的列表
	Ls(context.Context, ...options.PinLsOption) ([]Pin, error)

//rm删除由路径指定的对象的pin
	Rm(context.Context, Path) error

//更新将一个管脚更改为另一个管脚，跳过对中匹配路径的检查
//老树
	Update(ctx context.Context, from Path, to Path, opts ...options.PinUpdateOption) error

//验证验证固定对象的完整性
	Verify(context.Context) (<-chan PinStatus, error)
}
