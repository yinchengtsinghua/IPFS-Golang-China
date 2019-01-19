
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package commands

import (
	"fmt"
	"io"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	lwriter "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log/writer"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

//golang os.args重写*并将字符参数替换为
//包含用户的CWD中的每个文件的数组。作为一个
//解决方法，我们用“全部”代替。Util库仍然使用*so
//我们在这一步转换它。
var logAllKeyword = "all"

var LogCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with the daemon log output.",
		ShortDescription: `
'ipfs log' contains utility commands to affect or read the logging
output of a running daemon.
`,
	},

	Subcommands: map[string]*cmds.Command{
		"level": logLevelCmd,
		"ls":    logLsCmd,
		"tail":  logTailCmd,
	},
}

var logLevelCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Change the logging level.",
		ShortDescription: `
Change the verbosity of one or all subsystems log output. This does not affect
the event log.
`,
	},

	Arguments: []cmdkit.Argument{
//todo对“all”使用不同的关键字，因为理论上所有的关键字都可以
//与子系统名称冲突
		cmdkit.StringArg("subsystem", true, false, fmt.Sprintf("The subsystem logging identifier. Use '%s' for all subsystems.", logAllKeyword)),
		cmdkit.StringArg("level", true, false, `The log level, with 'debug' the most verbose and 'critical' the least verbose.
			One of: debug, info, warning, error, critical.
		`),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		args := req.Arguments
		subsystem, level := args[0], args[1]

		if subsystem == logAllKeyword {
			subsystem = "*"
		}

		if err := logging.SetLogLevel(subsystem, level); err != nil {
			return err
		}

		s := fmt.Sprintf("Changed log level of '%s' to '%s'\n", subsystem, level)
		log.Info(s)

		return cmds.EmitOnce(res, &MessageOutput{s})
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, out *MessageOutput) error {
			fmt.Fprint(w, out.Message)
			return nil
		}),
	},
	Type: MessageOutput{},
}

var logLsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "List the logging subsystems.",
		ShortDescription: `
'ipfs log ls' is a utility command used to list the logging
subsystems of a running daemon.
`,
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		return cmds.EmitOnce(res, &stringList{logging.GetSubsystems()})
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, list *stringList) error {
			for _, s := range list.Strings {
				fmt.Fprintln(w, s)
			}
			return nil
		}),
	},
	Type: stringList{},
}

var logTailCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Read the event log.",
		ShortDescription: `
Outputs event log messages (not other log messages) as they are generated.
`,
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		ctx := req.Context
		r, w := io.Pipe()
		go func() {
			defer w.Close()
			<-ctx.Done()
		}()
		lwriter.WriterGroup.AddWriter(w)
		return res.Emit(r)
	},
}
