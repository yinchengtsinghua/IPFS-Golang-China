
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
	"fmt"
	"io"
	"net/http"
	"net/url"
	gopath "path"
	"runtime/debug"
	"strings"
	"time"

	"github.com/ipfs/go-ipfs/core"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	"github.com/ipfs/go-ipfs/dagutils"

	"gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	"gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path/resolver"
	"gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	ft "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs"
	"gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/importer"
	chunker "gx/ipfs/QmR4QQVkBZsZENRjYFVi8dEtPL3daZRNKk24m4r6WKJHNm/go-ipfs-chunker"
	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	"gx/ipfs/QmTiRqrF5zkdZyrdsL5qndG1UbeWi8k8N2pYxCtXWrahR2/go-libp2p-routing"
	"gx/ipfs/QmXWZCd8jfaHmt4UDSnjKmGcrQMw95bDGWqEeVLVJjoANX/go-ipfs-files"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	"gx/ipfs/QmekxXDhCxCJRNuzmHreuaT3BsuJcsjcXWNrtV9C8DRHtd/go-multibase"
)

const (
	ipfsPathPrefix = "/ipfs/"
	ipnsPathPrefix = "/ipns/"
)

//gatewayhandler是一个为IPFS对象提供服务的HTTP处理程序（默认情况下可在/ipfs/<path>访问）
//（它提供诸如get/ipfs/qmvrzpkppzntsrezbfm2uzfxmpagnalke4dmcerbsggsafe/link等请求）
type gatewayHandler struct {
	node   *core.IpfsNode
	config GatewayConfig
	api    coreiface.CoreAPI
}

func newGatewayHandler(n *core.IpfsNode, c GatewayConfig, api coreiface.CoreAPI) *gatewayHandler {
	i := &gatewayHandler{
		node:   n,
		config: c,
		api:    api,
	}
	return i
}

//托多（Cryptix）：在其他地方找到这些助手
func (i *gatewayHandler) newDagFromReader(r io.Reader) (ipld.Node, error) {
//TODO（Cryptix）：合并pr1136后更改并删除此助手
//返回ufs.addFromReader（i.node，r.body）
	return importer.BuildDagFromReader(
		i.node.DAG,
		chunker.DefaultSplitter(r))
}

func (i *gatewayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//时间是一个艰难的倒退，我们不希望它发生，但以防万一。
	ctx, cancel := context.WithTimeout(r.Context(), time.Hour)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Error("A panic occurred in the gateway handler!")
			log.Error(r)
			debug.PrintStack()
		}
	}()

	if i.config.Writable {
		switch r.Method {
		case "POST":
			i.postHandler(ctx, w, r)
			return
		case "PUT":
			i.putHandler(w, r)
			return
		case "DELETE":
			i.deleteHandler(w, r)
			return
		}
	}

	if r.Method == "GET" || r.Method == "HEAD" {
		i.getOrHeadHandler(ctx, w, r)
		return
	}

	if r.Method == "OPTIONS" {
		i.optionsHandler(w, r)
		return
	}

	errmsg := "Method " + r.Method + " not allowed: "
	if !i.config.Writable {
		w.WriteHeader(http.StatusMethodNotAllowed)
		errmsg = errmsg + "read only access"
	} else {
		w.WriteHeader(http.StatusBadRequest)
		errmsg = errmsg + "bad request for " + r.URL.Path
	}
	fmt.Fprint(w, errmsg)
}

func (i *gatewayHandler) optionsHandler(w http.ResponseWriter, r *http.Request) {
 /*
  选项是浏览器用来检查的NOOP请求
  如果服务器接受跨站点XMLHttpRequest（由CORS头的存在指示）
  https://developer.mozilla.org/en-us/docs/web/http/access-control-cors预燃请求
 **/

i.addUserHeaders(w) //返回所有自定义头（包括CORS头，如果设置）
}

func (i *gatewayHandler) getOrHeadHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	escapedURLPath := r.URL.EscapedPath()

