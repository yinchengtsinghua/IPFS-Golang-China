
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//package命令实现IPFS命令接口
//
//使用github.com/ipfs/go-ipfs/commands定义命令行和http
//API。这是从外部使用IPF的人可以使用的接口
//Go语言。
package commands

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	e "github.com/ipfs/go-ipfs/core/commands/e"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	"gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

type commandEncoder struct {
	w io.Writer
}

func (e *commandEncoder) Encode(v interface{}) error {
	var (
		cmd *Command
		ok  bool
	)

	if cmd, ok = v.(*Command); !ok {
		return fmt.Errorf(`core/commands: uenxpected type %T, expected *"core/commands".Command`, v)
	}

	for _, s := range cmdPathStrings(cmd, cmd.showOpts) {
		_, err := e.w.Write([]byte(s + "\n"))
		if err != nil {
			return err
		}
	}

	return nil
}

type Command struct {
	Name        string
	Subcommands []Command
	Options     []Option

	showOpts bool
}

type Option struct {
	Names []string
}

const (
	flagsOptionName = "flags"
)

//commandscmd接受根命令，
//并返回一个命令，该命令列出该根目录中的子命令
func CommandsCmd(root *cmds.Command) *cmds.Command {
	return &cmds.Command{
		Helptext: cmdkit.HelpText{
			Tagline:          "List all available commands.",
			ShortDescription: `Lists all available commands (and subcommands) and exits.`,
		},
		Options: []cmdkit.Option{
			cmdkit.BoolOption(flagsOptionName, "f", "Show command flags"),
		},
		Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
			rootCmd := cmd2outputCmd("ipfs", root)
			rootCmd.showOpts, _ = req.Options[flagsOptionName].(bool)
			return cmds.EmitOnce(res, &rootCmd)
		},
		Encoders: cmds.EncoderMap{
			cmds.Text: func(req *cmds.Request) func(io.Writer) cmds.Encoder {
				return func(w io.Writer) cmds.Encoder { return &commandEncoder{w} }
			},
		},
		Type: Command{},
	}
}

func cmd2outputCmd(name string, cmd *cmds.Command) Command {
	opts := make([]Option, len(cmd.Options))
	for i, opt := range cmd.Options {
		opts[i] = Option{opt.Names()}
	}

	output := Command{
		Name:        name,
		Subcommands: make([]Command, 0, len(cmd.Subcommands)),
		Options:     opts,
	}

	for name, sub := range cmd.Subcommands {
		output.Subcommands = append(output.Subcommands, cmd2outputCmd(name, sub))
	}

	return output
}

func cmdPathStrings(cmd *Command, showOptions bool) []string {
	var cmds []string

	var recurse func(prefix string, cmd *Command)
	recurse = func(prefix string, cmd *Command) {
		newPrefix := prefix + cmd.Name
		cmds = append(cmds, newPrefix)
		if prefix != "" && showOptions {
			for _, options := range cmd.Options {
				var cmdOpts []string
				for _, flag := range options.Names {
					if len(flag) == 1 {
						flag = "-" + flag
					} else {
						flag = "--" + flag
					}
					cmdOpts = append(cmdOpts, newPrefix+" "+flag)
				}
				cmds = append(cmds, strings.Join(cmdOpts, " / "))
			}
		}
		for _, sub := range cmd.Subcommands {
			recurse(newPrefix+" ", &sub)
		}
	}

	recurse("", cmd)
	sort.Sort(sort.StringSlice(cmds))
	return cmds
}

//此处的更改也需要应用于
//-/DAG/dAG.Go
//-../object/object.go
//-../files/files.go
//—/unixfs/unixfs.go
func unwrapOutput(i interface{}) (interface{}, error) {
	var (
		ch <-chan interface{}
		ok bool
	)

	if ch, ok = i.(<-chan interface{}); !ok {
		return nil, e.TypeErr(ch, i)
	}

	return <-ch, nil
}

type nonFatalError string

//streamresult是一个辅助函数，用于流式处理可能
//包含非致命错误。允许helper函数死机
//内部错误。
func streamResult(procVal func(interface{}, io.Writer) nonFatalError) func(cmds.Response, cmds.ResponseEmitter) error {
	return func(res cmds.Response, re cmds.ResponseEmitter) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("internal error: %v", r)
			}
			re.Close()
		}()

		var errors bool
		for {
			v, err := res.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				return err
			}

			errorMsg := procVal(v, os.Stdout)

			if errorMsg != "" {
				errors = true
				fmt.Fprintf(os.Stderr, "%s\n", errorMsg)
			}
		}

		if errors {
			return fmt.Errorf("errors while displaying some entries")
		}
		return nil
	}
}
