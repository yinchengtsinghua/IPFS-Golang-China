
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package git

import (
	"compress/zlib"
	"fmt"
	"io"
	"math"

	"github.com/ipfs/go-ipfs/core/coredag"
	"github.com/ipfs/go-ipfs/plugin"

	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	git "gx/ipfs/QmanwVNHFVTMvXmvxu4ngaPSNEKDcNbwg35ZZSJBFfmWGt/go-ipld-git"
	"gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	mh "gx/ipfs/QmerPMzPk1mJVowm8KgmoknWa4yCYvvugMPsgWmDNUvDLW/go-multihash"
)

//插件导出将加载的插件列表
var Plugins = []plugin.Plugin{
	&gitPlugin{},
}

type gitPlugin struct{}

var _ plugin.PluginIPLD = (*gitPlugin)(nil)

func (*gitPlugin) Name() string {
	return "ipld-git"
}

func (*gitPlugin) Version() string {
	return "0.0.1"
}

func (*gitPlugin) Init() error {
	return nil
}

func (*gitPlugin) RegisterBlockDecoders(dec format.BlockDecoder) error {
	dec.Register(cid.GitRaw, git.DecodeBlock)
	return nil
}

func (*gitPlugin) RegisterInputEncParsers(iec coredag.InputEncParsers) error {
	iec.AddParser("raw", "git", parseRawGit)
	iec.AddParser("zlib", "git", parseZlibGit)
	return nil
}

func parseRawGit(r io.Reader, mhType uint64, mhLen int) ([]format.Node, error) {
	if mhType != math.MaxUint64 && mhType != mh.SHA1 {
		return nil, fmt.Errorf("unsupported mhType %d", mhType)
	}

	if mhLen != -1 && mhLen != mh.DefaultLengths[mh.SHA1] {
		return nil, fmt.Errorf("invalid mhLen %d", mhLen)
	}

	nd, err := git.ParseObject(r)
	if err != nil {
		return nil, err
	}

	return []format.Node{nd}, nil
}

func parseZlibGit(r io.Reader, mhType uint64, mhLen int) ([]format.Node, error) {
	rc, err := zlib.NewReader(r)
	if err != nil {
		return nil, err
	}

	defer rc.Close()
	return parseRawGit(rc, mhType, mhLen)
}
