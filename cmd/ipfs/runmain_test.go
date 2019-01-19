
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
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

//这种滥用非常严重，以至于我在写这段代码时感到很肮脏。
//但这是在不编写自定义编译器的情况下完成此任务的唯一方法
//通过go测试成为go build的克隆
func TestRunMain(t *testing.T) {
	args := flag.Args()
	os.Args = append([]string{os.Args[0]}, args...)
	ret := mainRet()

	p := os.Getenv("IPFS_COVER_RET_FILE")
	if len(p) != 0 {
		ioutil.WriteFile(p, []byte(fmt.Sprintf("%d\n", ret)), 0777)
	}

//关闭输出，这样Go测试就不会打印任何内容
	null, _ := os.Open(os.DevNull)
	os.Stderr = null
	os.Stdout = null
}
