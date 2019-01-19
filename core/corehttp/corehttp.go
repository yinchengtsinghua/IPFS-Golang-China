
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
/*
包coreHTTP为WebUI、网关和其他
IPF的高级HTTP接口。
**/

package corehttp

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	core "github.com/ipfs/go-ipfs/core"
	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	"gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	periodicproc "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/periodic"
	manet "gx/ipfs/QmZcLBXKaFe8ND5YHPkJRAwmhJGrVsi1JqDZNyJ4nRK5Mj/go-multiaddr-net"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("core/server")

//ShutdownTimeout是在超时之后我们将停止等待挂起
//关闭时返回的命令。
const shutdownTimeout = 30 * time.Second

//servocation注册它在给定的mux上提供的任何HTTP处理程序。
//它返回mux以公开将来的选项，如果它
//有兴趣调解对未来选项或相同mux的请求
//如果没有，则最初传入。
type ServeOption func(*core.IpfsNode, net.Listener, *http.ServeMux) (*http.ServeMux, error)

//makehandler将一个servocations列表转换为一个实现
//所有给定的选项，依次。
func makeHandler(n *core.IpfsNode, l net.Listener, options ...ServeOption) (http.Handler, error) {
	topMux := http.NewServeMux()
	mux := topMux
	for _, option := range options {
		var err error
		mux, err = option(n, l, mux)
		if err != nil {
			return nil, err
		}
	}
	return topMux, nil
}

//listenandserve运行HTTP服务器，使用
//给定的发球选项。地址必须以multiaddr格式提供。
//
//要智能地解析其他格式的地址字符串，只要它们
//明确映射到有效的multiaddr。例如，为了方便起见，“：8080”应该
//映射到“/ip4/0.0.0.0/tcp/8080”。
func ListenAndServe(n *core.IpfsNode, listeningMultiAddr string, options ...ServeOption) error {
	addr, err := ma.NewMultiaddr(listeningMultiAddr)
	if err != nil {
		return err
	}

	list, err := manet.Listen(addr)
	if err != nil {
		return err
	}

//我们可能听过/tcp/0-让我们看看我们列出的是什么
	addr = list.Multiaddr()
	fmt.Printf("API server listening on %s\n", addr)

	return Serve(n, manet.NetListener(list), options...)
}

func Serve(node *core.IpfsNode, lis net.Listener, options ...ServeOption) error {
//不管怎样，一定要把这个关上。
	defer lis.Close()

	handler, err := makeHandler(node, lis, options...)
	if err != nil {
		return err
	}

	addr, err := manet.FromNetAddr(lis.Addr())
	if err != nil {
		return err
	}

	select {
	case <-node.Process().Closing():
		return fmt.Errorf("failed to start server, process closing")
	default:
	}

	server := &http.Server{
		Handler: handler,
	}

	var serverError error
	serverProc := node.Process().Go(func(p goprocess.Process) {
		serverError = server.Serve(lis)
	})

//等待服务器退出。
	select {
	case <-serverProc.Closed():
//如果在服务器退出之前关闭节点，请关闭服务器
	case <-node.Process().Closing():
		log.Infof("server at %s terminating...", addr)

		warnProc := periodicproc.Tick(5*time.Second, func(_ goprocess.Process) {
			log.Infof("waiting for server at %s to terminate...", addr)
		})

//如果我们的所有命令
//正在遵守它们的上下文，但我们应该有*一些*超时。
		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		err := server.Shutdown(ctx)

//应该已经关门了，但我们还得等着呢
//设置错误。
		<-serverProc.Closed()
		serverError = err

		warnProc.Close()
	}

	log.Infof("server at %s terminated", addr)
	return serverError
}
