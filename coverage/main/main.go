
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+生成testrunmain

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	coverDir := os.Getenv("IPFS_COVER_DIR")
	if len(coverDir) == 0 {
		fmt.Println("IPFS_COVER_DIR not defined")
		os.Exit(1)
	}
	coverFile, err := ioutil.TempFile(coverDir, "coverage-")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	retFile, err := ioutil.TempFile("", "cover-ret-file")
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	args := []string{"-test.run", "^TestRunMain$", "-test.coverprofile=" + coverFile.Name(), "--"}
	args = append(args, os.Args[1:]...)

	p := exec.Command("ipfs-test-cover", args...)
	p.Stdin = os.Stdin
	p.Stdout = os.Stdout
	p.Stderr = os.Stderr
	p.Env = append(os.Environ(), "IPFS_COVER_RET_FILE="+retFile.Name())

	p.SysProcAttr = &syscall.SysProcAttr{
		Pdeathsig: syscall.SIGTERM,
	}

	sig := make(chan os.Signal, 1)
	go func() {
		for {
			p.Process.Signal(<-sig)
		}
	}()
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	err = p.Run()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	b, err := ioutil.ReadAll(retFile)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	b = b[:len(b)-1]
	d, err := strconv.Atoi(string(b))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	os.Exit(d)
}
