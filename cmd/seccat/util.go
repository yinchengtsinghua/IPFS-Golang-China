
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
	"net"
	"os"

	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("seccat")

func exit(format string, vals ...interface{}) {
	if format != "" {
		fmt.Fprintf(os.Stderr, "seccat: error: "+format+"\n", vals...)
	}
	Usage()
	os.Exit(1)
}

func out(format string, vals ...interface{}) {
	if verbose {
		fmt.Fprintf(os.Stderr, "seccat: "+format+"\n", vals...)
	}
}

type logConn struct {
	net.Conn
	n string
}

func (r *logConn) Read(buf []byte) (int, error) {
	n, err := r.Conn.Read(buf)
	if n > 0 {
		log.Debugf("%s read: %v", r.n, buf)
	}
	return n, err
}

func (r *logConn) Write(buf []byte) (int, error) {
	log.Debugf("%s write: %v", r.n, buf)
	return r.Conn.Write(buf)
}

func (r *logConn) Close() error {
	return r.Conn.Close()
}
