
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+构建Linux darwin freebsd netbsd openbsd
//+建设！诺福斯

package ipns

import (
	core "github.com/ipfs/go-ipfs/core"
	mount "github.com/ipfs/go-ipfs/fuse/mount"
)

//mount在给定位置安装IPN，并返回mount.mount实例。
func Mount(ipfs *core.IpfsNode, ipnsmp, ipfsmp string) (mount.Mount, error) {
	cfg, err := ipfs.Repo.Config()
	if err != nil {
		return nil, err
	}

	allow_other := cfg.Mounts.FuseAllowOther

	fsys, err := NewFileSystem(ipfs, ipfs.PrivateKey, ipfsmp, ipnsmp)
	if err != nil {
		return nil, err
	}

	return mount.NewMount(ipfs.Process(), fsys, ipnsmp, allow_other)
}
