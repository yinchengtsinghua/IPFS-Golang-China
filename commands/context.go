
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package commands

import (
	"context"
	"errors"
	"strings"
	"time"

	core "github.com/ipfs/go-ipfs/core"
	coreapi "github.com/ipfs/go-ipfs/core/coreapi"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"
	loader "github.com/ipfs/go-ipfs/plugin/loader"

	"gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("command")

//上下文表示请求上下文
type Context struct {
	Online     bool
	ConfigRoot string
	ReqLog     *ReqLog

	Plugins *loader.PluginLoader

	config     *config.Config
	LoadConfig func(path string) (*config.Config, error)

	Gateway       bool
	api           coreiface.CoreAPI
	node          *core.IpfsNode
	ConstructNode func() (*core.IpfsNode, error)
}

//getconfig返回当前命令执行的配置
//语境。它可以使用提供的功能加载它。
func (c *Context) GetConfig() (*config.Config, error) {
	var err error
	if c.config == nil {
		if c.LoadConfig == nil {
			return nil, errors.New("nil LoadConfig function")
		}
		c.config, err = c.LoadConfig(c.ConfigRoot)
	}
	return c.config, err
}

//getnode返回当前命令执行的节点
//语境。它可以用提供的函数构造它。
func (c *Context) GetNode() (*core.IpfsNode, error) {
	var err error
	if c.node == nil {
		if c.ConstructNode == nil {
			return nil, errors.New("nil ConstructNode function")
		}
		c.node, err = c.ConstructNode()
	}
	return c.node, err
}

//getapi返回由ipfs节点支持的coreapi实例。
//它可以用提供的函数构造节点
func (c *Context) GetAPI() (coreiface.CoreAPI, error) {
	if c.api == nil {
		n, err := c.GetNode()
		if err != nil {
			return nil, err
		}
		fetchBlocks := true
		if c.Gateway {
			cfg, err := c.GetConfig()
			if err != nil {
				return nil, err
			}
			fetchBlocks = !cfg.Gateway.NoFetch
		}

		c.api, err = coreapi.NewCoreAPI(n, options.Api.FetchBlocks(fetchBlocks))
		if err != nil {
			return nil, err
		}
	}
	return c.api, nil
}

//Context返回节点的上下文。
func (c *Context) Context() context.Context {
	n, err := c.GetNode()
	if err != nil {
		log.Debug("error getting node: ", err)
		return context.Background()
	}

	return n.Context()
}

//log request将传递的请求添加到请求日志中，然后
//返回请求时应调用的函数
//一生都结束了。
func (c *Context) LogRequest(req *cmds.Request) func() {
	rle := &ReqLogEntry{
		StartTime: time.Now(),
		Active:    true,
		Command:   strings.Join(req.Path, "/"),
		Options:   req.Options,
		Args:      req.Arguments,
		ID:        c.ReqLog.nextID,
		log:       c.ReqLog,
	}
	c.ReqLog.AddEntry(rle)

	return func() {
		c.ReqLog.Finish(rle)
	}
}

//关闭将清除应用程序状态。
func (c *Context) Close() {
//让我们不要忘记眼泪。如果一个节点被初始化，我们必须关闭它。
//注意，这意味着基础req.context（）.node变量是公开的。
//这很粗糙，在提取exec上下文时应该进行更改。
	if c.node != nil {
		log.Info("Shutting down node...")
		c.node.Close()
	}
}
