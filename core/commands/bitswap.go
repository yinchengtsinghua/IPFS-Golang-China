
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

	cmdenv "github.com/ipfs/go-ipfs/core/commands/cmdenv"
	e "github.com/ipfs/go-ipfs/core/commands/e"

	humanize "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	bitswap "gx/ipfs/QmYoGLuLwTUv1SYBmsw1EVNC9MyLVUxwxzXYtKgAGHyEfw/go-bitswap"
	decision "gx/ipfs/QmYoGLuLwTUv1SYBmsw1EVNC9MyLVUxwxzXYtKgAGHyEfw/go-bitswap/decision"
	cidutil "gx/ipfs/QmbfKu17LbMWyGUxHEUns9Wf5Dkm8PT6be4uPhTkk4YvaV/go-cidutil"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

var BitswapCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:          "Interact with the bitswap agent.",
		ShortDescription: ``,
	},

	Subcommands: map[string]*cmds.Command{
		"stat":      bitswapStatCmd,
		"wantlist":  showWantlistCmd,
		"ledger":    ledgerCmd,
		"reprovide": reprovideCmd,
	},
}

const (
	peerOptionName = "peer"
)

var showWantlistCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Show blocks currently on the wantlist.",
		ShortDescription: `
Print out all blocks currently on the bitswap wantlist for the local peer.`,
	},
	Options: []cmdkit.Option{
		cmdkit.StringOption(peerOptionName, "p", "Specify which peer to show wantlist for. Default: self."),
	},
	Type: KeyList{},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		if !nd.OnlineMode() {
			return ErrNotOnline
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			return e.TypeErr(bs, nd.Exchange)
		}

		pstr, found := req.Options[peerOptionName].(string)
		if found {
			pid, err := peer.IDB58Decode(pstr)
			if err != nil {
				return err
			}
			if pid != nd.Identity {
				return cmds.EmitOnce(res, &KeyList{bs.WantlistForPeer(pid)})
			}
		}

		return cmds.EmitOnce(res, &KeyList{bs.GetWantlist()})
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, out *KeyList) error {
//先把钥匙分类
			cidutil.Sort(out.Keys)
			for _, key := range out.Keys {
				fmt.Fprintln(w, key)
			}

			return nil
		}),
	},
}

var bitswapStatCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:          "Show some diagnostic information on the bitswap agent.",
		ShortDescription: ``,
	},
	Type: bitswap.Stat{},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		if !nd.OnlineMode() {
			return cmdkit.Errorf(cmdkit.ErrClient, ErrNotOnline.Error())
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			return e.TypeErr(bs, nd.Exchange)
		}

		st, err := bs.Stat()
		if err != nil {
			return err
		}

		return cmds.EmitOnce(res, st)
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, s *bitswap.Stat) error {
			fmt.Fprintln(w, "bitswap status")
			fmt.Fprintf(w, "\tprovides buffer: %d / %d\n", s.ProvideBufLen, bitswap.HasBlockBufferSize)
			fmt.Fprintf(w, "\tblocks received: %d\n", s.BlocksReceived)
			fmt.Fprintf(w, "\tblocks sent: %d\n", s.BlocksSent)
			fmt.Fprintf(w, "\tdata received: %d\n", s.DataReceived)
			fmt.Fprintf(w, "\tdata sent: %d\n", s.DataSent)
			fmt.Fprintf(w, "\tdup blocks received: %d\n", s.DupBlksReceived)
			fmt.Fprintf(w, "\tdup data received: %s\n", humanize.Bytes(s.DupDataReceived))
			fmt.Fprintf(w, "\twantlist [%d keys]\n", len(s.Wantlist))
			for _, k := range s.Wantlist {
				fmt.Fprintf(w, "\t\t%s\n", k.String())
			}
			fmt.Fprintf(w, "\tpartners [%d]\n", len(s.Peers))
			for _, p := range s.Peers {
				fmt.Fprintf(w, "\t\t%s\n", p)
			}

			return nil
		}),
	},
}

var ledgerCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Show the current ledger for a peer.",
		ShortDescription: `
The Bitswap decision engine tracks the number of bytes exchanged between IPFS
nodes, and stores this information as a collection of ledgers. This command
prints the ledger associated with a given peer.
`,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("peer", true, false, "The PeerID (B58) of the ledger to inspect."),
	},
	Type: decision.Receipt{},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		if !nd.OnlineMode() {
			return ErrNotOnline
		}

		bs, ok := nd.Exchange.(*bitswap.Bitswap)
		if !ok {
			return e.TypeErr(bs, nd.Exchange)
		}

		partner, err := peer.IDB58Decode(req.Arguments[0])
		if err != nil {
			return err
		}

		return cmds.EmitOnce(res, bs.LedgerForPeer(partner))
	},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, out *decision.Receipt) error {
			fmt.Fprintf(w, "Ledger for %s\n"+
				"Debt ratio:\t%f\n"+
				"Exchanges:\t%d\n"+
				"Bytes sent:\t%d\n"+
				"Bytes received:\t%d\n\n",
				out.Peer, out.Value, out.Exchanged,
				out.Sent, out.Recv)
			return nil
		}),
	},
}

var reprovideCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Trigger reprovider.",
		ShortDescription: `
Trigger reprovider to announce our data to network.
`,
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		nd, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		if !nd.OnlineMode() {
			return ErrNotOnline
		}

		err = nd.Reprovider.Trigger(req.Context)
		if err != nil {
			return err
		}

		return nil
	},
}
