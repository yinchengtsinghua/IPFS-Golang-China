
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corehttp

import (
	"context"
	"errors"
	"io/ioutil"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	version "github.com/ipfs/go-ipfs"
	core "github.com/ipfs/go-ipfs/core"
	coreunix "github.com/ipfs/go-ipfs/core/coreunix"
	namesys "github.com/ipfs/go-ipfs/namesys"
	nsopts "github.com/ipfs/go-ipfs/namesys/opts"
	repo "github.com/ipfs/go-ipfs/repo"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	id "gx/ipfs/QmYxivS34F2M2n44WQQnRHGAKS8aoRUxwGpi9wk4Cdn4Jf/go-libp2p/p2p/protocol/identify"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
	datastore "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
	syncds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/sync"
)

//`IPFS object new unixfs dir`
var emptyDir = "/ipfs/QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn"

type mockNamesys map[string]path.Path

func (m mockNamesys) Resolve(ctx context.Context, name string, opts ...nsopts.ResolveOpt) (value path.Path, err error) {
	cfg := nsopts.DefaultResolveOpts()
	for _, o := range opts {
		o(&cfg)
	}
	depth := cfg.Depth
	if depth == nsopts.UnlimitedDepth {
		depth = math.MaxUint64
	}
	for strings.HasPrefix(name, "/ipns/") {
		if depth <= 0 {
			return value, namesys.ErrResolveRecursion
		}
		depth--

		var ok bool
		value, ok = m[name]
		if !ok {
			return "", namesys.ErrResolveFailed
		}
		name = value.String()
	}
	return value, nil
}

func (m mockNamesys) ResolveAsync(ctx context.Context, name string, opts ...nsopts.ResolveOpt) <-chan namesys.Result {
	out := make(chan namesys.Result, 1)
	v, err := m.Resolve(ctx, name, opts...)
	out <- namesys.Result{Path: v, Err: err}
	close(out)
	return out
}

func (m mockNamesys) Publish(ctx context.Context, name ci.PrivKey, value path.Path) error {
	return errors.New("not implemented for mockNamesys")
}

func (m mockNamesys) PublishWithEOL(ctx context.Context, name ci.PrivKey, value path.Path, _ time.Time) error {
	return errors.New("not implemented for mockNamesys")
}

func (m mockNamesys) GetResolver(subs string) (namesys.Resolver, bool) {
	return nil, false
}

func newNodeWithMockNamesys(ns mockNamesys) (*core.IpfsNode, error) {
	c := config.Config{
		Identity: config.Identity{
PeerID: "QmTFauExutTsy4XP6JbMFcw2Wa9645HJt2bTqL6qYDCKfe", //脱机节点所需
		},
	}
	r := &repo.Mock{
		C: c,
		D: syncds.MutexWrap(datastore.NewMapDatastore()),
	}
	n, err := core.NewNode(context.Background(), &core.BuildCfg{Repo: r})
	if err != nil {
		return nil, err
	}
	n.Namesys = ns
	return n, nil
}

type delegatedHandler struct {
	http.Handler
}

func (dh *delegatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dh.Handler.ServeHTTP(w, r)
}

func doWithoutRedirect(req *http.Request) (*http.Response, error) {
	tag := "without-redirect"
	c := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return errors.New(tag)
		},
	}
	res, err := c.Do(req)
	if err != nil && !strings.Contains(err.Error(), tag) {
		return nil, err
	}
	return res, nil
}

func newTestServerAndNode(t *testing.T, ns mockNamesys) (*httptest.Server, *core.IpfsNode) {
	n, err := newNodeWithMockNamesys(ns)
	if err != nil {
		t.Fatal(err)
	}

	cfg, err := n.Repo.Config()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Gateway.PathPrefixes = []string{"/good-prefix"}

//这里需要这个变量，因为我们需要用
//侦听器和带有处理程序的服务器。YAY循环。
	dh := &delegatedHandler{}
	ts := httptest.NewServer(dh)

	dh.Handler, err = makeHandler(n,
		ts.Listener,
		IPNSHostnameOption(),
		GatewayOption(false, "/ipfs", "/ipns"),
		VersionOption(),
	)
	if err != nil {
		t.Fatal(err)
	}

	return ts, n
}

