
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	madns "gx/ipfs/QmQc7jbDUsxUJZyFJzxVrnrWeECCct6fErEpMqtjyWvCX8/go-multiaddr-dns"
)

var (
	ctx         = context.Background()
	testAddr, _ = ma.NewMultiaddr("/dns4/example.com/tcp/5001")
)

func makeResolver(n uint8) *madns.Resolver {
	results := make([]net.IPAddr, n)
	for i := uint8(0); i < n; i++ {
		results[i] = net.IPAddr{IP: net.ParseIP(fmt.Sprintf("192.0.2.%d", i))}
	}

	backend := &madns.MockBackend{
		IP: map[string][]net.IPAddr{
			"example.com": results,
		}}

	return &madns.Resolver{
		Backend: backend,
	}
}

func TestApiEndpointResolveDNSOneResult(t *testing.T) {
	dnsResolver = makeResolver(1)

	addr, err := resolveAddr(ctx, testAddr)
	if err != nil {
		t.Error(err)
	}

	if ref, _ := ma.NewMultiaddr("/ip4/192.0.2.0/tcp/5001"); !addr.Equal(ref) {
		t.Errorf("resolved address was different than expected")
	}
}

func TestApiEndpointResolveDNSMultipleResults(t *testing.T) {
	dnsResolver = makeResolver(4)

	addr, err := resolveAddr(ctx, testAddr)
	if err != nil {
		t.Error(err)
	}

	if ref, _ := ma.NewMultiaddr("/ip4/192.0.2.0/tcp/5001"); !addr.Equal(ref) {
		t.Errorf("resolved address was different than expected")
	}
}

func TestApiEndpointResolveDNSNoResults(t *testing.T) {
	dnsResolver = makeResolver(0)

	addr, err := resolveAddr(ctx, testAddr)
	if addr != nil || err == nil {
		t.Error("expected test address not to resolve, and to throw an error")
	}

	if !strings.HasPrefix(err.Error(), "non-resolvable API endpoint") {
		t.Errorf("expected error not thrown; actual: %v", err)
	}
}
