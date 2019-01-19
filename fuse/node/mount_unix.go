
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+建设！窗户，！诺福斯

package node

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	core "github.com/ipfs/go-ipfs/core"
	ipns "github.com/ipfs/go-ipfs/fuse/ipns"
	mount "github.com/ipfs/go-ipfs/fuse/mount"
	rofs "github.com/ipfs/go-ipfs/fuse/readonly"

	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("node")

//fusenoDirectory用于检查返回的fuse错误
const fuseNoDirectory = "fusermount: failed to access mountpoint"

//FuseExitStatus1用于检查返回的保险丝错误
const fuseExitStatus1 = "fusermount: exit status 1"

//PlatformFuseChecks可以被特定于arch的文件覆盖
//运行保险丝检查（如检查OSXFuse版本）
var platformFuseChecks = func(*core.IpfsNode) error {
	return nil
}

func Mount(node *core.IpfsNode, fsdir, nsdir string) error {
//检查我们是否已经有活的坐骑。
//如果用户说“mount”，那么肯定有什么问题。
//所以，关闭它们然后再试一次。
	if node.Mounts.Ipfs != nil && node.Mounts.Ipfs.IsActive() {
		node.Mounts.Ipfs.Unmount()
	}
	if node.Mounts.Ipns != nil && node.Mounts.Ipns.IsActive() {
		node.Mounts.Ipns.Unmount()
	}

	if err := platformFuseChecks(node); err != nil {
		return err
	}

	return doMount(node, fsdir, nsdir)
}

func doMount(node *core.IpfsNode, fsdir, nsdir string) error {
	fmtFuseErr := func(err error, mountpoint string) error {
		s := err.Error()
		if strings.Contains(s, fuseNoDirectory) {
			s = strings.Replace(s, `fusermount: "fusermount:`, "", -1)
			s = strings.Replace(s, `\n", exit status 1`, "", -1)
			return errors.New(s)
		}
		if s == fuseExitStatus1 {
			s = fmt.Sprintf("fuse failed to access mountpoint %s", mountpoint)
			return errors.New(s)
		}
		return err
	}

//这种同步功能使两者都可以同时安装。
	var fsmount, nsmount mount.Mount
	var err1, err2 error

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		fsmount, err1 = rofs.Mount(node, fsdir)
	}()

	if node.OnlineMode() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			nsmount, err2 = ipns.Mount(node, nsdir, fsdir)
		}()
	}

	wg.Wait()

	if err1 != nil {
		log.Errorf("error mounting: %s", err1)
	}

	if err2 != nil {
		log.Errorf("error mounting: %s", err2)
	}

	if err1 != nil || err2 != nil {
		if fsmount != nil {
			fsmount.Unmount()
		}
		if nsmount != nil {
			nsmount.Unmount()
		}

		if err1 != nil {
			return fmtFuseErr(err1, fsdir)
		}
		return fmtFuseErr(err2, nsdir)
	}

//设置节点状态，以便可以取消
	node.Mounts.Ipfs = fsmount
	node.Mounts.Ipns = nsmount
	return nil
}
