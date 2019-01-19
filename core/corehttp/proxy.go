
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corehttp

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	core "github.com/ipfs/go-ipfs/core"

	p2phttp "gx/ipfs/QmQz2LTeFhCwFthG1r28ETquLtVk9oNzqPdB4DnTaya4eH/go-libp2p-http"
	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
)

//proxyoption是一个端点，用于将HTTP请求代理到另一个IPF对等端。
func ProxyOption() ServeOption {
	return func(ipfsNode *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.HandleFunc("/p2p/", func(w http.ResponseWriter, request *http.Request) {
//解析请求
			parsedRequest, err := parseRequest(request)
			if err != nil {
				handleError(w, "failed to parse request", err, 400)
				return
			}

request.Host = "" //以URL的主机为准。
			request.URL.Path = parsedRequest.httpPath
target, err := url.Parse(fmt.Sprintf("libp2p://%s“，parsedRequest.target”）。
			if err != nil {
				handleError(w, "failed to parse url", err, 400)
				return
			}

			rt := p2phttp.NewTransport(ipfsNode.PeerHost, p2phttp.ProtocolOption(parsedRequest.name))
			proxy := httputil.NewSingleHostReverseProxy(target)
			proxy.Transport = rt
			proxy.ServeHTTP(w, request)
		})
		return mux, nil
	}
}

type proxyRequest struct {
	target   string
	name     protocol.ID
httpPath string //发送到代理主机的路径
}

//从URL路径分析对等端ID、名称和HTTP路径
///p2p/$peer_id/http/$http_路径
//或
///p2p/$peer_id/x/$protocol/http/$http_路径
func parseRequest(request *http.Request) (*proxyRequest, error) {
	path := request.URL.Path

	split := strings.SplitN(path, "/", 5)
	if len(split) < 5 {
		return nil, fmt.Errorf("Invalid request path '%s'", path)
	}

	if split[3] == "http" {
		return &proxyRequest{split[2], protocol.ID("/http"), split[4]}, nil
	}

	split = strings.SplitN(path, "/", 7)
	if split[3] != "x" || split[5] != "http" {
		return nil, fmt.Errorf("Invalid request path '%s'", path)
	}

	return &proxyRequest{split[2], protocol.ID("/x/" + split[4] + "/http"), split[6]}, nil
}

func handleError(w http.ResponseWriter, msg string, err error, code int) {
	w.WriteHeader(code)
	fmt.Fprintf(w, "%s: %s\n", msg, err)
	log.Warningf("http proxy error: %s: %s", err)
}
