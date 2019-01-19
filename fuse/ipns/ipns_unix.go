
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+建设！诺福斯

//package fuse/ipns实现了一个fuse文件系统，该文件系统与
//有了IPN，IPF的命名系统。
package ipns

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	core "github.com/ipfs/go-ipfs/core"
	namesys "github.com/ipfs/go-ipfs/namesys"
	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	ft "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"

	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	mfs "gx/ipfs/QmP9eu5X5Ax8169jNWqAJcc42mdZgzLR1aKCEzqhNoBLKk/go-mfs"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	fuse "gx/ipfs/QmSJBsmLP1XMjv8hxYg2rUMdPDB7YUpyBo9idjrJ6Cmq6F/fuse"
	fs "gx/ipfs/QmSJBsmLP1XMjv8hxYg2rUMdPDB7YUpyBo9idjrJ6Cmq6F/fuse/fs"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

func init() {
	if os.Getenv("IPFS_FUSE_DEBUG") != "" {
		fuse.Debug = func(msg interface{}) {
			fmt.Println(msg)
		}
	}
}

var log = logging.Logger("fuse/ipns")

//filesystem是readwrite ipns fuse文件系统。
type FileSystem struct {
	Ipfs     *core.IpfsNode
	RootNode *Root
}

//newfilesystem使用给定的core.ipfsnode实例构造新的fs。
func NewFileSystem(ipfs *core.IpfsNode, sk ci.PrivKey, ipfspath, ipnspath string) (*FileSystem, error) {

	kmap := map[string]ci.PrivKey{
		"local": sk,
	}
	root, err := CreateRoot(ipfs, kmap, ipfspath, ipnspath)
	if err != nil {
		return nil, err
	}

	return &FileSystem{Ipfs: ipfs, RootNode: root}, nil
}

//根构造文件系统的根，根对象。
func (f *FileSystem) Root() (fs.Node, error) {
	log.Debug("filesystem, get root")
	return f.RootNode, nil
}

func (f *FileSystem) Destroy() {
	err := f.RootNode.Close()
	if err != nil {
		log.Errorf("Error Shutting Down Filesystem: %s\n", err)
	}
}

//根是文件系统树的根对象。
type Root struct {
	Ipfs *core.IpfsNode
	Keys map[string]ci.PrivKey

//用于与IPF的符号链接
	IpfsRoot  string
	IpnsRoot  string
	LocalDirs map[string]fs.Node
	Roots     map[string]*keyRoot

	LocalLinks map[string]*Link
}

func ipnsPubFunc(ipfs *core.IpfsNode, k ci.PrivKey) mfs.PubFunc {
	return func(ctx context.Context, c cid.Cid) error {
		return ipfs.Namesys.Publish(ctx, k, path.FromCid(c))
	}
}

func loadRoot(ctx context.Context, rt *keyRoot, ipfs *core.IpfsNode, name string) (fs.Node, error) {
	p, err := path.ParsePath("/ipns/" + name)
	if err != nil {
		log.Errorf("mkpath %s: %s", name, err)
		return nil, err
	}

	node, err := core.Resolve(ctx, ipfs.Namesys, ipfs.Resolver, p)
	switch err {
	case nil:
	case namesys.ErrResolveFailed:
		node = ft.EmptyDirNode()
	default:
		log.Errorf("looking up %s: %s", p, err)
		return nil, err
	}

	pbnode, ok := node.(*dag.ProtoNode)
	if !ok {
		return nil, dag.ErrNotProtobuf
	}

	root, err := mfs.NewRoot(ctx, ipfs.DAG, pbnode, ipnsPubFunc(ipfs, rt.k))
	if err != nil {
		return nil, err
	}

	rt.root = root

	return &Directory{dir: root.GetDirectory()}, nil
}

type keyRoot struct {
	k     ci.PrivKey
	alias string
	root  *mfs.Root
}