//如果网关位于反向代理之后并安装在子路径上，
//前缀头可以设置为向此子路径发送信号。
//它将预先添加到目录列表和index.html重定向中的链接。
	prefix := ""
	if prfx := r.Header.Get("X-Ipfs-Gateway-Prefix"); len(prfx) > 0 {
		for _, p := range i.config.PathPrefixes {
			if prfx == p || strings.HasPrefix(prfx, p+"/") {
				prefix = prfx
				break
			}
		}
	}

//ipnshostnameoption可能已使用主机头构造了IPN路径。
//在这种情况下，我们需要原始路径来构造重定向
//以及与请求的URL匹配的链接。
//例如，http://example.net将变为/ipns/example.net，并且
//重定向和链接将以http://example.net/ipns/example.net结尾。
	originalUrlPath := prefix + urlPath
	ipnsHostname := false
	if hdr := r.Header.Get("X-Ipns-Original-Path"); len(hdr) > 0 {
		originalUrlPath = prefix + hdr
		ipnsHostname = true
	}

	parsedPath, err := coreiface.ParsePath(urlPath)
	if err != nil {
		webError(w, "invalid ipfs path", err, http.StatusBadRequest)
		return
	}

//解析etag的最后一个dag节点的路径
	resolvedPath, err := i.api.ResolvePath(ctx, parsedPath)
	if err == coreiface.ErrOffline && !i.node.OnlineMode() {
		webError(w, "ipfs resolve -r "+escapedURLPath, err, http.StatusServiceUnavailable)
		return
	} else if err != nil {
		webError(w, "ipfs resolve -r "+escapedURLPath, err, http.StatusNotFound)
		return
	}

	dr, err := i.api.Unixfs().Get(ctx, resolvedPath)
	if err != nil {
		webError(w, "ipfs cat "+escapedURLPath, err, http.StatusNotFound)
		return
	}

	defer dr.Close()

//检查ETag发送回我们
	etag := "\"" + resolvedPath.Cid().String() + "\""
	if r.Header.Get("If-None-Match") == etag || r.Header.Get("If-None-Match") == "W/"+etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}

i.addUserHeaders(w) //好，现在写用户的头。
	w.Header().Set("X-IPFS-Path", urlPath)
	w.Header().Set("Etag", etag)

//Suborigin头文件，沙盒应用程序在浏览器中相互连接（甚至
//尽管它们来自同一个网关域）。
//
//例如，如果路径由ipnshostNameOption（）处理，则省略
//http://example.net/的请求将更改为/ipns/example.net/，
//会变成一个不正确的子装配头。
//在这种情况下，正确的做法是省略头部，因为它已经
//正确处理，无子装配。
//
//注意：浏览器还没有广泛支持这一点。
	if !ipnsHostname {
//例如：1=“ipfs”，2=“qmyunakwy…”，…
		pathComponents := strings.SplitN(urlPath, "/", 4)

		var suboriginRaw []byte
		cidDecoded, err := cid.Decode(pathComponents[2])
		if err != nil {
//组件2不使用cid解码，因此它必须是主机名
			suboriginRaw = []byte(strings.ToLower(pathComponents[2]))
		} else {
			suboriginRaw = cidDecoded.Bytes()
		}

		base32Encoded, err := multibase.Encode(multibase.Base32, suboriginRaw)
		if err != nil {
			internalWebError(w, err)
			return
		}

		suborigin := pathComponents[1] + "000" + strings.ToLower(base32Encoded)
		w.Header().Set("Suborigin", suborigin)
	}

