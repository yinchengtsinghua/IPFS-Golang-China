
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package coreunix

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	gopath "path"
	"path/filepath"
	"strconv"

	"github.com/ipfs/go-ipfs/core"
	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	"github.com/ipfs/go-ipfs/pin"

	"gx/ipfs/QmP9eu5X5Ax8169jNWqAJcc42mdZgzLR1aKCEzqhNoBLKk/go-mfs"
	"gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs"
	"gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/importer/balanced"
	ihelper "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/importer/helpers"
	"gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/importer/trickle"
	chunker "gx/ipfs/QmR4QQVkBZsZENRjYFVi8dEtPL3daZRNKk24m4r6WKJHNm/go-ipfs-chunker"
	"gx/ipfs/QmR6YMs8EkXQLXNwQKxLnQp2VBZSepoEJ8KCZAyanJHhJu/go-ipfs-posinfo"
	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	bstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	"gx/ipfs/QmXWZCd8jfaHmt4UDSnjKmGcrQMw95bDGWqEeVLVJjoANX/go-ipfs-files"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("coreunix")

//发送进度更新消息前要等待多少字节的进度
const progressReaderIncrement = 1024 * 256

var liveCacheSize = uint64(256 << 10)

type Link struct {
	Name, Hash string
	Size       uint64
}

//new adder返回用于文件添加操作的新adder。
func NewAdder(ctx context.Context, p pin.Pinner, bs bstore.GCLocker, ds ipld.DAGService) (*Adder, error) {
	bufferedDS := ipld.NewBufferedDAG(ctx, ds)

	return &Adder{
		ctx:        ctx,
		pinning:    p,
		gcLocker:   bs,
		dagService: ds,
		bufferedDS: bufferedDS,
		Progress:   false,
		Hidden:     true,
		Pin:        true,
		Trickle:    false,
		Wrap:       false,
		Chunker:    "",
	}, nil
}

//adder保存传递给'add'命令的开关。
type Adder struct {
	ctx        context.Context
	pinning    pin.Pinner
	gcLocker   bstore.GCLocker
	dagService ipld.DAGService
	bufferedDS *ipld.BufferedDAG
	Out        chan<- interface{}
	Progress   bool
	Hidden     bool
	Pin        bool
	Trickle    bool
	RawLeaves  bool
	Silent     bool
	Wrap       bool
	Name       string
	NoCopy     bool
	Chunker    string
	root       ipld.Node
	mroot      *mfs.Root
	unlocker   bstore.Unlocker
	tempRoot   cid.Cid
	CidBuilder cid.Builder
	liveNodes  uint64
}

func (adder *Adder) mfsRoot() (*mfs.Root, error) {
	if adder.mroot != nil {
		return adder.mroot, nil
	}
	rnode := unixfs.EmptyDirNode()
	rnode.SetCidBuilder(adder.CidBuilder)
	mr, err := mfs.NewRoot(adder.ctx, adder.dagService, rnode, nil)
	if err != nil {
		return nil, err
	}
	adder.mroot = mr
	return adder.mroot, nil
}

//setmfsroot将'r'设置为加法器的根。
func (adder *Adder) SetMfsRoot(r *mfs.Root) {
	adder.mroot = r
}

//从读卡器数据构造一个节点，并添加它。不针。
func (adder *Adder) add(reader io.Reader) (ipld.Node, error) {
	chnk, err := chunker.FromString(reader, adder.Chunker)
	if err != nil {
		return nil, err
	}

//确保完成后写入所有添加的节点。
	defer adder.bufferedDS.Commit()

	params := ihelper.DagBuilderParams{
		Dagserv:    adder.bufferedDS,
		RawLeaves:  adder.RawLeaves,
		Maxlinks:   ihelper.DefaultLinksPerBlock,
		NoCopy:     adder.NoCopy,
		CidBuilder: adder.CidBuilder,
	}

	if adder.Trickle {
		return trickle.Layout(params.New(chnk))
	}

	return balanced.Layout(params.New(chnk))
}

