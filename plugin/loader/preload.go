
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package loader

import (
	"github.com/ipfs/go-ipfs/plugin"
	pluginbadgerds "github.com/ipfs/go-ipfs/plugin/plugins/badgerds"
	pluginflatfs "github.com/ipfs/go-ipfs/plugin/plugins/flatfs"
	pluginipldgit "github.com/ipfs/go-ipfs/plugin/plugins/git"
	pluginlevelds "github.com/ipfs/go-ipfs/plugin/plugins/levelds"
)

//不编辑此文件
//此文件正在作为插件生成过程的一部分生成
//要更改它，请修改plugin/loader/preload.sh

var preloadPlugins = []plugin.Plugin{
	pluginipldgit.Plugins[0],
	pluginbadgerds.Plugins[0],
	pluginflatfs.Plugins[0],
	pluginlevelds.Plugins[0],
}