//在错误之后设置这些标题，因为我们可能没有它
//不希望客户端缓存500个响应…
//只有当它是/ipfs的时候！
//TODO:当我们拆分/ipfs/ipns路由时，将其打破。
	modtime := time.Now()

	if f, ok := dr.(files.File); ok {
		if strings.HasPrefix(urlPath, ipfsPathPrefix) {
			w.Header().Set("Cache-Control", "public, max-age=29030400, immutable")

//将modtime设置为非常长的时间以前，因为文件是不可变的，应该保持缓存
			modtime = time.Unix(1, 0)
		}

		urlFilename := r.URL.Query().Get("filename")
		var name string
		if urlFilename != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("inline; filename*=UTF-8''%s", url.PathEscape(urlFilename)))
			name = urlFilename
		} else {
			name = getFilename(urlPath)
		}
		i.serveFile(w, r, name, modtime, f)
		return
	}
	dir, ok := dr.(files.Directory)
	if !ok {
		internalWebError(w, fmt.Errorf("unsupported file type"))
		return
	}

	idx, err := i.api.Unixfs().Get(ctx, coreiface.Join(resolvedPath, "index.html"))
	switch err.(type) {
	case nil:
		dirwithoutslash := urlPath[len(urlPath)-1] != '/'
		goget := r.URL.Query().Get("go-get") == "1"
		if dirwithoutslash && !goget {
//请参见上面声明originalPath的注释。
			http.Redirect(w, r, originalUrlPath+"/", 302)
			return
		}

		f, ok := idx.(files.File)
		if !ok {
			internalWebError(w, files.ErrNotReader)
			return
		}

//写入请求
		http.ServeContent(w, r, "index.html", modtime, f)
		return
	case resolver.ErrNoLink:
//没有index.html；noop
	default:
		internalWebError(w, err)
		return
	}

	if r.Method == "HEAD" {
		return
	}

//目录列表存储
	var dirListing []directoryItem
	dirit := dir.Entries()
	for dirit.Next() {
//请参见上面声明originalPath的注释。
		s, err := dirit.Node().Size()
		if err != nil {
			internalWebError(w, err)
			return
		}

		di := directoryItem{humanize.Bytes(uint64(s)), dirit.Name(), gopath.Join(originalUrlPath, dirit.Name())}
		dirListing = append(dirListing, di)
	}
	if dirit.Err() != nil {
		internalWebError(w, dirit.Err())
		return
	}

//构造正确的回链
//https://github.com/ipfs/go-ipfs/issues/1365
	var backLink string = prefix + urlPath

//不要超过/ipfs/$hash/
	pathSplit := path.SplitList(backLink)
	switch {
//保持反向链接
case len(pathSplit) == 3: //URL:/ipfs/$hash

//保持反向链接
case len(pathSplit) == 4 && pathSplit[3] == "": //网址：/ipfs/$hash/

//根据路径是否以斜线结束添加正确的链接
	default:
		if strings.HasSuffix(backLink, "/") {
			backLink += "./.."
		} else {
			backLink += "/.."
		}
	}

//如果ipnshostnameoption接触到路径，则从反向链接中除去/ipfs/$hash。
	if ipnsHostname {
		backLink = prefix + "/"
		if len(pathSplit) > 5 {
//同时剥离尾随段，因为它是一个反向链接
			backLinkParts := pathSplit[3 : len(pathSplit)-2]
			backLink += path.Join(backLinkParts) + "/"
		}
	}

	var hash string
	if !strings.HasPrefix(originalUrlPath, ipfsPathPrefix) {
		hash = resolvedPath.Cid().String()
	}

//请参见上面声明originalPath的注释。
	tplData := listingTemplateData{
		Listing:  dirListing,
		Path:     originalUrlPath,
		BackLink: backLink,
		Hash:     hash,
	}
	err = listingTemplate.Execute(w, tplData)
	if err != nil {
		internalWebError(w, err)
		return
	}
}

type sizeReadSeeker interface {
	Size() (int64, error)

	io.ReadSeeker
}

type sizeSeeker struct {
	sizeReadSeeker
}

func (s *sizeSeeker) Seek(offset int64, whence int) (int64, error) {
	if whence == io.SeekEnd && offset == 0 {
		return s.Size()
	}

	return s.sizeReadSeeker.Seek(offset, whence)
}

func (i *gatewayHandler) serveFile(w http.ResponseWriter, req *http.Request, name string, modtime time.Time, content io.ReadSeeker) {
	if sp, ok := content.(sizeReadSeeker); ok {
		content = &sizeSeeker{
			sizeReadSeeker: sp,
		}
	}

	http.ServeContent(w, req, name, modtime, content)
}

func (i *gatewayHandler) postHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	p, err := i.api.Unixfs().Add(ctx, files.NewReaderFile(r.Body))
	if err != nil {
		internalWebError(w, err)
		return
	}

