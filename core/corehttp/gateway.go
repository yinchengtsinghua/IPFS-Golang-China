
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
	"sort"

	version "github.com/ipfs/go-ipfs"
	core "github.com/ipfs/go-ipfs/core"
	coreapi "github.com/ipfs/go-ipfs/core/coreapi"
	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	id "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p/p2p/protocol/identify"
)

type GatewayConfig struct {
	Headers      map[string][]string
	Writable     bool
	PathPrefixes []string
}

//清理一组头的助手函数：
//1。规范化。
//2。去重复。
//三。种类。
func cleanHeaderSet(headers []string) []string {
//消除重复和规范化。
	m := make(map[string]struct{}, len(headers))
	for _, h := range headers {
		m[http.CanonicalHeaderKey(h)] = struct{}{}
	}
	result := make([]string, 0, len(m))
	for k := range m {
		result = append(result, k)
	}

//排序
	sort.Strings(result)
	return result
}

func GatewayOption(writable bool, paths ...string) ServeOption {
	return func(n *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		cfg, err := n.Repo.Config()
		if err != nil {
			return nil, err
		}

		api, err := coreapi.NewCoreAPI(n, options.Api.FetchBlocks(!cfg.Gateway.NoFetch))
		if err != nil {
			return nil, err
		}

		headers := make(map[string][]string, len(cfg.Gateway.HTTPHeaders))
		for h, v := range cfg.Gateway.HTTPHeaders {
			headers[http.CanonicalHeaderKey(h)] = v
		}

//硬编码的标题。
		const ACAHeadersName = "Access-Control-Allow-Headers"
		const ACEHeadersName = "Access-Control-Expose-Headers"
		const ACAOriginName = "Access-Control-Allow-Origin"
		const ACAMethodsName = "Access-Control-Allow-Methods"

		if _, ok := headers[ACAOriginName]; !ok {
//默认为*全部*
			headers[ACAOriginName] = []string{"*"}
		}
		if _, ok := headers[ACAMethodsName]; !ok {
//默认获取
			headers[ACAMethodsName] = []string{"GET"}
		}

		headers[ACAHeadersName] = cleanHeaderSet(
			append([]string{
				"Content-Type",
				"User-Agent",
				"Range",
				"X-Requested-With",
			}, headers[ACAHeadersName]...))

		headers[ACEHeadersName] = cleanHeaderSet(
			append([]string{
				"Content-Range",
				"X-Chunked-Output",
				"X-Stream-Output",
			}, headers[ACEHeadersName]...))

		gateway := newGatewayHandler(n, GatewayConfig{
			Headers:      headers,
			Writable:     writable,
			PathPrefixes: cfg.Gateway.PathPrefixes,
		}, api)

		for _, p := range paths {
			mux.Handle(p+"/", gateway)
		}
		return mux, nil
	}
}

func VersionOption() ServeOption {
	return func(_ *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Commit: %s\n", version.CurrentCommit)
			fmt.Fprintf(w, "Client Version: %s\n", id.ClientVersion)
			fmt.Fprintf(w, "Protocol Version: %s\n", id.LibP2PVersion)
		})
		return mux, nil
	}
}
