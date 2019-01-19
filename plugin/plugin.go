
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package plugin

//插件是各种go ipfs插件的基本接口
//它将包含在不同插件的接口中
type Plugin interface {
//名称应返回插件的唯一名称
	Name() string
//版本返回插件的当前版本
	Version() string
//加载插件时调用一次init
	Init() error
}