i.addUserHeaders(w) //好，现在写用户的头。
	w.Header().Set("IPFS-Hash", p.Cid().String())
	http.Redirect(w, r, p.String(), http.StatusCreated)
}

func (i *gatewayHandler) putHandler(w http.ResponseWriter, r *http.Request) {
//TODO（cryptix）：将我移至servehtp并传递到所有处理程序
	ctx, cancel := context.WithCancel(i.node.Context())
	defer cancel()

	rootPath, err := path.ParsePath(r.URL.Path)
	if err != nil {
		webError(w, "putHandler: IPFS path not valid", err, http.StatusBadRequest)
		return
	}

	rsegs := rootPath.Segments()
	if rsegs[0] == ipnsPathPrefix {
		webError(w, "putHandler: updating named entries not supported", errors.New("WritableGateway: ipns put not supported"), http.StatusBadRequest)
		return
	}

	var newnode ipld.Node
	if rsegs[len(rsegs)-1] == "QmUNLLsPACCz1vLxQVkXqqLX5R1X345qqfHbsf67hvA3Nn" {
		newnode = ft.EmptyDirNode()
	} else {
		putNode, err := i.newDagFromReader(r.Body)
		if err != nil {
			webError(w, "putHandler: Could not create DAG from request", err, http.StatusInternalServerError)
			return
		}
		newnode = putNode
	}

	var newPath string
	if len(rsegs) > 1 {
		newPath = path.Join(rsegs[2:])
	}

	var newcid cid.Cid
	rnode, err := core.Resolve(ctx, i.node.Namesys, i.node.Resolver, rootPath)
	switch ev := err.(type) {
	case resolver.ErrNoLink:
//ev.node<解析失败的节点
//ev.name<新建链接
//但我们需要从根部修补
		c, err := cid.Decode(rsegs[1])
		if err != nil {
			webError(w, "putHandler: bad input path", err, http.StatusBadRequest)
			return
		}

		rnode, err := i.node.DAG.Get(ctx, c)
		if err != nil {
			webError(w, "putHandler: Could not create DAG from request", err, http.StatusInternalServerError)
			return
		}

		pbnd, ok := rnode.(*dag.ProtoNode)
		if !ok {
			webError(w, "Cannot read non protobuf nodes through gateway", dag.ErrNotProtobuf, http.StatusBadRequest)
			return
		}

		e := dagutils.NewDagEditor(pbnd, i.node.DAG)
		err = e.InsertNodeAtPath(ctx, newPath, newnode, ft.EmptyDirNode)
		if err != nil {
			webError(w, "putHandler: InsertNodeAtPath failed", err, http.StatusInternalServerError)
			return
		}

		nnode, err := e.Finalize(ctx, i.node.DAG)
		if err != nil {
			webError(w, "putHandler: could not get node", err, http.StatusInternalServerError)
			return
		}

		newcid = nnode.Cid()

	case nil:
		pbnd, ok := rnode.(*dag.ProtoNode)
		if !ok {
			webError(w, "Cannot read non protobuf nodes through gateway", dag.ErrNotProtobuf, http.StatusBadRequest)
			return
		}

		pbnewnode, ok := newnode.(*dag.ProtoNode)
		if !ok {
			webError(w, "Cannot read non protobuf nodes through gateway", dag.ErrNotProtobuf, http.StatusBadRequest)
			return
		}

//对象集数据事例
		pbnd.SetData(pbnewnode.Data())

		newcid = pbnd.Cid()
		err = i.node.DAG.Add(ctx, pbnd)
		if err != nil {
			nnk := newnode.Cid()
			webError(w, fmt.Sprintf("putHandler: Could not add newnode(%q) to root(%q)", nnk.String(), newcid.String()), err, http.StatusInternalServerError)
			return
		}
	default:
		webError(w, "could not resolve root DAG", ev, http.StatusInternalServerError)
		return
	}

i.addUserHeaders(w) //好，现在写用户的头。
	w.Header().Set("IPFS-Hash", newcid.String())
	http.Redirect(w, r, gopath.Join(ipfsPathPrefix, newcid.String(), newPath), http.StatusCreated)
}

