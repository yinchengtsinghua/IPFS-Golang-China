
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package cmdenv

import (
	"fmt"
	"strings"

	"github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("core/commands/cmdenv")

//getnode从环境中提取节点。
func GetNode(env interface{}) (*core.IpfsNode, error) {
	ctx, ok := env.(*commands.Context)
	if !ok {
		return nil, fmt.Errorf("expected env to be of type %T, got %T", ctx, env)
	}

	return ctx.GetNode()
}

//getapi从环境中提取coreapi实例。
func GetApi(env cmds.Environment, req *cmds.Request) (coreiface.CoreAPI, error) {
	ctx, ok := env.(*commands.Context)
	if !ok {
		return nil, fmt.Errorf("expected env to be of type %T, got %T", ctx, env)
	}

	offline, _ := req.Options["offline"].(bool)
	if !offline {
		offline, _ = req.Options["local"].(bool)
		if offline {
			log.Errorf("Command '%s', --local is deprecated, use --offline instead", strings.Join(req.Path, " "))
		}
	}
	api, err := ctx.GetAPI()
	if err != nil {
		return nil, err
	}
	if offline {
		return api.WithOptions(options.Api.Offline(offline))
	}

	return api, nil
}

//getconfig从环境中提取配置。
func GetConfig(env cmds.Environment) (*config.Config, error) {
	ctx, ok := env.(*commands.Context)
	if !ok {
		return nil, fmt.Errorf("expected env to be of type %T, got %T", ctx, env)
	}

	return ctx.GetConfig()
}

//
func GetConfigRoot(env cmds.Environment) (string, error) {
	ctx, ok := env.(*commands.Context)
	if !ok {
		return "", fmt.Errorf("expected env to be of type %T, got %T", ctx, env)
	}

	return ctx.ConfigRoot, nil
}
