
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corehttp

import (
	"net/http"
	"strings"
	"testing"

	"github.com/ipfs/go-ipfs/thirdparty/assert"

	protocol "gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
)

type TestCase struct {
	urlprefix string
	target    string
	name      string
	path      string
}

var validtestCases = []TestCase{
{"http://本地主机：5001“，”qmt8jtu54xsmc38xsb1xhfsmm775vuteajg7lwwtawzxt“，/http”，“path/to/index.txt”，
{"http://本地主机：5001“，”qmt8jtu54xsmc38xsb1xhfsmm775vuteajg7lwwtawzxt“，/x/custom/http”，“path/to/index.txt”，
{"http://localhost:5001“，”qmt8jtu54xsmc38xsb1xhfsmm775vuteajg7lwwtawzxt“，/x/custom/http”，“http/path/to/index.txt”，
}

func TestParseRequest(t *testing.T) {
	for _, tc := range validtestCases {
		url := tc.urlprefix + "/p2p/" + tc.target + tc.name + "/" + tc.path
		req, _ := http.NewRequest("GET", url, strings.NewReader(""))

		parsed, err := parseRequest(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.True(parsed.httpPath == tc.path, t, "proxy request path")
		assert.True(parsed.name == protocol.ID(tc.name), t, "proxy request name")
		assert.True(parsed.target == tc.target, t, "proxy request peer-id")
	}
}

var invalidtestCases = []string{
"http://本地主机：5001/p2p/http/foobar“，
"http://本地主机：5001/p2p/qmt8jtu54xsmc38xsb1xhfsmm775vuteajg7lwwtawzxt/x/custom/foobar“，
}

func TestParseRequestInvalidPath(t *testing.T) {
	for _, tc := range invalidtestCases {
		url := tc
		req, _ := http.NewRequest("GET", url, strings.NewReader(""))

		_, err := parseRequest(req)
		if err == nil {
			t.Fail()
		}
	}
}
