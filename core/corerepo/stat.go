
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corerepo

import (
	"fmt"
	"math"

	context "context"

	"github.com/ipfs/go-ipfs/core"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"

	humanize "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
)

//sizestat包装有关存储库大小及其限制的信息。
type SizeStat struct {
RepoSize   uint64 //字节大小
StorageMax uint64 //字节大小
}

//stat包装有关存储在磁盘上的对象的信息。
type Stat struct {
	SizeStat
	NumObjects uint64
	RepoPath   string
	Version    string
}

//nolimit表示无限存储的值
const NoLimit uint64 = math.MaxUint64

//repostat返回一个设置了所有字段的*stat对象。
func RepoStat(ctx context.Context, n *core.IpfsNode) (Stat, error) {
	sizeStat, err := RepoSize(ctx, n)
	if err != nil {
		return Stat{}, err
	}

	allKeys, err := n.Blockstore.AllKeysChan(ctx)
	if err != nil {
		return Stat{}, err
	}

	count := uint64(0)
	for range allKeys {
		count++
	}

	path, err := fsrepo.BestKnownPath()
	if err != nil {
		return Stat{}, err
	}

	return Stat{
		SizeStat: SizeStat{
			RepoSize:   sizeStat.RepoSize,
			StorageMax: sizeStat.StorageMax,
		},
		NumObjects: count,
		RepoPath:   path,
		Version:    fmt.Sprintf("fs-repo@%d", fsrepo.RepoVersion),
	}, nil
}

//
func RepoSize(ctx context.Context, n *core.IpfsNode) (SizeStat, error) {
	r := n.Repo

	cfg, err := r.Config()
	if err != nil {
		return SizeStat{}, err
	}

	usage, err := r.GetStorageUsage()
	if err != nil {
		return SizeStat{}, err
	}

	storageMax := NoLimit
	if cfg.Datastore.StorageMax != "" {
		storageMax, err = humanize.ParseBytes(cfg.Datastore.StorageMax)
		if err != nil {
			return SizeStat{}, err
		}
	}

	return SizeStat{
		RepoSize:   usage,
		StorageMax: storageMax,
	}, nil
}
