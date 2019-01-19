
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	commands "github.com/ipfs/go-ipfs/commands"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

func ExternalBinary() *cmds.Command {
	return &cmds.Command{
		Arguments: []cmdkit.Argument{
			cmdkit.StringArg("args", false, true, "Arguments for subcommand."),
		},
		External: true,
		Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
			binname := strings.Join(append([]string{"ipfs"}, req.Path...), "-")
			_, err := exec.LookPath(binname)
			if err != nil {
//已卸载二进制文件的“--help”的特殊情况。
				for _, arg := range req.Arguments {
					if arg == "--help" || arg == "-h" {
						buf := new(bytes.Buffer)
						fmt.Fprintf(buf, "%s is an 'external' command.\n", binname)
						fmt.Fprintf(buf, "It does not currently appear to be installed.\n")
						fmt.Fprintf(buf, "Please refer to the ipfs documentation for instructions.\n")
						return res.Emit(buf)
					}
				}

				return fmt.Errorf("%s not installed", binname)
			}

			r, w := io.Pipe()

			cmd := exec.Command(binname, req.Arguments...)

//TODO:使命令库能够通过守护进程传递stdin
//cmd.stdin=请求stdin（）
			cmd.Stdin = io.LimitReader(nil, 0)
			cmd.Stdout = w
			cmd.Stderr = w

//子程序设置环境
			osenv := os.Environ()

//获取已经定义的节点iff。
			if cctx, ok := env.(*commands.Context); ok && cctx.Online {
				nd, err := cctx.GetNode()
				if err != nil {
					return fmt.Errorf("failed to start ipfs node: %s", err)
				}
				osenv = append(osenv, fmt.Sprintf("IPFS_ONLINE=%t", nd.OnlineMode()))
			}

			cmd.Env = osenv

			err = cmd.Start()
			if err != nil {
				return fmt.Errorf("failed to start subcommand: %s", err)
			}

			errC := make(chan error)

			go func() {
				var err error
				defer func() { errC <- err }()
				err = cmd.Wait()
				w.Close()
			}()

			err = res.Emit(r)
			if err != nil {
				return err
			}

			return <-errC
		},
	}
}
