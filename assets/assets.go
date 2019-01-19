
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//go:generate go bindata-pkg=assets-prefix=$gopath/src/gx/ipfs/qmt1jwrqzsjjlg5obd9w4p9vxpkqkswuf5ghse3q88zv init doc$gopath/src/gx/ipfs/qmt1jwrqzsjlg5obd9w4p9vxpkqkswuf5ghse3q88zv/dir index html
//go：生成gofmt-w bindata.go

package assets

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ipfs/go-ipfs/core"
	"github.com/ipfs/go-ipfs/core/coreunix"
	uio "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/io"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"

//这种导入使GX不会认为DEP未被使用。
	_ "gx/ipfs/QmT1jwrqzSMjSjLG5oBd9w4P9vXPKQksWuf5ghsE3Q88ZV/dir-index-html"
)

//initdocpaths列出了在--init期间要种子化的文档的路径
var initDocPaths = []string{
	filepath.Join("init-doc", "about"),
	filepath.Join("init-doc", "readme"),
	filepath.Join("init-doc", "help"),
	filepath.Join("init-doc", "contact"),
	filepath.Join("init-doc", "security-notes"),
	filepath.Join("init-doc", "quick-start"),
	filepath.Join("init-doc", "ping"),
}

//seediinitdocs将嵌入的init文档列表添加到传递的节点，固定它并返回根键
func SeedInitDocs(nd *core.IpfsNode) (cid.Cid, error) {
	return addAssetList(nd, initDocPaths)
}

var initDirPath = filepath.Join(os.Getenv("GOPATH"), "gx", "ipfs", "QmT1jwrqzSMjSjLG5oBd9w4P9vXPKQksWuf5ghsE3Q88ZV", "dir-index-html")
var initDirIndex = []string{
	filepath.Join(initDirPath, "knownIcons.txt"),
	filepath.Join(initDirPath, "dir-index.html"),
}

func SeedInitDirIndex(nd *core.IpfsNode) (cid.Cid, error) {
	return addAssetList(nd, initDirIndex)
}

func addAssetList(nd *core.IpfsNode, l []string) (cid.Cid, error) {
	dirb := uio.NewDirectory(nd.DAG)

	for _, p := range l {
		d, err := Asset(p)
		if err != nil {
			return cid.Cid{}, fmt.Errorf("assets: could load Asset '%s': %s", p, err)
		}

		s, err := coreunix.Add(nd, bytes.NewBuffer(d))
		if err != nil {
			return cid.Cid{}, fmt.Errorf("assets: could not Add '%s': %s", p, err)
		}

		fname := filepath.Base(p)

		c, err := cid.Decode(s)
		if err != nil {
			return cid.Cid{}, err
		}

		node, err := nd.DAG.Get(nd.Context(), c)
		if err != nil {
			return cid.Cid{}, err
		}

		if err := dirb.AddChild(nd.Context(), fname, node); err != nil {
			return cid.Cid{}, fmt.Errorf("assets: could not add '%s' as a child: %s", fname, err)
		}
	}

	dir, err := dirb.GetNode()
	if err != nil {
		return cid.Cid{}, err
	}

	if err := nd.Pinning.Pin(nd.Context(), dir, true); err != nil {
		return cid.Cid{}, fmt.Errorf("assets: Pinning on init-docu failed: %s", err)
	}

	if err := nd.Pinning.Flush(); err != nil {
		return cid.Cid{}, fmt.Errorf("assets: Pinning flush failed: %s", err)
	}

	return dir.Cid(), nil
}
