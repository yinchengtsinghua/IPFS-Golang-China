
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package loader

import (
	"fmt"
	"github.com/ipfs/go-ipfs/core/coredag"
	"github.com/ipfs/go-ipfs/plugin"
	"github.com/ipfs/go-ipfs/repo/fsrepo"
	"os"

	opentracing "gx/ipfs/QmWLWmRVSiagqP15jczsGME1qpob6HDbtbHAY2he9W5iUo/opentracing-go"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("plugin/loader")

var loadPluginsFunc = func(string) ([]plugin.Plugin, error) {
	return nil, nil
}

//PluginLoader跟踪加载的插件
type PluginLoader struct {
	plugins []plugin.Plugin
}

//NewPluginLoader创建新的插件加载程序
func NewPluginLoader(pluginDir string) (*PluginLoader, error) {
	plMap := make(map[string]plugin.Plugin)
	for _, v := range preloadPlugins {
		plMap[v.Name()] = v
	}

	if pluginDir != "" {
		newPls, err := loadDynamicPlugins(pluginDir)
		if err != nil {
			return nil, err
		}

		for _, pl := range newPls {
			if ppl, ok := plMap[pl.Name()]; ok {
//插件已预加载
				return nil, fmt.Errorf(
					"plugin: %s, is duplicated in version: %s, "+
						"while trying to load dynamically: %s",
					ppl.Name(), ppl.Version(), pl.Version())
			}
			plMap[pl.Name()] = pl
		}
	}

	loader := &PluginLoader{plugins: make([]plugin.Plugin, 0, len(plMap))}

	for _, v := range plMap {
		loader.plugins = append(loader.plugins, v)
	}

	return loader, nil
}

func loadDynamicPlugins(pluginDir string) ([]plugin.Plugin, error) {
	_, err := os.Stat(pluginDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return loadPluginsFunc(pluginDir)
}

//初始化所有加载的插件
func (loader *PluginLoader) Initialize() error {
	for _, p := range loader.plugins {
		err := p.Init()
		if err != nil {
			return err
		}
	}

	return nil
}

//运行插件
func (loader *PluginLoader) Run() error {
	for _, pl := range loader.plugins {
		switch pl := pl.(type) {
		case plugin.PluginIPLD:
			err := runIPLDPlugin(pl)
			if err != nil {
				return err
			}
		case plugin.PluginTracer:
			err := runTracerPlugin(pl)
			if err != nil {
				return err
			}
		case plugin.PluginDatastore:
			err := fsrepo.AddDatastoreConfigHandler(pl.DatastoreTypeName(), pl.DatastoreConfigParser())
			if err != nil {
				return err
			}
		default:
			panic(pl)
		}
	}
	return nil
}

func runIPLDPlugin(pl plugin.PluginIPLD) error {
	err := pl.RegisterBlockDecoders(ipld.DefaultBlockDecoder)
	if err != nil {
		return err
	}
	return pl.RegisterInputEncParsers(coredag.DefaultInputEncParsers)
}

func runTracerPlugin(pl plugin.PluginTracer) error {
	tracer, err := pl.InitTracer()
	if err != nil {
		return err
	}
	opentracing.SetGlobalTracer(tracer)
	return nil
}
