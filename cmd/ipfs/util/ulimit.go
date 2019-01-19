
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package util

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"syscall"

	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("ulimit")

var (
	supportsFDManagement = false

//GetLimit返回文件描述符计数的软限制和硬限制
	getLimit func() (uint64, uint64, error)
//设置限制设置文件描述符计数的软限制和硬限制
	setLimit func(uint64, uint64) error
)

//maxfds是进入ipfs的最大文件描述符数。
//可以使用。默认值为2048。这可以被覆盖
//ipfs_fd_max env变量
var maxFds = uint64(2048)

//setmaxfds从ipfs_fd_max设置maxfds值
//env变量（如果它存在于系统中）
func setMaxFds() {
//检查是否设置了ipfs_fd_max，是否设置了
//没有有效的FDS编号通知用户
	if val := os.Getenv("IPFS_FD_MAX"); val != "" {

		fds, err := strconv.ParseUint(val, 10, 64)
		if err != nil {
			log.Errorf("bad value for IPFS_FD_MAX: %s", err)
			return
		}

		maxFds = fds
	}
}

//managefdlimit提高当前最大文件描述符计数
//基于ipfs_fd_max值的过程
func ManageFdLimit() (changed bool, newLimit uint64, err error) {
	if !supportsFDManagement {
		return false, 0, nil
	}

	setMaxFds()
	soft, hard, err := getLimit()
	if err != nil {
		return false, 0, err
	}

	if maxFds <= soft {
		return false, 0, nil
	}

//软限制是内核为
//对应资源
//硬极限作为软极限的上限。
//非特权进程只能将其软限制设置为
//值在0到硬限制之间
	if err = setLimit(maxFds, maxFds); err != nil {
		if err != syscall.EPERM {
			return false, 0, fmt.Errorf("error setting: ulimit: %s", err)
		}

//进程没有权限，因此我们只能
//设置软值
		if maxFds > hard {
			return false, 0, errors.New(
				"cannot set rlimit, IPFS_FD_MAX is larger than the hard limit",
			)
		}

		if err = setLimit(maxFds, hard); err != nil {
			return false, 0, fmt.Errorf("error setting ulimit wihout hard limit: %s", err)
		}
	}

	return true, maxFds, nil
}
