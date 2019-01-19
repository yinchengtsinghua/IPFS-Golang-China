
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package nsopts

import (
	"time"
)

const (
//DefaultDepthLimit是解析所使用的默认深度限制。
	DefaultDepthLimit = 32

//UnlimitedDepth允许在解析中进行无限递归。你
//可能不想用这个，但如果你绝对想用的话，它就在这里。
//相信决议最终完成，不能占上风
//限制将要采取的步骤。
	UnlimitedDepth = 0
)

//resolveopts指定用于解析IPN路径的选项
type ResolveOpts struct {
//递归深度限制
	Depth uint
//要从DHT检索的IPN记录数
//（最佳记录从该集合中选择）
	DhtRecordCount uint
//等待提取DHT记录的时间量
//并进行了验证。零值表示没有显式
//超时（尽管由于拨号有一个隐式超时
//DHT内超时）
	DhtTimeout time.Duration
}

//DefaultResolveOpts返回用于解析的默认选项
//IPNS路径
func DefaultResolveOpts() ResolveOpts {
	return ResolveOpts{
		Depth:          DefaultDepthLimit,
		DhtRecordCount: 16,
		DhtTimeout:     time.Minute,
	}
}

//resolveopt用于设置选项
type ResolveOpt func(*ResolveOpts)

//深度是递归深度限制
func Depth(depth uint) ResolveOpt {
	return func(o *ResolveOpts) {
		o.Depth = depth
	}
}

//dhtrecordcount是要从DHT检索的IPN记录数。
func DhtRecordCount(count uint) ResolveOpt {
	return func(o *ResolveOpts) {
		o.DhtRecordCount = count
	}
}

//DHTTimeout是等待提取DHT记录的时间量。
//并进行了验证。零值表示没有显式超时
func DhtTimeout(timeout time.Duration) ResolveOpt {
	return func(o *ResolveOpts) {
		o.DhtTimeout = timeout
	}
}

//processopts将resolveOpt数组转换为resolveOpts对象
func ProcessOpts(opts []ResolveOpt) ResolveOpts {
	rsopts := DefaultResolveOpts()
	for _, option := range opts {
		option(&rsopts)
	}
	return rsopts
}