func TestGatewayGet(t *testing.T) {
	ns := mockNamesys{}
	ts, n := newTestServerAndNode(t, ns)
	defer ts.Close()

	k, err := coreunix.Add(n, strings.NewReader("fnord"))
	if err != nil {
		t.Fatal(err)
	}
	ns["/ipns/example.com"] = path.FromString("/ipfs/" + k)
	ns["/ipns/working.example.com"] = path.FromString("/ipfs/" + k)
	ns["/ipns/double.example.com"] = path.FromString("/ipns/working.example.com")
	ns["/ipns/triple.example.com"] = path.FromString("/ipns/double.example.com")
	ns["/ipns/broken.example.com"] = path.FromString("/ipns/" + k)
//我们选了。因为：
//1。这是一个有效的TLD。
//2。go将其视为“man”文件的文件扩展名（即使
//实际上没有人会使用这个扩展，阿法克）。
//
//不幸的是，这可能不适用于所有平台的文件类型
//检测依赖于平台。
	ns["/ipns/example.man"] = path.FromString("/ipfs/" + k)

	t.Log(ts.URL)
	for _, test := range []struct {
		host   string
		path   string
		status int
		text   string
	}{
		{"localhost:5001", "/", http.StatusNotFound, "404 page not found\n"},
		{"localhost:5001", "/" + k, http.StatusNotFound, "404 page not found\n"},
		{"localhost:5001", "/ipfs/" + k, http.StatusOK, "fnord"},
		{"localhost:5001", "/ipns/nxdomain.example.com", http.StatusNotFound, "ipfs resolve -r /ipns/nxdomain.example.com: " + namesys.ErrResolveFailed.Error() + "\n"},
		{"localhost:5001", "/ipns/%0D%0A%0D%0Ahello", http.StatusNotFound, "ipfs resolve -r /ipns/%0D%0A%0D%0Ahello: " + namesys.ErrResolveFailed.Error() + "\n"},
		{"localhost:5001", "/ipns/example.com", http.StatusOK, "fnord"},
		{"example.com", "/", http.StatusOK, "fnord"},

		{"working.example.com", "/", http.StatusOK, "fnord"},
		{"double.example.com", "/", http.StatusOK, "fnord"},
		{"triple.example.com", "/", http.StatusOK, "fnord"},
		{"working.example.com", "/ipfs/" + k, http.StatusNotFound, "ipfs resolve -r /ipns/working.example.com/ipfs/" + k + ": no link named \"ipfs\" under " + k + "\n"},
		{"broken.example.com", "/", http.StatusNotFound, "ipfs resolve -r /ipns/broken.example.com/: " + namesys.ErrResolveFailed.Error() + "\n"},
		{"broken.example.com", "/ipfs/" + k, http.StatusNotFound, "ipfs resolve -r /ipns/broken.example.com/ipfs/" + k + ": " + namesys.ErrResolveFailed.Error() + "\n"},
//这个测试用例确保我们不把TLD当作文件扩展名。
		{"example.man", "/", http.StatusOK, "fnord"},
	} {
		var c http.Client
		r, err := http.NewRequest("GET", ts.URL+test.path, nil)
		if err != nil {
			t.Fatal(err)
		}
		r.Host = test.host
		resp, err := c.Do(r)

urlstr := "http://“+test.host+test.path
		if err != nil {
			t.Errorf("error requesting %s: %s", urlstr, err)
			continue
		}
		defer resp.Body.Close()
		contentType := resp.Header.Get("Content-Type")
		if contentType != "text/plain; charset=utf-8" {
			t.Errorf("expected content type to be text/plain, got %s", contentType)
		}
		if resp.StatusCode != test.status {
			t.Errorf("got %d, expected %d from %s", resp.StatusCode, test.status, urlstr)
			continue
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("error reading response from %s: %s", urlstr, err)
		}
		if string(body) != test.text {
			t.Errorf("unexpected response body from %s: expected %q; got %q", urlstr, test.text, body)
			continue
		}
	}
}

