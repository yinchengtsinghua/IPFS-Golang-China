
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
	"fmt"
	"io"
	"os"

	"github.com/ipfs/go-ipfs/core/commands/cmdenv"
	"github.com/ipfs/go-ipfs/core/coreapi/interface"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	"gx/ipfs/QmXWZCd8jfaHmt4UDSnjKmGcrQMw95bDGWqEeVLVJjoANX/go-ipfs-files"
	"gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

const (
progressBarMinSize = 1024 * 1024 * 8 //显示输出>8mib的进度条
	offsetOptionName   = "offset"
	lengthOptionName   = "length"
)

var CatCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline:          "Show IPFS object data.",
		ShortDescription: "Displays the data contained by an IPFS or IPNS object(s) at the given path.",
	},

	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("ipfs-path", true, true, "The path to the IPFS object(s) to be outputted.").EnableStdin(),
	},
	Options: []cmdkit.Option{
		cmdkit.Int64Option(offsetOptionName, "o", "Byte offset to begin reading from."),
		cmdkit.Int64Option(lengthOptionName, "l", "Maximum number of bytes to read."),
	},
	Run: func(req *cmds.Request, res cmds.ResponseEmitter, env cmds.Environment) error {
		api, err := cmdenv.GetApi(env, req)
		if err != nil {
			return err
		}

		offset, _ := req.Options[offsetOptionName].(int64)
		if offset < 0 {
			return fmt.Errorf("cannot specify negative offset")
		}

		max, found := req.Options[lengthOptionName].(int64)

		if max < 0 {
			return fmt.Errorf("cannot specify negative length")
		}
		if !found {
			max = -1
		}

		err = req.ParseBodyArgs()
		if err != nil {
			return err
		}

		readers, length, err := cat(req.Context, api, req.Arguments, int64(offset), int64(max))
		if err != nil {
			return err
		}

  /*
   如果错误：=correpo.conditionalgc（req.context、node、length）；错误！= nIL{
    重置错误（err，CmdKit.errNormal）
    返回
   }
  **/


		res.SetLength(length)
		reader := io.MultiReader(readers...)

//因为读卡器返回一个块丢失的错误，而该错误是
//从IO返回。复制到Emit中，我们需要获取Emit错误并发送
//他们给客户。通常我们不这样做，因为这意味着连接
//被破坏或我们提供了非法参数等。
		return res.Emit(reader)
	},
	PostRun: cmds.PostRunMap{
		cmds.CLI: func(res cmds.Response, re cmds.ResponseEmitter) error {
			if res.Length() > 0 && res.Length() < progressBarMinSize {
				return cmds.Copy(re, res)
			}

			for {
				v, err := res.Next()
				if err != nil {
					if err == io.EOF {
						return nil
					}
					return err
				}

				switch val := v.(type) {
				case io.Reader:
					bar, reader := progressBarForReader(os.Stderr, val, int64(res.Length()))
					bar.Start()

					err = re.Emit(reader)
					if err != nil {
						return err
					}
				default:
					log.Warningf("cat postrun: received unexpected type %T", val)
				}
			}
		},
	},
}

func cat(ctx context.Context, api iface.CoreAPI, paths []string, offset int64, max int64) ([]io.Reader, uint64, error) {
	readers := make([]io.Reader, 0, len(paths))
	length := uint64(0)
	if max == 0 {
		return nil, 0, nil
	}
	for _, p := range paths {
		fpath, err := iface.ParsePath(p)
		if err != nil {
			return nil, 0, err
		}

		f, err := api.Unixfs().Get(ctx, fpath)
		if err != nil {
			return nil, 0, err
		}

		file, ok := f.(files.File)
		if !ok {
			return nil, 0, iface.ErrNotFile
		}

		fsize, err := file.Size()
		if err != nil {
			return nil, 0, err
		}

		if offset > fsize {
			offset = offset - fsize
			continue
		}

		count, err := file.Seek(offset, io.SeekStart)
		if err != nil {
			return nil, 0, err
		}
		offset = 0

		fsize, err = file.Size()
		if err != nil {
			return nil, 0, err
		}

		size := uint64(fsize - count)
		length += size
		if max > 0 && length >= uint64(max) {
			var r io.Reader = file
			if overshoot := int64(length - uint64(max)); overshoot != 0 {
				r = io.LimitReader(file, int64(size)-overshoot)
				length = uint64(max)
			}
			readers = append(readers, r)
			break
		}
		readers = append(readers, file)
	}
	return readers, length, nil
}
