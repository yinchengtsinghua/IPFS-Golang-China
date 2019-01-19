
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package commands

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	cmdenv "github.com/ipfs/go-ipfs/core/commands/cmdenv"
	ncmd "github.com/ipfs/go-ipfs/core/commands/name"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"
	ns "github.com/ipfs/go-ipfs/namesys"
	nsopts "github.com/ipfs/go-ipfs/namesys/opts"
	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

const (
	resolveRecursiveOptionName      = "recursive"
	resolveDhtRecordCountOptionName = "dht-record-count"
	resolveDhtTimeoutOptionName     = "dht-timeout"
)

var ResolveCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Resolve the value of names to IPFS.",
		ShortDescription: `
There are a number of mutable name protocols that can link among
themselves and into IPNS. This command accepts any of these
identifiers and resolves them to the referenced item.
`,
		LongDescription: `
There are a number of mutable name protocols that can link among
themselves and into IPNS. For example IPNS references can (currently)
point at an IPFS object, and DNS links can point at other DNS links, IPNS
entries, or IPFS objects. This command accepts any of these
identifiers and resolves them to the referenced item.

EXAMPLES

Resolve the value of your identity:

  $ ipfs resolve /ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy
  /ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj

Resolve the value of another name:

  $ ipfs resolve /ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n
  /ipns/QmatmE9msSfkKxoffpHwNLNKgwZG8eT9Bud6YoPab52vpy

Resolve the value of another name recursively:

  $ ipfs resolve -r /ipns/QmbCMUZw6JFeZ7Wp9jkzbye3Fzp2GGcPgC3nmeUjfVF87n
  /ipfs/Qmcqtw8FfrVSBaRmbWwHxt3AuySBhJLcvmFYi3Lbc4xnwj

Resolve the value of an IPFS DAG path:

  $ ipfs resolve /ipfs/QmeZy1fGbwgVSrqbfh9fKQrAWgeyRnj7h8fsHS1oy3k99x/beep/boop
  /ipfs/QmYRMjyvAiHKN9UTi8Bzt1HUspmSRD8T8DwxfSMzLgBon1

`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("name", true, false, "The name to resolve.").EnableStdin(),
	},
	Options: []cmdkit.Option{
		cmdkit.BoolOption(resolveRecursiveOptionName, "r", "Resolve until the result is an IPFS name."),
		cmdkit.IntOption(resolveDhtRecordCountOptionName, "dhtrc", "Number of records to request for DHT resolution."),
		cmdkit.StringOption(resolveDhtTimeoutOptionName, "dhtt", "Max time to collect values during DHT resolution eg \"30s\". Pass 0 for no timeout."),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		api, err := cmdenv.GetApi(env, req)
		if err != nil {
			return err
		}

		name := req.Arguments[0]
		recursive, _ := req.Options[resolveRecursiveOptionName].(bool)

//IPN逐步解决的情况
		if strings.HasPrefix(name, "/ipns/") && !recursive {
			rc, rcok := req.Options[resolveDhtRecordCountOptionName].(uint)
			dhtt, dhttok := req.Options[resolveDhtTimeoutOptionName].(string)
			ropts := []options.NameResolveOption{
				options.Name.ResolveOption(nsopts.Depth(1)),
			}

			if rcok {
				ropts = append(ropts, options.Name.ResolveOption(nsopts.DhtRecordCount(rc)))
			}
			if dhttok {
				d, err := time.ParseDuration(dhtt)
				if err != nil {
					return err
				}
				if d < 0 {
					return errors.New("DHT timeout value must be >= 0")
				}
				ropts = append(ropts, options.Name.ResolveOption(nsopts.DhtTimeout(d)))
			}
			p, err := api.Name().Resolve(req.Context, name, ropts...)
//err分解器执行正常
			if err != nil && err != ns.ErrResolveRecursion {
				return err
			}
			return cmds.EmitOnce(res, &ncmd.ResolvedPath{Path: path.Path(p.String())})
		}

//
		p, err := coreiface.ParsePath(name)
		if err != nil {
			return err
		}

		rp, err := api.ResolvePath(req.Context, p)
		if err != nil {
			return err
		}

		if rp.Remainder() != "" {
//TODO:JS需要此错误。而不是修复这个
//错误，我们应该修复5703。
			return fmt.Errorf("found non-link at given path")
		}

		return cmds.EmitOnce(res, &ncmd.ResolvedPath{Path: path.Path("/" + rp.Namespace() + "/" + rp.Cid().String())})
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, rp *ncmd.ResolvedPath) error {
			fmt.Fprintln(w, rp.Path.String())
			return nil
		}),
	},
	Type: ncmd.ResolvedPath{},
}