func (i *gatewayHandler) deleteHandler(w http.ResponseWriter, r *http.Request) {
	urlPath := r.URL.Path
	ctx, cancel := context.WithCancel(i.node.Context())
	defer cancel()

	p, err := path.ParsePath(urlPath)
	if err != nil {
		webError(w, "failed to parse path", err, http.StatusBadRequest)
		return
	}

	c, components, err := path.SplitAbsPath(p)
	if err != nil {
		webError(w, "Could not split path", err, http.StatusInternalServerError)
		return
	}

	tctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	rootnd, err := i.node.Resolver.DAG.Get(tctx, c)
	if err != nil {
		webError(w, "Could not resolve root object", err, http.StatusBadRequest)
		return
	}

	pathNodes, err := i.node.Resolver.ResolveLinks(tctx, rootnd, components[:len(components)-1])
	if err != nil {
		webError(w, "Could not resolve parent object", err, http.StatusBadRequest)
		return
	}

	pbnd, ok := pathNodes[len(pathNodes)-1].(*dag.ProtoNode)
	if !ok {
		webError(w, "Cannot read non protobuf nodes through gateway", dag.ErrNotProtobuf, http.StatusBadRequest)
		return
	}

//TODO（cyrptix）：假设len（pathnodes）>1-未找到是上面的错误吗？
	err = pbnd.RemoveNodeLink(components[len(components)-1])
	if err != nil {
		webError(w, "Could not delete link", err, http.StatusBadRequest)
		return
	}

	var newnode *dag.ProtoNode = pbnd
	for j := len(pathNodes) - 2; j >= 0; j-- {
		if err := i.node.DAG.Add(ctx, newnode); err != nil {
			webError(w, "Could not add node", err, http.StatusInternalServerError)
			return
		}

		pathpb, ok := pathNodes[j].(*dag.ProtoNode)
		if !ok {
			webError(w, "Cannot read non protobuf nodes through gateway", dag.ErrNotProtobuf, http.StatusBadRequest)
			return
		}

		newnode, err = pathpb.UpdateNodeLink(components[j], newnode)
		if err != nil {
			webError(w, "Could not update node links", err, http.StatusInternalServerError)
			return
		}
	}

	if err := i.node.DAG.Add(ctx, newnode); err != nil {
		webError(w, "Could not add root node", err, http.StatusInternalServerError)
		return
	}

//重定向到新路径
	ncid := newnode.Cid()

i.addUserHeaders(w) //好，现在写用户的头。
	w.Header().Set("IPFS-Hash", ncid.String())
	http.Redirect(w, r, gopath.Join(ipfsPathPrefix+ncid.String(), path.Join(components[:len(components)-1])), http.StatusCreated)
}

func (i *gatewayHandler) addUserHeaders(w http.ResponseWriter) {
	for k, v := range i.config.Headers {
		w.Header()[k] = v
	}
}

func webError(w http.ResponseWriter, message string, err error, defaultCode int) {
	if _, ok := err.(resolver.ErrNoLink); ok {
		webErrorWithCode(w, message, err, http.StatusNotFound)
	} else if err == routing.ErrNotFound {
		webErrorWithCode(w, message, err, http.StatusNotFound)
	} else if err == context.DeadlineExceeded {
		webErrorWithCode(w, message, err, http.StatusRequestTimeout)
	} else {
		webErrorWithCode(w, message, err, defaultCode)
	}
}

func webErrorWithCode(w http.ResponseWriter, message string, err error, code int) {
	w.WriteHeader(code)

	fmt.Fprintf(w, "%s: %s\n", message, err)
	if code >= 500 {
		log.Warningf("server error: %s: %s", err)
	}
}

//返回500个错误并记录
func internalWebError(w http.ResponseWriter, err error) {
	webErrorWithCode(w, "internalWebError", err, http.StatusInternalServerError)
}

func getFilename(s string) string {
	if (strings.HasPrefix(s, ipfsPathPrefix) || strings.HasPrefix(s, ipnsPathPrefix)) && strings.Count(gopath.Clean(s), "/") <= 2 {
//不希望将/ipns/ipfs.io中的ipfs.io视为文件名。
		return ""
	}
	return gopath.Base(s)
}