//root node返回添加的根节点。
func (adder *Adder) RootNode() (ipld.Node, error) {
//为了记忆
	if adder.root != nil {
		return adder.root, nil
	}

	mr, err := adder.mfsRoot()
	if err != nil {
		return nil, err
	}
	root, err := mr.GetDirectory().GetNode()
	if err != nil {
		return nil, err
	}

//如果不包装，并且只有一个根文件，则将该哈希用作根文件。
	if !adder.Wrap && len(root.Links()) == 1 {
		nd, err := root.Links()[0].GetNode(adder.ctx, adder.dagService)
		if err != nil {
			return nil, err
		}

		root = nd
	}

	adder.root = root
	return root, err
}

//递归地固定加法器的根节点和
//将pin状态写入备份数据存储。
func (adder *Adder) PinRoot() error {
	root, err := adder.RootNode()
	if err != nil {
		return err
	}
	if !adder.Pin {
		return nil
	}

	rnk := root.Cid()

	err = adder.dagService.Add(adder.ctx, root)
	if err != nil {
		return err
	}

	if adder.tempRoot.Defined() {
		err := adder.pinning.Unpin(adder.ctx, adder.tempRoot, true)
		if err != nil {
			return err
		}
		adder.tempRoot = rnk
	}

	adder.pinning.PinWithMode(rnk, pin.Recursive)
	return adder.pinning.Flush()
}

//Finalize刷新mfs根目录并返回mfs根节点。
func (adder *Adder) Finalize() (ipld.Node, error) {
	mr, err := adder.mfsRoot()
	if err != nil {
		return nil, err
	}
	var root mfs.FSNode
	rootdir := mr.GetDirectory()
	root = rootdir

	err = root.Flush()
	if err != nil {
		return nil, err
	}

	var name string
	if !adder.Wrap {
		children, err := rootdir.ListNames(adder.ctx)
		if err != nil {
			return nil, err
		}

		if len(children) == 0 {
			return nil, fmt.Errorf("expected at least one child dir, got none")
		}

//用第一个子项替换根
		name = children[0]
		root, err = rootdir.Child(name)
		if err != nil {
			return nil, err
		}
	}

	err = adder.outputDirs(name, root)
	if err != nil {
		return nil, err
	}

	err = mr.Close()
	if err != nil {
		return nil, err
	}

	return root.GetNode()
}

func (adder *Adder) outputDirs(path string, fsn mfs.FSNode) error {
	switch fsn := fsn.(type) {
	case *mfs.File:
		return nil
	case *mfs.Directory:
		names, err := fsn.ListNames(adder.ctx)
		if err != nil {
			return err
		}

		for _, name := range names {
			child, err := fsn.Child(name)
			if err != nil {
				return err
			}

			childpath := gopath.Join(path, name)
			err = adder.outputDirs(childpath, child)
			if err != nil {
				return err
			}

			fsn.Uncache(name)
		}
		nd, err := fsn.GetNode()
		if err != nil {
			return err
		}

		return outputDagnode(adder.Out, path, nd)
	default:
		return fmt.Errorf("unrecognized fsn type: %#v", fsn)
	}
}

//add从读卡器生成一个merkledag节点，将其添加到blockstore，
//并返回表示该节点的键。
//如果要固定它，请使用newadder（）和adder.pinroot（）。
func Add(n *core.IpfsNode, r io.Reader) (string, error) {
	return AddWithContext(n.Context(), n, r)
}

//addWithContext与add执行相同的操作，但使用自定义上下文。
func AddWithContext(ctx context.Context, n *core.IpfsNode, r io.Reader) (string, error) {
	defer n.Blockstore.PinLock().Unlock()

	fileAdder, err := NewAdder(ctx, n.Pinning, n.Blockstore, n.DAG)
	if err != nil {
		return "", err
	}

	node, err := fileAdder.add(r)
	if err != nil {
		return "", err
	}

	return node.Cid().String(), nil
}

