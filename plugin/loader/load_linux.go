
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+建设！诺普洛金

package loader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"plugin"

	iplugin "github.com/ipfs/go-ipfs/plugin"
)

func init() {
	loadPluginsFunc = linuxLoadFunc
}

func linuxLoadFunc(pluginDir string) ([]iplugin.Plugin, error) {
	var plugins []iplugin.Plugin

	err := filepath.Walk(pluginDir, func(fi string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if fi != pluginDir {
				log.Warningf("found directory inside plugins directory: %s", fi)
			}
			return nil
		}

		if info.Mode().Perm()&0111 == 0 {
//文件不是可执行文件，我们不加载它
//这是为了防止加载插件，例如不可执行的插件
//安装，一些/tmp安装标记为安全
			log.Errorf("non-executable file in plugins directory: %s", fi)
			return nil
		}

		if newPlugins, err := loadPlugin(fi); err == nil {
			plugins = append(plugins, newPlugins...)
		} else {
			return fmt.Errorf("loading plugin %s: %s", fi, err)
		}
		return nil
	})

	return plugins, err
}

func loadPlugin(fi string) ([]iplugin.Plugin, error) {
	pl, err := plugin.Open(fi)
	if err != nil {
		return nil, err
	}
	pls, err := pl.Lookup("Plugins")
	if err != nil {
		return nil, err
	}

	typePls, ok := pls.(*[]iplugin.Plugin)
	if !ok {
		return nil, errors.New("filed 'Plugins' didn't contain correct type")
	}

	return *typePls, nil
}