func TestIPNSHostnameRedirect(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ns := mockNamesys{}
	ts, n := newTestServerAndNode(t, ns)
	t.Logf("test server url: %s", ts.URL)
	defer ts.Close()

//创建/ipns/example.net/foo/index.html
	_, dagn1, err := coreunix.AddWrapped(n, strings.NewReader("_"), "_")
	if err != nil {
		t.Fatal(err)
	}

	_, dagn2, err := coreunix.AddWrapped(n, strings.NewReader("_"), "index.html")
	if err != nil {
		t.Fatal(err)
	}

	dagn1.(*dag.ProtoNode).AddNodeLink("foo", dagn2)
	if err != nil {
		t.Fatal(err)
	}

	err = n.DAG.Add(ctx, dagn2)
	if err != nil {
		t.Fatal(err)
	}

	err = n.DAG.Add(ctx, dagn1)
	if err != nil {
		t.Fatal(err)
	}

	k := dagn1.Cid()
	t.Logf("k: %s\n", k)
	ns["/ipns/example.net"] = path.FromString("/ipfs/" + k.String())

//向包含index.html的目录发出请求
	req, err := http.NewRequest("GET", ts.URL+"/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"

	res, err := doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//预期302重定向到相同的路径，但带有尾随斜杠
	if res.StatusCode != 302 {
		t.Errorf("status is %d, expected 302", res.StatusCode)
	}
	hdr := res.Header["Location"]
	if len(hdr) < 1 {
		t.Errorf("location header not present")
	} else if hdr[0] != "/foo/" {
		t.Errorf("location header is %v, expected /foo/", hdr[0])
	}

//向包含index.html的目录发出前缀为的请求
	req, err = http.NewRequest("GET", ts.URL+"/foo", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"
	req.Header.Set("X-Ipfs-Gateway-Prefix", "/good-prefix")

	res, err = doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//
	if res.StatusCode != 302 {
		t.Errorf("status is %d, expected 302", res.StatusCode)
	}
	hdr = res.Header["Location"]
	if len(hdr) < 1 {
		t.Errorf("location header not present")
	} else if hdr[0] != "/good-prefix/foo/" {
		t.Errorf("location header is %v, expected /good-prefix/foo/", hdr[0])
	}

//确保/版本未暴露
	req, err = http.NewRequest("GET", ts.URL+"/version", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"
	req.Header.Set("X-Ipfs-Gateway-Prefix", "/good-prefix")

	res, err = doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 404 {
		t.Fatalf("expected a 404 error, got: %s", res.Status)
	}
}

func TestIPNSHostnameBacklinks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ns := mockNamesys{}
	ts, n := newTestServerAndNode(t, ns)
	t.Logf("test server url: %s", ts.URL)
	defer ts.Close()

//创建/ipns/example.net/foo/
	_, dagn1, err := coreunix.AddWrapped(n, strings.NewReader("1"), "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, dagn2, err := coreunix.AddWrapped(n, strings.NewReader("2"), "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	_, dagn3, err := coreunix.AddWrapped(n, strings.NewReader("3"), "file.txt")
	if err != nil {
		t.Fatal(err)
	}
	dagn2.(*dag.ProtoNode).AddNodeLink("bar", dagn3)
	dagn1.(*dag.ProtoNode).AddNodeLink("foo? #<'", dagn2)
	if err != nil {
		t.Fatal(err)
	}

	err = n.DAG.Add(ctx, dagn3)
	if err != nil {
		t.Fatal(err)
	}
	err = n.DAG.Add(ctx, dagn2)
	if err != nil {
		t.Fatal(err)
	}
	err = n.DAG.Add(ctx, dagn1)
	if err != nil {
		t.Fatal(err)
	}

	k := dagn1.Cid()
	t.Logf("k: %s\n", k)
	ns["/ipns/example.net"] = path.FromString("/ipfs/" + k.String())

//请求目录列表
	req, err := http.NewRequest("GET", ts.URL+"/foo%3F%20%23%3C%27/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"

	res, err := doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//期望正确的反向链接
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("error reading response: %s", err)
	}
	s := string(body)
	t.Logf("body: %s\n", string(body))

	if !strings.Contains(s, "Index of /foo? #&lt;&#39;/") {
		t.Fatalf("expected a path in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/\">") {
		t.Fatalf("expected backlink in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/foo%3F%20%23%3C%27/file.txt\">") {
		t.Fatalf("expected file in directory listing")
	}
	if !strings.Contains(s, dagn2.Cid().String()) {
		t.Fatalf("expected hash in directory listing")
	}

//请求根目录列表
	req, err = http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"

	res, err = doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//期望根处的反向链接正确
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("error reading response: %s", err)
	}
	s = string(body)
	t.Logf("body: %s\n", string(body))

	if !strings.Contains(s, "Index of /") {
		t.Fatalf("expected a path in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/\">") {
		t.Fatalf("expected backlink in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/file.txt\">") {
		t.Fatalf("expected file in directory listing")
	}
	if !strings.Contains(s, dagn1.Cid().String()) {
		t.Fatalf("expected hash in directory listing")
	}

//请求目录列表
	req, err = http.NewRequest("GET", ts.URL+"/foo%3F%20%23%3C%27/bar/", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"

	res, err = doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//期望正确的反向链接
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("error reading response: %s", err)
	}
	s = string(body)
	t.Logf("body: %s\n", string(body))

	if !strings.Contains(s, "Index of /foo? #&lt;&#39;/bar/") {
		t.Fatalf("expected a path in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/foo%3F%20%23%3C%27/\">") {
		t.Fatalf("expected backlink in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/foo%3F%20%23%3C%27/bar/file.txt\">") {
		t.Fatalf("expected file in directory listing")
	}
	if !strings.Contains(s, dagn3.Cid().String()) {
		t.Fatalf("expected hash in directory listing")
	}

//以前缀向目录列表发出请求
	req, err = http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"
	req.Header.Set("X-Ipfs-Gateway-Prefix", "/good-prefix")

	res, err = doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//期望具有前缀的正确反向链接
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("error reading response: %s", err)
	}
	s = string(body)
	t.Logf("body: %s\n", string(body))

	if !strings.Contains(s, "Index of /good-prefix") {
		t.Fatalf("expected a path in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/good-prefix/\">") {
		t.Fatalf("expected backlink in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/good-prefix/file.txt\">") {
		t.Fatalf("expected file in directory listing")
	}
	if !strings.Contains(s, dagn1.Cid().String()) {
		t.Fatalf("expected hash in directory listing")
	}

//以非法前缀请求目录列表
	req, err = http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"
	req.Header.Set("X-Ipfs-Gateway-Prefix", "/bad-prefix")

//请求以邪恶前缀列出目录
	req, err = http.NewRequest("GET", ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.net"
req.Header.Set("X-Ipfs-Gateway-Prefix", "//好前缀/foo“）

	res, err = doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//期望没有非法前缀的正确反向链接
	body, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("error reading response: %s", err)
	}
	s = string(body)
	t.Logf("body: %s\n", string(body))

	if !strings.Contains(s, "Index of /") {
		t.Fatalf("expected a path in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/\">") {
		t.Fatalf("expected backlink in directory listing")
	}
	if !strings.Contains(s, "<a href=\"/file.txt\">") {
		t.Fatalf("expected file in directory listing")
	}
	if !strings.Contains(s, dagn1.Cid().String()) {
		t.Fatalf("expected hash in directory listing")
	}
}

func TestCacheControlImmutable(t *testing.T) {
	ts, _ := newTestServerAndNode(t, nil)
	t.Logf("test server url: %s", ts.URL)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+emptyDir+"/", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

//检查是否未设置不可变标记
	hdrs, ok := res.Header["Cache-Control"]
	if ok {
		for _, hdr := range hdrs {
			if strings.Contains(hdr, "immutable") {
				t.Fatalf("unexpected Cache-Control: immutable on directory listing: %s", hdr)
			}
		}
	}
}

func TestGoGetSupport(t *testing.T) {
	ts, _ := newTestServerAndNode(t, nil)
	t.Logf("test server url: %s", ts.URL)
	defer ts.Close()

//模仿棋
	req, err := http.NewRequest("GET", ts.URL+emptyDir+"?go-get=1", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != 200 {
		t.Errorf("status is %d, expected 200", res.StatusCode)
	}
}

func TestVersion(t *testing.T) {
	version.CurrentCommit = "theshortcommithash"

	ns := mockNamesys{}
	ts, _ := newTestServerAndNode(t, ns)
	t.Logf("test server url: %s", ts.URL)
	defer ts.Close()

	req, err := http.NewRequest("GET", ts.URL+"/version", nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := doWithoutRedirect(req)
	if err != nil {
		t.Fatal(err)
	}
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("error reading response: %s", err)
	}
	s := string(body)

	if !strings.Contains(s, "Commit: theshortcommithash") {
		t.Fatalf("response doesn't contain commit:\n%s", s)
	}

	if !strings.Contains(s, "Client Version: "+id.ClientVersion) {
		t.Fatalf("response doesn't contain client version:\n%s", s)
	}

	if !strings.Contains(s, "Protocol Version: "+id.LibP2PVersion) {
		t.Fatalf("response doesn't contain protocol version:\n%s", s)
	}
}
