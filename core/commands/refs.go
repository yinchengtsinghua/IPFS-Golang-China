
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
	"fmt"
	"io"
	"strings"

	core "github.com/ipfs/go-ipfs/core"
	cmdenv "github.com/ipfs/go-ipfs/core/commands/cmdenv"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

var refsEncoderMap = cmds.EncoderMap{
	cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, out *RefWrapper) error {
		if out.Err != "" {
			return fmt.Errorf(out.Err)
		}
		fmt.Fprintln(w, out.Ref)

		return nil
	}),
}

//keylist是用于输出键列表的常规类型
type KeyList struct {
	Keys []cid.Cid
}

const (
	refsFormatOptionName    = "format"
	refsEdgesOptionName     = "edges"
	refsUniqueOptionName    = "unique"
	refsRecursiveOptionName = "recursive"
	refsMaxDepthOptionName  = "max-depth"
)

//refscmd是'ipfs refs'命令
var RefsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "List links (references) from an object.",
		ShortDescription: `
Lists the hashes of all the links an IPFS or IPNS object(s) contains,
with the following format:

  <link base58 hash>

NOTE: List all references recursively by using the flag '-r'.
`,
	},
	Subcommands: map[string]*cmds.Command{
		"local": RefsLocalCmd,
	},
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("ipfs-path", true, true, "Path to the object(s) to list refs from.").EnableStdin(),
	},
	Options: []cmdkit.Option{
		cmdkit.StringOption(refsFormatOptionName, "Emit edges with given format. Available tokens: <src> <dst> <linkname>.").WithDefault("<dst>"),
		cmdkit.BoolOption(refsEdgesOptionName, "e", "Emit edge format: `<from> -> <to>`."),
		cmdkit.BoolOption(refsUniqueOptionName, "u", "Omit duplicate refs from output."),
		cmdkit.BoolOption(refsRecursiveOptionName, "r", "Recursively list links of child nodes."),
		cmdkit.IntOption(refsMaxDepthOptionName, "Only for recursive refs, limits fetch and listing to the given depth").WithDefault(-1),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		err := req.ParseBodyArgs()
		if err != nil {
			return err
		}

		ctx := req.Context
		n, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

		unique, _ := req.Options[refsUniqueOptionName].(bool)
		recursive, _ := req.Options[refsRecursiveOptionName].(bool)
		maxDepth, _ := req.Options[refsMaxDepthOptionName].(int)
		edges, _ := req.Options[refsEdgesOptionName].(bool)
		format, _ := req.Options[refsFormatOptionName].(string)

		if !recursive {
maxDepth = 1 //只写直接引用
		}

		if edges {
			if format != "<dst>" {
				return errors.New("using format argument with edges is not allowed")
			}

			format = "<src> -> <dst>"
		}

		objs, err := objectsForPaths(ctx, n, req.Arguments)
		if err != nil {
			return err
		}

		rw := RefWriter{
			res:      res,
			DAG:      n.DAG,
			Ctx:      ctx,
			Unique:   unique,
			PrintFmt: format,
			MaxDepth: maxDepth,
		}

		for _, o := range objs {
			if _, err := rw.WriteRefs(o); err != nil {
				if err := res.Emit(&RefWrapper{Err: err.Error()}); err != nil {
					return err
				}
			}
		}

		return nil
	},
	Encoders: refsEncoderMap,
	Type:     RefWrapper{},
}

var RefsLocalCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "List all local references.",
		ShortDescription: `
Displays the hashes of all local objects.
`,
	},

	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		ctx := req.Context
		n, err := cmdenv.GetNode(env)
		if err != nil {
			return err
		}

//TODO:使异步
		allKeys, err := n.Blockstore.AllKeysChan(ctx)
		if err != nil {
			return err
		}

		for k := range allKeys {
			err := res.Emit(&RefWrapper{Ref: k.String()})
			if err != nil {
				return err
			}
		}

		return nil
	},
	Encoders: refsEncoderMap,
	Type:     RefWrapper{},
}

func objectsForPaths(ctx context.Context, n *core.IpfsNode, paths []string) ([]ipld.Node, error) {
	objects := make([]ipld.Node, len(paths))
	for i, sp := range paths {
		p, err := path.ParsePath(sp)
		if err != nil {
			return nil, err
		}

		o, err := core.Resolve(ctx, n.Namesys, n.Resolver, p)
		if err != nil {
			return nil, err
		}
		objects[i] = o
	}
	return objects, nil
}

type RefWrapper struct {
	Ref string
	Err string
}

