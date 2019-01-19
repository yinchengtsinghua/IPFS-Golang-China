
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//包装入提供了围绕装入点的简单抽象
package mount

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"time"

	goprocess "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("mount")

var MountTimeout = time.Second * 5

//mount表示文件系统装载
type Mount interface {
//mountpoint是安装此装载的路径
	MountPoint() string

//卸载装载
	Unmount() error

//检查安装是否仍处于活动状态。
	IsActive() bool

//进程返回mount的进程，以便能够将其链接
//其他流程。关闭时卸载。
	Process() goprocess.Process
}

//forceUnmount尝试强制卸载给定的装载。
//它通过直接调用diskutil或fusermount来实现。
func ForceUnmount(m Mount) error {
	point := m.MountPoint()
	log.Warningf("Force-Unmounting %s...", point)

	cmd, err := UnmountCmd(point)
	if err != nil {
		return err
	}

	errc := make(chan error, 1)
	go func() {
		defer close(errc)

//先试试香草卸载。
		if err := exec.Command("umount", point).Run(); err == nil {
			return
		}

//使用fallback命令重试卸载
		errc <- cmd.Run()
	}()

	select {
	case <-time.After(7 * time.Second):
		return fmt.Errorf("umount timeout")
	case err := <-errc:
		return err
	}
}

//unmountcmd创建一个特定于goos的exec.cmd
//用于拆卸保险丝座
func UnmountCmd(point string) (*exec.Cmd, error) {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("diskutil", "umount", "force", point), nil
	case "linux":
		return exec.Command("fusermount", "-u", point), nil
	default:
		return nil, fmt.Errorf("unmount: unimplemented")
	}
}

//强制卸载每次尝试强制卸载给定的装载，
//很多次。它通过直接调用diskutil或fusermount来实现。
//尝试给定次数。
func ForceUnmountManyTimes(m Mount, attempts int) error {
	var err error
	for i := 0; i < attempts; i++ {
		err = ForceUnmount(m)
		if err == nil {
			return err
		}

		<-time.After(time.Millisecond * 500)
	}
	return fmt.Errorf("unmount %s failed after 10 seconds of trying", m.MountPoint())
}

type closer struct {
	M Mount
}

func (c *closer) Close() error {
	log.Warning(" (c *closer) Close(),", c.M.MountPoint())
	return c.M.Unmount()
}

func Closer(m Mount) io.Closer {
	return &closer{m}
}
