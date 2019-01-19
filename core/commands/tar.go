
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

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/commands/cmdenv"
	tar "github.com/ipfs/go-ipfs/tar"

	"gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	"gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	"gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

var TarCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Utility functions for tar files in ipfs.",
	},

	Subcommands: map[string]*cmds.Command{
		"add": tarAddCmd,
		"cat": tarCatCmd,
	},
}

var tarAddCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Import a tar file into ipfs.",
		ShortDescription: `
'ipfs tar add' will parse a tar file and create a merkledag structure to
represent it.
`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.FileArg("file", true, false, "Tar file to add.").EnableStdin(),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		it := req.Files.Entries()
		file, err := cmdenv.GetFileArg(it)
		if err != nil {
			return err
		}

		node, err := tar.ImportTar(req.Context, file, nd.DAG)
		if err != nil {
			return err
		}

		c := node.Cid()

		return cmds.EmitOnce(res, &AddEvent{
			Name: it.Name(),
			Hash: c.String(),
		})
	},
	Type: AddEvent{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, out *AddEvent) error {
			fmt.Fprintln(w, out.Hash)
			return nil
		}),
	},
}

var tarCatCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Export a tar file from IPFS.",
		ShortDescription: `
'ipfs tar cat' will export a tar file from a previously imported one in IPFS.
`,
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("path", true, false, "ipfs path of archive to export.").EnableStdin(),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		p, err := path.ParsePath(req.Arguments[0])
		if err != nil {
			return err
		}

		root, err := core.Resolve(req.Context, nd.Namesys, nd.Resolver, p)
		if err != nil {
			return err
		}

		rootpb, ok := root.(*dag.ProtoNode)
		if !ok {
			return dag.ErrNotProtobuf
		}

		r, err := tar.ExportTar(req.Context, rootpb, nd.DAG)
		if err != nil {
			return err
		}

		return res.Emit(r)
	},
}