//addr以递归方式在路径中添加文件。
func AddR(n *core.IpfsNode, root string) (key string, err error) {
	defer n.Blockstore.PinLock().Unlock()

	stat, err := os.Lstat(root)
	if err != nil {
		return "", err
	}

	f, err := files.NewSerialFile(root, false, stat)
	if err != nil {
		return "", err
	}
	defer f.Close()

	fileAdder, err := NewAdder(n.Context(), n.Pinning, n.Blockstore, n.DAG)
	if err != nil {
		return "", err
	}

	err = fileAdder.addFileNode(filepath.Base(root), f)
	if err != nil {
		return "", err
	}

	nd, err := fileAdder.Finalize()
	if err != nil {
		return "", err
	}

	return nd.String(), nil
}

//addwrapped从读卡器添加数据，并用目录对象对其进行包装
//保留文件名。
//返回添加文件（“<dir hash>/filename”）的路径，DAG节点为
//目录和错误（如果有）。
func AddWrapped(n *core.IpfsNode, r io.Reader, filename string) (string, ipld.Node, error) {
	fileAdder, err := NewAdder(n.Context(), n.Pinning, n.Blockstore, n.DAG)
	if err != nil {
		return "", nil, err
	}
	fileAdder.Wrap = true

	defer n.Blockstore.PinLock().Unlock()

	err = fileAdder.addFileNode(filename, files.NewReaderFile(r))
	if err != nil {
		return "", nil, err
	}

	dagnode, err := fileAdder.Finalize()
	if err != nil {
		return "", nil, err
	}

	c := dagnode.Cid()
	return gopath.Join(c.String(), filename), dagnode, nil
}

func (adder *Adder) addNode(node ipld.Node, path string) error {
//把它插到根上
	if path == "" {
		path = node.Cid().String()
	}

	if pi, ok := node.(*posinfo.FilestoreNode); ok {
		node = pi.Node
	}

	mr, err := adder.mfsRoot()
	if err != nil {
		return err
	}
	dir := gopath.Dir(path)
	if dir != "." {
		opts := mfs.MkdirOpts{
			Mkparents:  true,
			Flush:      false,
			CidBuilder: adder.CidBuilder,
		}
		if err := mfs.Mkdir(mr, dir, opts); err != nil {
			return err
		}
	}

	if err := mfs.PutNode(mr, path, node); err != nil {
		return err
	}

	if !adder.Silent {
		return outputDagnode(adder.Out, path, node)
	}
	return nil
}

//addallandpin添加给定请求的文件并对其进行pin。
func (adder *Adder) AddAllAndPin(file files.Node) (ipld.Node, error) {
	if adder.Pin {
		adder.unlocker = adder.gcLocker.PinLock()
	}
	defer func() {
		if adder.unlocker != nil {
			adder.unlocker.Unlock()
		}
	}()

	switch tf := file.(type) {
	case files.Directory:
//迭代每个顶级文件并分别添加。否则
//单个文件。文件f被视为目录，影响隐藏文件
//语义学。
		it := tf.Entries()
		for it.Next() {
			if err := adder.addFileNode(it.Name(), it.Node()); err != nil {
				return nil, err
			}
		}
		if it.Err() != nil {
			return nil, it.Err()
		}
		break
	default:
		if err := adder.addFileNode("", file); err != nil {
			return nil, err
		}
		break
	}

//将中间节点从编辑器复制到实际的DAGService
	nd, err := adder.Finalize()
	if err != nil {
		return nil, err
	}

	if !adder.Pin {
		return nd, nil
	}
	return nd, adder.PinRoot()
}

func (adder *Adder) addFileNode(path string, file files.Node) error {
	defer file.Close()
	err := adder.maybePauseForGC()
	if err != nil {
		return err
	}

	if adder.liveNodes >= liveCacheSize {
//TODO:一个更智能的缓存，它使用某种带有逐出处理程序的LRU缓存
		mr, err := adder.mfsRoot()
		if err != nil {
			return err
		}
		if err := mr.FlushMemFree(adder.ctx); err != nil {
			return err
		}

		adder.liveNodes = 0
	}
	adder.liveNodes++

	switch f := file.(type) {
	case files.Directory:
		return adder.addDir(path, f)
	case *files.Symlink:
		return adder.addSymlink(path, f)
	case files.File:
		return adder.addFile(path, f)
	default:
		return errors.New("unknown file type")
	}
}

