
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package filestore

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	pb "github.com/ipfs/go-ipfs/filestore/pb"

	posinfo "gx/ipfs/QmR6YMs8EkXQLXNwQKxLnQp2VBZSepoEJ8KCZAyanJHhJu/go-ipfs-posinfo"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	blockstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	blocks "gx/ipfs/QmWoXtvgC8inqFkAATB7cp2Dax7XBi9VDvSg9RCCZufmRk/go-block-format"
	dshelp "gx/ipfs/QmauEMWPoSqggfpSDHMMXuDn12DTd7TaFBvn39eeurzKT2/go-ipfs-ds-help"
	proto "gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
	dsns "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/namespace"
	dsq "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/query"
)

//filestoreprefix标识filemanager块的键前缀。
var FilestorePrefix = ds.NewKey("filestore")

//文件管理器是一个块存储实现，它存储特殊的
//阻止文件存储代码类型。这些节点只包含引用
//到文件系统中块数据的实际位置
//（路径和偏移）。
type FileManager struct {
	AllowFiles bool
	AllowUrls  bool
	ds         ds.Batching
	root       string
}

//corruptreferenceerror实现错误接口。
//用于指示块内容指向
//无法通过引用块检索（即
//找不到文件，或者数据在读取时发生了更改）。
type CorruptReferenceError struct {
	Code Status
	Err  error
}

//error（）返回corruptreferenceerror中的错误消息
//作为字符串。
func (c CorruptReferenceError) Error() string {
	return c.Err.Error()
}

//NewFileManager使用给定的
//数据存储和根目录。所有filestorenodes路径都是相对于
//这里给出的根路径，它是为任何操作准备的。
func NewFileManager(ds ds.Batching, root string) *FileManager {
	return &FileManager{ds: dsns.Wrap(ds, FilestorePrefix), root: root}
}

//allkeyschan返回一个用于读取存储在其中的密钥的通道
//文件管理器。如果取消给定的上下文，则通道将
//关闭。
func (f *FileManager) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	q := dsq.Query{KeysOnly: true}

	res, err := f.ds.Query(q)
	if err != nil {
		return nil, err
	}

	out := make(chan cid.Cid, dsq.KeysOnlyBufSize)
	go func() {
		defer close(out)
		for {
			v, ok := res.NextSync()
			if !ok {
				return
			}

			k := ds.RawKey(v.Key)
			c, err := dshelp.DsKeyToCid(k)
			if err != nil {
				log.Errorf("decoding cid from filestore: %s", err)
				continue
			}

			select {
			case out <- c:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

//删除块从底层删除引用块
//数据存储。它不接触参考数据。
func (f *FileManager) DeleteBlock(c cid.Cid) error {
	err := f.ds.Delete(dshelp.CidToDsKey(c))
	if err == ds.ErrNotFound {
		return blockstore.ErrNotFound
	}
	return err
}

//get从数据存储中读取块。读块
//分两步完成：第一步检索引用
//从数据存储区阻止。第二步使用存储的
//直接从磁盘读取原始块数据的路径和偏移量。
func (f *FileManager) Get(c cid.Cid) (blocks.Block, error) {
	dobj, err := f.getDataObj(c)
	if err != nil {
		return nil, err
	}
	out, err := f.readDataObj(c, dobj)
	if err != nil {
		return nil, err
	}

	return blocks.NewBlockWithCid(out, c)
}

//GetSize从数据存储中获取块的大小。
//
//即使返回块，此方法也可能成功返回大小
//将失败，因为关联的文件不再可用。
func (f *FileManager) GetSize(c cid.Cid) (int, error) {
	dobj, err := f.getDataObj(c)
	if err != nil {
		return -1, err
	}
	return int(dobj.GetSize_()), nil
}

func (f *FileManager) readDataObj(c cid.Cid, d *pb.DataObj) ([]byte, error) {
	if IsURL(d.GetFilePath()) {
		return f.readURLDataObj(c, d)
	}
	return f.readFileDataObj(c, d)
}

func (f *FileManager) getDataObj(c cid.Cid) (*pb.DataObj, error) {
	o, err := f.ds.Get(dshelp.CidToDsKey(c))
	switch err {
	case ds.ErrNotFound:
		return nil, blockstore.ErrNotFound
	default:
		return nil, err
	case nil:
//
	}

	return unmarshalDataObj(o)
}

func unmarshalDataObj(data []byte) (*pb.DataObj, error) {
	var dobj pb.DataObj
	if err := proto.Unmarshal(data, &dobj); err != nil {
		return nil, err
	}

	return &dobj, nil
}

func (f *FileManager) readFileDataObj(c cid.Cid, d *pb.DataObj) ([]byte, error) {
	if !f.AllowFiles {
		return nil, ErrFilestoreNotEnabled
	}

	p := filepath.FromSlash(d.GetFilePath())
	abspath := filepath.Join(f.root, p)

	fi, err := os.Open(abspath)
	if os.IsNotExist(err) {
		return nil, &CorruptReferenceError{StatusFileNotFound, err}
	} else if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}
	defer fi.Close()

	_, err = fi.Seek(int64(d.GetOffset()), io.SeekStart)
	if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}

	outbuf := make([]byte, d.GetSize_())
	_, err = io.ReadFull(fi, outbuf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil, &CorruptReferenceError{StatusFileChanged, err}
	} else if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}

	outcid, err := c.Prefix().Sum(outbuf)
	if err != nil {
		return nil, err
	}

	if !c.Equals(outcid) {
		return nil, &CorruptReferenceError{StatusFileChanged,
			fmt.Errorf("data in file did not match. %s offset %d", d.GetFilePath(), d.GetOffset())}
	}

	return outbuf, nil
}

