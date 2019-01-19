
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//package main使用secio包提供netcat的实现。
//这意味着通道是加密的（和maced）。
//这是为了练习spipe包。
//用途：
//seccat[<local address>><remote address>
//seccat-l<本地地址>
//
//地址格式为：【主机】：端口
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"

	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	pstore "gx/ipfs/QmPiemjiKBC9VA7vZF82m4x1oygtg2c2YVqag8PX7dN1BD/go-libp2p-peerstore"
	pstoremem "gx/ipfs/QmPiemjiKBC9VA7vZF82m4x1oygtg2c2YVqag8PX7dN1BD/go-libp2p-peerstore/pstoremem"
	secio "gx/ipfs/QmQoDgEDBPJ6RfKDgdoMpPwoK4WEZHUosHoUs6YVDGoen3/go-libp2p-secio"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var verbose = false

//用法打印出此模块的用法。
//假定标志使用go stdlib标志包。
var Usage = func() {
	text := `seccat - secure netcat in Go

Usage:

  listen: %s [<local address>] <remote address>
  dial:   %s -l <local address>

Address format is Go's: [host]:port
`

	fmt.Fprintf(os.Stderr, text, os.Args[0], os.Args[0])
	flag.PrintDefaults()
}

type args struct {
	listen     bool
	verbose    bool
	debug      bool
	localAddr  string
	remoteAddr string
//关键文件字符串
	keybits int
}

func parseArgs() args {
	var a args

//设置+分析标志
	flag.BoolVar(&a.listen, "listen", false, "listen for connections")
	flag.BoolVar(&a.listen, "l", false, "listen for connections (short)")
	flag.BoolVar(&a.verbose, "v", true, "verbose")
	flag.BoolVar(&a.debug, "debug", false, "debugging")
//flag.stringvar（&A.key file，“key”，“”，“private key file”）。
	flag.IntVar(&a.keybits, "keybits", 2048, "num bits for generating private key")
	flag.Usage = Usage
	flag.Parse()
	osArgs := flag.Args()

	if len(osArgs) < 1 {
		exit("")
	}

	if a.verbose {
		out("verbose on")
	}

	if a.listen {
		a.localAddr = osArgs[0]
	} else {
		if len(osArgs) > 1 {
			a.localAddr = osArgs[0]
			a.remoteAddr = osArgs[1]
		} else {
			a.remoteAddr = osArgs[0]
		}
	}

	return a
}

func main() {
	args := parseArgs()
	verbose = args.verbose
	if args.debug {
		logging.SetDebugLogging()
	}

	go func() {
//等我们离开。
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGABRT)
		<-sigc
		panic("ABORT! ABORT! ABORT!")
	}()

	if err := connect(args); err != nil {
		exit("%s", err)
	}
}

func setupPeer(a args) (peer.ID, pstore.Peerstore, error) {
	if a.keybits < 1024 {
		return "", nil, errors.New("bitsize less than 1024 is considered unsafe")
	}

	out("generating key pair...")
	sk, pk, err := ci.GenerateKeyPair(ci.RSA, a.keybits)
	if err != nil {
		return "", nil, err
	}

	p, err := peer.IDFromPublicKey(pk)
	if err != nil {
		return "", nil, err
	}

	ps := pstoremem.NewPeerstore()
	ps.AddPrivKey(p, sk)
	ps.AddPubKey(p, pk)

	out("local peer id: %s", p)
	return p, ps, nil
}

func connect(args args) error {
	p, ps, err := setupPeer(args)
	if err != nil {
		return err
	}

	var conn net.Conn
	if args.listen {
		conn, err = Listen(args.localAddr)
	} else {
		conn, err = Dial(args.localAddr, args.remoteAddr)
	}
	if err != nil {
		return err
	}

//记录通过连接的所有信息
	rwc := &logConn{n: "conn", Conn: conn}

//好，我们来设置频道。
	sk := ps.PrivKey(p)
	sg, err := secio.New(sk)
	if err != nil {
		return err
	}
	sconn, err := sg.SecureInbound(context.TODO(), rwc)
	if err != nil {
		return err
	}
	out("remote peer id: %s", sconn.RemotePeer())
	netcat(sconn)
	return nil
}

//侦听侦听并接受给定端口上的一个传入UDT连接，
//并将所有传入数据传输到os.stdout。
func Listen(localAddr string) (net.Conn, error) {
	l, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, err
	}
	out("listening at %s", l.Addr())

	c, err := l.Accept()
	if err != nil {
		return nil, err
	}
	out("accepted connection from %s", c.RemoteAddr())

//听者完成
	l.Close()

	return c, nil
}

//拨号连接到远程地址，并将所有OS.stdin连接到远程端。
//如果设置了localaddr，则使用它进行拨号。
func Dial(localAddr, remoteAddr string) (net.Conn, error) {

	var laddr net.Addr
	var err error
	if localAddr != "" {
		laddr, err = net.ResolveTCPAddr("tcp", localAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve address %s", localAddr)
		}
	}

	if laddr != nil {
		out("dialing %s from %s", remoteAddr, laddr)
	} else {
		out("dialing %s", remoteAddr)
	}

	d := net.Dialer{LocalAddr: laddr}
	c, err := d.Dial("tcp", remoteAddr)
	if err != nil {
		return nil, err
	}
	out("connected to %s", c.RemoteAddr())

	return c, nil
}

func netcat(c io.ReadWriteCloser) {
	out("piping stdio to connection")

	done := make(chan struct{}, 2)

	go func() {
		n, _ := io.Copy(c, os.Stdin)
		out("sent %d bytes", n)
		done <- struct{}{}
	}()
	go func() {
		n, _ := io.Copy(os.Stdout, c)
		out("received %d bytes", n)
		done <- struct{}{}
	}()

//等我们离开。
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT,
		syscall.SIGTERM, syscall.SIGQUIT)

	select {
	case <-done:
	case <-sigc:
		return
	}

	c.Close()
}