func (adder *Adder) addSymlink(path string, l *files.Symlink) error {
	sdata, err := unixfs.SymlinkData(l.Target)
	if err != nil {
		return err
	}

	dagnode := dag.NodeWithData(sdata)
	dagnode.SetCidBuilder(adder.CidBuilder)
	err = adder.dagService.Add(adder.ctx, dagnode)
	if err != nil {
		return err
	}

	return adder.addNode(dagnode, path)
}

func (adder *Adder) addFile(path string, file files.File) error {
//如果指定了进度标志，请包装文件以便发送
//客户端进度更新（通过输出通道）
	var reader io.Reader = file
	if adder.Progress {
		rdr := &progressReader{file: reader, path: path, out: adder.Out}
		if fi, ok := file.(files.FileInfo); ok {
			reader = &progressReader2{rdr, fi}
		} else {
			reader = rdr
		}
	}

	dagnode, err := adder.add(reader)
	if err != nil {
		return err
	}

	addFileInfo, ok := file.(files.FileInfo)
	if ok {
		if addFileInfo.AbsPath() == os.Stdin.Name() && adder.Name != "" {
			path = adder.Name
			adder.Name = ""
		}
	}
//把它插到根上
	return adder.addNode(dagnode, path)
}

func (adder *Adder) addDir(path string, dir files.Directory) error {
	log.Infof("adding directory: %s", path)

	mr, err := adder.mfsRoot()
	if err != nil {
		return err
	}
	err = mfs.Mkdir(mr, path, mfs.MkdirOpts{
		Mkparents:  true,
		Flush:      false,
		CidBuilder: adder.CidBuilder,
	})
	if err != nil {
		return err
	}

	it := dir.Entries()
	for it.Next() {
		fpath := gopath.Join(path, it.Name())

//递归添加时跳过隐藏文件，除非启用了隐藏。
		if files.IsHidden(fpath, it.Node()) && !adder.Hidden {
			log.Infof("%s is hidden, skipping", fpath)
			continue
		}
		err = adder.addFileNode(fpath, it.Node())
		if err != nil {
			return err
		}
	}

	return it.Err()
}

func (adder *Adder) maybePauseForGC() error {
	if adder.unlocker != nil && adder.gcLocker.GCRequested() {
		err := adder.PinRoot()
		if err != nil {
			return err
		}

		adder.unlocker.Unlock()
		adder.unlocker = adder.gcLocker.PinLock()
	}
	return nil
}

//output dagnode通过输出通道发送dagnode信息
func outputDagnode(out chan<- interface{}, name string, dn ipld.Node) error {
	if out == nil {
		return nil
	}

	o, err := getOutput(dn)
	if err != nil {
		return err
	}

	out <- &coreiface.AddEvent{
		Path: o.Path,
		Name: name,
		Size: o.Size,
	}

	return nil
}

//从core/commands/object.go
func getOutput(dagnode ipld.Node) (*coreiface.AddEvent, error) {
	c := dagnode.Cid()
	s, err := dagnode.Size()
	if err != nil {
		return nil, err
	}

	output := &coreiface.AddEvent{
		Path: coreiface.IpfsPath(c),
		Size: strconv.FormatUint(s, 10),
	}

	return output, nil
}

type progressReader struct {
	file         io.Reader
	path         string
	out          chan<- interface{}
	bytes        int64
	lastProgress int64
}

func (i *progressReader) Read(p []byte) (int, error) {
	n, err := i.file.Read(p)

	i.bytes += int64(n)
	if i.bytes-i.lastProgress >= progressReaderIncrement || err == io.EOF {
		i.lastProgress = i.bytes
		i.out <- &coreiface.AddEvent{
			Name:  i.path,
			Bytes: i.bytes,
		}
	}

	return n, err
}

type progressReader2 struct {
	*progressReader
	files.FileInfo
}

func (i *progressReader2) Read(p []byte) (int, error) {
	return i.progressReader.Read(p)
}
