
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//PollEndpoint是一个帮助工具，它等待HTTP端点可访问并返回http.statusok
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"time"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	manet "gx/ipfs/QmZcLBXKaFe8ND5YHPkJRAwmhJGrVsi1JqDZNyJ4nRK5Mj/go-multiaddr-net"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var (
	host     = flag.String("host", "/ip4/127.0.0.1/tcp/5001", "the multiaddr host to dial on")
	endpoint = flag.String("ep", "/version", "which http endpoint path to hit")
	tries    = flag.Int("tries", 10, "how many tries to make before failing")
	timeout  = flag.Duration("tout", time.Second, "how long to wait between attempts")
	verbose  = flag.Bool("v", false, "verbose logging")
)

var log = logging.Logger("pollEndpoint")

func main() {
	flag.Parse()

//从主机标志提取地址
	addr, err := ma.NewMultiaddr(*host)
	if err != nil {
		log.Fatal("NewMultiaddr() failed: ", err)
	}
	p := addr.Protocols()
	if len(p) < 2 {
		log.Fatal("need two protocols in host flag (/ip/tcp): ", addr)
	}
	_, host, err := manet.DialArgs(addr)
	if err != nil {
		log.Fatal("manet.DialArgs() failed: ", err)
	}

if *verbose { //低日志级别
		logging.SetDebugLogging()
	}

//构造要拨号的URL
	var u url.URL
	u.Scheme = "http"
	u.Host = host
	u.Path = *endpoint

//展示我们的实力
	start := time.Now()
	log.Debugf("starting at %s, tries: %d, timeout: %s, url: %s", start, *tries, *timeout, u)

	for *tries > 0 {

		err := checkOK(http.Get(u.String()))
		if err == nil {
			log.Debugf("ok -  endpoint reachable with %d tries remaining, took %s", *tries, time.Since(start))
			os.Exit(0)
		}
		log.Debug("get failed: ", err)
		time.Sleep(*timeout)
		*tries--
	}

	log.Error("failed.")
	os.Exit(1)
}

func checkOK(resp *http.Response, err error) error {
if err == nil { //请求工作
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return nil
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "pollEndpoint: ioutil.ReadAll() Error: %s", err)
		}
		return fmt.Errorf("response not OK. %d %s %q", resp.StatusCode, resp.Status, string(body))
	}
	return err
}
