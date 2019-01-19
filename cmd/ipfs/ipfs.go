
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"fmt"

	commands "github.com/ipfs/go-ipfs/core/commands"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
)

//这是cli根目录，用于执行cli客户机可访问的命令。
//某些子命令（如'ipfs daemon'或'ipfs init'）只能在此处访问，
//无法通过HTTP API调用。
var Root = &cmds.Command{
	Options:  commands.Root.Options,
	Helptext: commands.Root.Helptext,
}

//commandsClientCmd是本地CLI的“ipfs命令”命令
var commandsClientCmd = commands.CommandsCmd(Root)

//localcommands中的命令应始终在本地运行（即使守护进程正在运行）。
//它们可以通过定义同名的子命令来重写commands.root中的子命令。
var localCommands = map[string]*cmds.Command{
	"daemon":   daemonCmd,
	"init":     initCmd,
	"commands": commandsClientCmd,
}

func init() {
//在此处设置而不是在文本中设置以防止初始化循环
//（有些命令引用根目录）
	Root.Subcommands = localCommands

	for k, v := range commands.Root.Subcommands {
		if _, found := Root.Subcommands[k]; !found {
			Root.Subcommands[k] = v
		}
	}
}

//注：必要时，用负片描述属性，以便
//提供所需的默认值
type cmdDetails struct {
	cannotRunOnClient bool
	cannotRunOnDaemon bool
	doesNotUseRepo    bool

//doesnotuseconfigasinput将不使用配置的命令描述为
//输入。这些命令初始化配置或执行操作
//不需要访问配置。
//
//必须先运行需要配置的预命令挂钩，然后才能运行这些挂钩
//命令。
	doesNotUseConfigAsInput bool

//preemptsautoupdate描述必须在没有
//自动更新预命令挂钩
	preemptsAutoUpdate bool
}

func (d *cmdDetails) String() string {
	return fmt.Sprintf("on client? %t, on daemon? %t, uses repo? %t",
		d.canRunOnClient(), d.canRunOnDaemon(), d.usesRepo())
}

func (d *cmdDetails) Loggable() map[string]interface{} {
	return map[string]interface{}{
		"canRunOnClient":     d.canRunOnClient(),
		"canRunOnDaemon":     d.canRunOnDaemon(),
		"preemptsAutoUpdate": d.preemptsAutoUpdate,
		"usesConfigAsInput":  d.usesConfigAsInput(),
		"usesRepo":           d.usesRepo(),
	}
}

func (d *cmdDetails) usesConfigAsInput() bool { return !d.doesNotUseConfigAsInput }
func (d *cmdDetails) canRunOnClient() bool    { return !d.cannotRunOnClient }
func (d *cmdDetails) canRunOnDaemon() bool    { return !d.cannotRunOnDaemon }
func (d *cmdDetails) usesRepo() bool          { return !d.doesNotUseRepo }

//“这是什么疯狂！“你问。我们的命令有一个不幸的问题
//无法在所有相同的上下文上运行。这张地图描述了这些
//属性，以便其他代码可以决定是否调用
//命令或向用户返回错误。
var cmdDetailsMap = map[string]cmdDetails{
	"init":        {doesNotUseConfigAsInput: true, cannotRunOnDaemon: true, doesNotUseRepo: true},
	"daemon":      {doesNotUseConfigAsInput: true, cannotRunOnDaemon: true},
	"commands":    {doesNotUseRepo: true},
"version":     {doesNotUseConfigAsInput: true, doesNotUseRepo: true}, //必须允许在init之前运行
	"log":         {cannotRunOnClient: true},
	"diag/cmds":   {cannotRunOnClient: true},
	"repo/fsck":   {cannotRunOnDaemon: true},
	"config/edit": {cannotRunOnDaemon: true, doesNotUseRepo: true},
	"cid":         {doesNotUseRepo: true},
}