type RefWriter struct {
	res cmds.ResponseEmitter
	DAG ipld.DAGService
	Ctx context.Context

	Unique   bool
	MaxDepth int
	PrintFmt string

	seen map[string]int
}

//writerefs将给定对象的refs写入基础编写器。
func (rw *RefWriter) WriteRefs(n ipld.Node) (int, error) {
	return rw.writeRefsRecursive(n, 0)
}

func (rw *RefWriter) writeRefsRecursive(n ipld.Node, depth int) (int, error) {
	nc := n.Cid()

	var count int
	for i, ng := range ipld.GetDAG(rw.Ctx, rw.DAG, n) {
		lc := n.Links()[i].Cid
goDeeper, shouldWrite := rw.visit(lc, depth+1) //孩子们在深度+1处

//避免在节点上出现“get（）”，继续下一个链接。
//我们可以这样做，如果：
//-我们以前印刷过它（因此它已经被看到并且
//用get（）获取
//我们不能再深入了。
//这是对已修剪的分支的优化
//以前参观过。
		if !shouldWrite && !goDeeper {
			continue
		}

//我们必须获取（）节点，因为：
//-它是新的（从未写过）
//或者我们需要更深入。
//这样可以确保始终获取打印的引用。
		nd, err := ng.Get(rw.Ctx)
		if err != nil {
			return count, err
		}

//如果之前未完成，则写入此节点（或！独特的）
		if shouldWrite {
			if err := rw.WriteEdge(nc, lc, n.Links()[i].Name); err != nil {
				return count, err
			}
			count++
		}

//继续深入。这种情况发生了：
//-在未探测的分支上
//-树枝探索不够深
//注意什么时候！唯一，始终考虑分支
//未勘探，仅深度限制适用。
		if goDeeper {
			c, err := rw.writeRefsRecursive(nd, depth+1)
			count += c
			if err != nil {
				return count, err
			}
		}
	}

	return count, nil
}

//访问返回两个值：
//-如果我们继续遍历DAG，第一个布尔值是真的。
//-如果我们要打印cid，第二个布尔值是真的。
//
//访问将根据先前访问的rw.maxdepth进行分支修剪。
//CIDS以及是否设置了rw.unique。即rw.unique=false和
//rw.maxdepth=-1禁用任何修剪。但是设置rw。真正的意志所独有的
//修剪已经访问过的分支机构，以保持访问集的成本
//内存中的CID。
func (rw *RefWriter) visit(c cid.Cid, depth int) (bool, bool) {
	atMaxDepth := rw.MaxDepth >= 0 && depth == rw.MaxDepth
	overMaxDepth := rw.MaxDepth >= 0 && depth > rw.MaxDepth

//当我们超过最大深度时的快捷方式。实际上，这个
//仅在使用--maxdepth=0作为根调用refs时适用
//孩子们已经超过了最大深度。否则什么都不应该
//打这个。
	if overMaxDepth {
		return false, false
	}

//
//-当不在最大深度时，我们继续遍历
//-总是打印
	if !rw.Unique {
		return !atMaxDepth, true
	}

//唯一==true。
//因此，我们跟踪观察到的CID及其深度。
	if rw.seen == nil {
		rw.seen = make(map[string]int)
	}
	key := string(c.Bytes())
	oldDepth, ok := rw.seen[key]

//唯一==true&&depth<maxdepth（或无限制）

//树枝修剪情况：
//-我们以前看过CID，或者：
//-深度不受限制（最大深度=-1）
//-我们在DAG中看到它更高（更小的深度）（意味着我们必须
//以前探索得够深）
//因为我们看到了CID，所以我们不再打印了。
	if ok && (rw.MaxDepth < 0 || oldDepth <= depth) {
		return false, false
	}

//最后一个案例，我们必须继续从这个CID探索DAG
//（除非我们达到深度限制）。
//我们记下了它的深度，因为它或者没有被看到
//or is lower than last time.
//如果看不到，我们就打印。
	rw.seen[key] = depth
	return !atMaxDepth, !ok
}

//写一个边
func (rw *RefWriter) WriteEdge(from, to cid.Cid, linkname string) error {
	if rw.Ctx != nil {
		select {
case <-rw.Ctx.Done(): //以防万一。
			return rw.Ctx.Err()
		default:
		}
	}

	var s string
	switch {
	case rw.PrintFmt != "":
		s = rw.PrintFmt
		s = strings.Replace(s, "<src>", from.String(), -1)
		s = strings.Replace(s, "<dst>", to.String(), -1)
		s = strings.Replace(s, "<linkname>", linkname, -1)
	default:
		s += to.String()
	}

	return rw.res.Emit(&RefWrapper{Ref: s})
}