func CreateRoot(ipfs *core.IpfsNode, keys map[string]ci.PrivKey, ipfspath, ipnspath string) (*Root, error) {
	ldirs := make(map[string]fs.Node)
	roots := make(map[string]*keyRoot)
	links := make(map[string]*Link)
	for alias, k := range keys {
		pid, err := peer.IDFromPrivateKey(k)
		if err != nil {
			return nil, err
		}
		name := pid.Pretty()

		kr := &keyRoot{k: k, alias: alias}
		fsn, err := loadRoot(ipfs.Context(), kr, ipfs, name)
		if err != nil {
			return nil, err
		}

		roots[name] = kr
		ldirs[name] = fsn

//设置别名符号链接
		links[alias] = &Link{
			Target: name,
		}
	}

	return &Root{
		Ipfs:       ipfs,
		IpfsRoot:   ipfspath,
		IpnsRoot:   ipnspath,
		Keys:       keys,
		LocalDirs:  ldirs,
		LocalLinks: links,
		Roots:      roots,
	}, nil
}

//ATTR返回文件属性。
func (*Root) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debug("Root Attr")
a.Mode = os.ModeDir | 0111 //- RW+X
	return nil
}

//查找在此节点下执行查找。
func (s *Root) Lookup(ctx context.Context, name string) (fs.Node, error) {
	switch name {
	case "mach_kernel", ".hidden", "._.":
//只是在OSX上消除一些日志噪音。
		return nil, fuse.ENOENT
	}

	if lnk, ok := s.LocalLinks[name]; ok {
		return lnk, nil
	}

	nd, ok := s.LocalDirs[name]
	if ok {
		switch nd := nd.(type) {
		case *Directory:
			return nd, nil
		case *FileNode:
			return nd, nil
		default:
			return nil, fuse.EIO
		}
	}

//其他链接通过IPN解析并符号链接到IPFS安装点。
	ipnsName := "/ipns/" + name
	resolved, err := s.Ipfs.Namesys.Resolve(s.Ipfs.Context(), ipnsName)
	if err != nil {
		log.Warningf("ipns: namesys resolve error: %s", err)
		return nil, fuse.ENOENT
	}

	segments := resolved.Segments()
	if segments[0] == "ipfs" {
		p := path.Join(resolved.Segments()[1:])
		return &Link{s.IpfsRoot + "/" + p}, nil
	}

	log.Error("Invalid path.Path: ", resolved)
	return nil, errors.New("invalid path from ipns record")
}