//从URL读取并验证块
func (f *FileManager) readURLDataObj(c cid.Cid, d *pb.DataObj) ([]byte, error) {
	if !f.AllowUrls {
		return nil, ErrUrlstoreNotEnabled
	}

	req, err := http.NewRequest("GET", d.GetFilePath(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", d.GetOffset(), d.GetOffset()+d.GetSize_()-1))

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusPartialContent {
		return nil, &CorruptReferenceError{StatusFileError,
			fmt.Errorf("expected HTTP 200 or 206 got %d", res.StatusCode)}
	}

	outbuf := make([]byte, d.GetSize_())
	_, err = io.ReadFull(res.Body, outbuf)
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		return nil, &CorruptReferenceError{StatusFileChanged, err}
	} else if err != nil {
		return nil, &CorruptReferenceError{StatusFileError, err}
	}
	res.Body.Close()

	outcid, err := c.Prefix().Sum(outbuf)
	if err != nil {
		return nil, err
	}

	if !c.Equals(outcid) {
		return nil, &CorruptReferenceError{StatusFileChanged,
			fmt.Errorf("data in file did not match. %s offset %d", d.GetFilePath(), d.GetOffset())}
	}

	return outbuf, nil
}

//如果文件管理器正在存储块引用，则返回。它不
//验证数据，也不检查引用是否有效。
func (f *FileManager) Has(c cid.Cid) (bool, error) {
//注意：有意思的事情要考虑。没有验证数据。
//
	dsk := dshelp.CidToDsKey(c)
	return f.ds.Has(dsk)
}

type putter interface {
	Put(ds.Key, []byte) error
}

//PUT向文件管理器添加新的引用块。它不检查
//证明该引用有效。
func (f *FileManager) Put(b *posinfo.FilestoreNode) error {
	return f.putTo(b, f.ds)
}

func (f *FileManager) putTo(b *posinfo.FilestoreNode, to putter) error {
	var dobj pb.DataObj

	if IsURL(b.PosInfo.FullPath) {
		if !f.AllowUrls {
			return ErrUrlstoreNotEnabled
		}
		dobj.FilePath = b.PosInfo.FullPath
	} else {
		if !f.AllowFiles {
			return ErrFilestoreNotEnabled
		}
		if !filepath.HasPrefix(b.PosInfo.FullPath, f.root) {
			return fmt.Errorf("cannot add filestore references outside ipfs root (%s)", f.root)
		}

		p, err := filepath.Rel(f.root, b.PosInfo.FullPath)
		if err != nil {
			return err
		}

		dobj.FilePath = filepath.ToSlash(p)
	}
	dobj.Offset = b.PosInfo.Offset
	dobj.Size_ = uint64(len(b.RawData()))

	data, err := proto.Marshal(&dobj)
	if err != nil {
		return err
	}

	return to.Put(dshelp.CidToDsKey(b.Cid()), data)
}

//putmany与put（）类似，但它使用块切片，
//允许它创建批处理事务。
func (f *FileManager) PutMany(bs []*posinfo.FilestoreNode) error {
	batch, err := f.ds.Batch()
	if err != nil {
		return err
	}

	for _, b := range bs {
		if err := f.putTo(b, batch); err != nil {
			return err
		}
	}

	return batch.Commit()
}

//如果字符串表示有效的URL
//骨灰可以处理。更具体地说，如果字符串
//以“http://”或“https://”开头。
func IsURL(str string) bool {
	return (len(str) > 7 && str[0] == 'h' && str[1] == 't' && str[2] == 't' && str[3] == 'p') &&
		((len(str) > 8 && str[4] == 's' && str[5] == ':' && str[6] == '/' && str[7] == '/') ||
			(str[4] == ':' && str[5] == '/' && str[6] == '/'))
}
