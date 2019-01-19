
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package flatfs

import (
	"fmt"
	"path/filepath"

	"github.com/ipfs/go-ipfs/plugin"
	"github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/fsrepo"

	flatfs "gx/ipfs/QmfZuoe973XBPt4AUQEQtcj3XhycT3cGufFnivLfpWUxMt/go-ds-flatfs"
)

//插件导出将加载的插件列表
var Plugins = []plugin.Plugin{
	&flatfsPlugin{},
}

type flatfsPlugin struct{}

var _ plugin.PluginDatastore = (*flatfsPlugin)(nil)

func (*flatfsPlugin) Name() string {
	return "ds-flatfs"
}

func (*flatfsPlugin) Version() string {
	return "0.1.0"
}

func (*flatfsPlugin) Init() error {
	return nil
}

func (*flatfsPlugin) DatastoreTypeName() string {
	return "flatfs"
}

type datastoreConfig struct {
	path      string
	shardFun  *flatfs.ShardIdV1
	syncField bool
}

//badgerdsdatastoreconfig返回badger数据存储区的配置存根
//从给定的参数
func (*flatfsPlugin) DatastoreConfigParser() fsrepo.ConfigFromMap {
	return func(params map[string]interface{}) (fsrepo.DatastoreConfig, error) {
		var c datastoreConfig
		var ok bool
		var err error

		c.path, ok = params["path"].(string)
		if !ok {
			return nil, fmt.Errorf("'path' field is missing or not boolean")
		}

		sshardFun, ok := params["shardFunc"].(string)
		if !ok {
			return nil, fmt.Errorf("'shardFunc' field is missing or not a string")
		}
		c.shardFun, err = flatfs.ParseShardFunc(sshardFun)
		if err != nil {
			return nil, err
		}

		c.syncField, ok = params["sync"].(bool)
		if !ok {
			return nil, fmt.Errorf("'sync' field is missing or not boolean")
		}
		return &c, nil
	}
}

func (c *datastoreConfig) DiskSpec() fsrepo.DiskSpec {
	return map[string]interface{}{
		"type":      "flatfs",
		"path":      c.path,
		"shardFunc": c.shardFun.String(),
	}
}

func (c *datastoreConfig) Create(path string) (repo.Datastore, error) {
	p := c.path
	if !filepath.IsAbs(p) {
		p = filepath.Join(path, p)
	}

	return flatfs.CreateOrOpen(p, c.shardFun, c.syncField)
}
