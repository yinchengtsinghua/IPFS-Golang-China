
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+建设！窗户

package util

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"testing"
)

func TestManageFdLimit(t *testing.T) {
	t.Log("Testing file descriptor count")
	if _, _, err := ManageFdLimit(); err != nil {
		t.Errorf("Cannot manage file descriptors")
	}

	if maxFds != uint64(2048) {
		t.Errorf("Maximum file descriptors default value changed")
	}
}

func TestManageInvalidNFds(t *testing.T) {
	t.Logf("Testing file descriptor invalidity")
	var err error
	if err = os.Unsetenv("IPFS_FD_MAX"); err != nil {
		t.Fatal("Cannot unset the IPFS_FD_MAX env variable")
	}

	rlimit := syscall.Rlimit{}
	if err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		t.Fatal("Cannot get the file descriptor count")
	}

	value := rlimit.Max + rlimit.Cur
	if err = os.Setenv("IPFS_FD_MAX", fmt.Sprintf("%d", value)); err != nil {
		t.Fatal("Cannot set the IPFS_FD_MAX env variable")
	}

//调用以检查并设置来自env的最大文件描述符
	setMaxFds()

	if _, _, err := ManageFdLimit(); err == nil {
		t.Errorf("ManageFdLimit should return an error")
	} else if err != nil {
		flag := strings.Contains(err.Error(),
			"cannot set rlimit, IPFS_FD_MAX is larger than the hard limit")
		if !flag {
			t.Errorf("ManageFdLimit returned unexpected error")
		}
	}

//取消设置所有以前的操作
	if err = os.Unsetenv("IPFS_FD_MAX"); err != nil {
		t.Fatal("Cannot unset the IPFS_FD_MAX env variable")
	}
}

func TestManageFdLimitWithEnvSet(t *testing.T) {
	t.Logf("Testing file descriptor manager with IPFS_FD_MAX set")
	var err error
	if err = os.Unsetenv("IPFS_FD_MAX"); err != nil {
		t.Fatal("Cannot unset the IPFS_FD_MAX env variable")
	}

	rlimit := syscall.Rlimit{}
	if err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit); err != nil {
		t.Fatal("Cannot get the file descriptor count")
	}

	value := rlimit.Max - rlimit.Cur + 1
	if err = os.Setenv("IPFS_FD_MAX", fmt.Sprintf("%d", value)); err != nil {
		t.Fatal("Cannot set the IPFS_FD_MAX env variable")
	}

	setMaxFds()
	if maxFds != uint64(value) {
		t.Errorf("The maxfds is not set from IPFS_FD_MAX")
	}

	if _, _, err = ManageFdLimit(); err != nil {
		t.Errorf("Cannot manage file descriptor count")
	}

//取消设置所有以前的操作
	if err = os.Unsetenv("IPFS_FD_MAX"); err != nil {
		t.Fatal("Cannot unset the IPFS_FD_MAX env variable")
	}
}