func (r *Root) Close() error {
	for _, mr := range r.Roots {
		err := mr.root.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

//当卸载文件系统时调用forget。可能。
//参见评论：http://godoc.org/bazil.org/fuse/fsdestroyer
func (r *Root) Forget() {
	err := r.Close()
	if err != nil {
		log.Error(err)
	}
}

//readdirall读取特定目录。将显示本地可用的密钥
//以及到对等密钥的符号链接
func (r *Root) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	log.Debug("Root ReadDirAll")

	var listing []fuse.Dirent
	for alias, k := range r.Keys {
		pid, err := peer.IDFromPrivateKey(k)
		if err != nil {
			continue
		}
		ent := fuse.Dirent{
			Name: pid.Pretty(),
			Type: fuse.DT_Dir,
		}
		link := fuse.Dirent{
			Name: alias,
			Type: fuse.DT_Link,
		}
		listing = append(listing, ent, link)
	}
	return listing, nil
}

//目录是对mfs目录的包装，以满足fuse fs接口
type Directory struct {
	dir *mfs.Directory
}

type FileNode struct {
	fi *mfs.File
}

//文件是对mfs文件的包装，以满足fuse fs接口
type File struct {
	fi mfs.FileDescriptor
}

//ATTR返回给定节点的属性。
func (d *Directory) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debug("Directory Attr")
	a.Mode = os.ModeDir | 0555
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

//ATTR返回给定节点的属性。
func (fi *FileNode) Attr(ctx context.Context, a *fuse.Attr) error {
	log.Debug("File Attr")
	size, err := fi.fi.Size()
	if err != nil {
//在这种情况下，所讨论的DAG节点可能不是unixfs
		return fmt.Errorf("fuse/ipns: failed to get file.Size(): %s", err)
	}
	a.Mode = os.FileMode(0666)
	a.Size = uint64(size)
	a.Uid = uint32(os.Getuid())
	a.Gid = uint32(os.Getgid())
	return nil
}

//查找在此节点下执行查找。
func (s *Directory) Lookup(ctx context.Context, name string) (fs.Node, error) {
	child, err := s.dir.Child(name)
	if err != nil {
//TODO:使此错误更加通用。
		return nil, fuse.ENOENT
	}

	switch child := child.(type) {
	case *mfs.Directory:
		return &Directory{dir: child}, nil
	case *mfs.File:
		return &FileNode{fi: child}, nil
	default:
//注意：如果发生这种情况，我们不想继续，不可预测的行为。
//可能会发生。
		panic("invalid type found under directory. programmer error.")
	}
}

//readdirall将链接结构作为目录项读取
func (dir *Directory) ReadDirAll(ctx context.Context) ([]fuse.Dirent, error) {
	var entries []fuse.Dirent
	listing, err := dir.dir.List(ctx)
	if err != nil {
		return nil, err
	}
	for _, entry := range listing {
		dirent := fuse.Dirent{Name: entry.Name}

		switch mfs.NodeType(entry.Type) {
		case mfs.TDir:
			dirent.Type = fuse.DT_Dir
		case mfs.TFile:
			dirent.Type = fuse.DT_File
		}

		entries = append(entries, dirent)
	}

	if len(entries) > 0 {
		return entries, nil
	}
	return nil, fuse.ENOENT
}

func (fi *File) Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error {
	_, err := fi.fi.Seek(req.Offset, io.SeekStart)
	if err != nil {
		return err
	}

	fisize, err := fi.fi.Size()
	if err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	readsize := min(req.Size, int(fisize-req.Offset))
	n, err := fi.fi.CtxReadFull(ctx, resp.Data[:readsize])
	resp.Data = resp.Data[:n]
	return err
}

func (fi *File) Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error {
//TODO:在某种程度上，确保writeat在这里尊重上下文
	wrote, err := fi.fi.WriteAt(req.Data, req.Offset)
	if err != nil {
		return err
	}
	resp.Size = wrote
	return nil
}

func (fi *File) Flush(ctx context.Context, req *fuse.FlushRequest) error {
	errs := make(chan error, 1)
	go func() {
		errs <- fi.fi.Flush()
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (fi *File) Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error {
	if req.Valid.Size() {
		cursize, err := fi.fi.Size()
		if err != nil {
			return err
		}
		if cursize != int64(req.Size) {
			err := fi.fi.Truncate(int64(req.Size))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

//fsync将文件中的内容刷新到磁盘。
func (fi *FileNode) Fsync(ctx context.Context, req *fuse.FsyncRequest) error {
//这需要执行*完全*刷新，因为在mfs中，写入
//保留到根目录更新。
	errs := make(chan error, 1)
	go func() {
		errs <- fi.fi.Flush()
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (fi *File) Forget() {
//托多（steb）：这似乎是一个我们应该“打开”而不是“冲水”的地方。
	err := fi.fi.Flush()
	if err != nil {
		log.Debug("forget file error: ", err)
	}
}

func (dir *Directory) Mkdir(ctx context.Context, req *fuse.MkdirRequest) (fs.Node, error) {
	child, err := dir.dir.Mkdir(req.Name)
	if err != nil {
		return nil, err
	}

	return &Directory{dir: child}, nil
}

func (fi *FileNode) Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (fs.Handle, error) {
	fd, err := fi.fi.Open(mfs.Flags{
		Read:  req.Flags.IsReadOnly() || req.Flags.IsReadWrite(),
		Write: req.Flags.IsWriteOnly() || req.Flags.IsReadWrite(),
		Sync:  true,
	})
	if err != nil {
		return nil, err
	}

	if req.Flags&fuse.OpenTruncate != 0 {
		if req.Flags.IsReadOnly() {
			log.Error("tried to open a readonly file with truncate")
			return nil, fuse.ENOTSUP
		}
		log.Info("Need to truncate file!")
		err := fd.Truncate(0)
		if err != nil {
			return nil, err
		}
	} else if req.Flags&fuse.OpenAppend != 0 {
		log.Info("Need to append to file!")
		if req.Flags.IsReadOnly() {
			log.Error("tried to open a readonly file with append")
			return nil, fuse.ENOTSUP
		}

		_, err := fd.Seek(0, io.SeekEnd)
		if err != nil {
			log.Error("seek reset failed: ", err)
			return nil, err
		}
	}

	return &File{fi: fd}, nil
}

func (fi *File) Release(ctx context.Context, req *fuse.ReleaseRequest) error {
	return fi.fi.Close()
}

func (dir *Directory) Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (fs.Node, fs.Handle, error) {
//新建“空”文件
	nd := dag.NodeWithData(ft.FilePBData(nil, 0))
	err := dir.dir.AddChild(req.Name, nd)
	if err != nil {
		return nil, nil, err
	}

	child, err := dir.dir.Child(req.Name)
	if err != nil {
		return nil, nil, err
	}

	fi, ok := child.(*mfs.File)
	if !ok {
		return nil, nil, errors.New("child creation failed")
	}

	nodechild := &FileNode{fi: fi}

	fd, err := fi.Open(mfs.Flags{
		Read:  req.Flags.IsReadOnly() || req.Flags.IsReadWrite(),
		Write: req.Flags.IsWriteOnly() || req.Flags.IsReadWrite(),
		Sync:  true,
	})
	if err != nil {
		return nil, nil, err
	}

	return nodechild, &File{fi: fd}, nil
}

func (dir *Directory) Remove(ctx context.Context, req *fuse.RemoveRequest) error {
	err := dir.dir.Unlink(req.Name)
	if err != nil {
		return fuse.ENOENT
	}
	return nil
}

//rename实现noderenamer
func (dir *Directory) Rename(ctx context.Context, req *fuse.RenameRequest, newDir fs.Node) error {
	cur, err := dir.dir.Child(req.OldName)
	if err != nil {
		return err
	}

	err = dir.dir.Unlink(req.OldName)
	if err != nil {
		return err
	}

	switch newDir := newDir.(type) {
	case *Directory:
		nd, err := cur.GetNode()
		if err != nil {
			return err
		}

		err = newDir.dir.AddChild(req.NewName, nd)
		if err != nil {
			return err
		}
	case *FileNode:
		log.Error("Cannot move node into a file!")
		return fuse.EPERM
	default:
		log.Error("Unknown node type for rename target dir!")
		return errors.New("unknown fs node type")
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

//检查out节点是否实现了我们想要的所有接口
type ipnsRoot interface {
	fs.Node
	fs.HandleReadDirAller
	fs.NodeStringLookuper
}

var _ ipnsRoot = (*Root)(nil)

type ipnsDirectory interface {
	fs.HandleReadDirAller
	fs.Node
	fs.NodeCreater
	fs.NodeMkdirer
	fs.NodeRemover
	fs.NodeRenamer
	fs.NodeStringLookuper
}

var _ ipnsDirectory = (*Directory)(nil)

type ipnsFile interface {
	fs.HandleFlusher
	fs.HandleReader
	fs.HandleWriter
	fs.HandleReleaser
}

type ipnsFileNode interface {
	fs.Node
	fs.NodeFsyncer
	fs.NodeOpener
}

var _ ipnsFileNode = (*FileNode)(nil)
var _ ipnsFile = (*File)(nil)
