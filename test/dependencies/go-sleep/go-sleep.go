
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	if len(os.Args) != 2 {
		usageError()
	}
	d, err := time.ParseDuration(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not parse duration: %s\n", err)
		usageError()
	}

	time.Sleep(d)
}

func usageError() {
	fmt.Fprintf(os.Stderr, "Usage: %s <duration>\n", os.Args[0])
	fmt.Fprintln(os.Stderr, `Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".`)
fmt.Fprintln(os.Stderr, "See https://godoc.org/time parseDuration for more.”）
	os.Exit(-1)
}
