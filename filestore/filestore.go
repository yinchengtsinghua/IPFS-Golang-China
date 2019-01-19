
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//package filestore实现一个blockstore，它能够读取特定的
//直接从文件系统中的原始位置获取的数据块。
//
//在文件存储中，对象叶存储为文件存储节点。丝状螺旋体
//包括文件系统路径和偏移量，允许块存储处理
//这样的块可以避免存储整个内容并从
//文件系统位置。
package filestore

import (
	"context"
	"errors"

	posinfo "gx/ipfs/QmR6YMs8EkXQLXNwQKxLnQp2VBZSepoEJ8KCZAyanJHhJu/go-ipfs-posinfo"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	blockstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	blocks "gx/ipfs/QmWoXtvgC8inqFkAATB7cp2Dax7XBi9VDvSg9RCCZufmRk/go-block-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	dsq "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/query"
)

var log = logging.Logger("filestore")

var ErrFilestoreNotEnabled = errors.New("filestore is not enabled, see https://GII.IO/VNITF“
var ErrUrlstoreNotEnabled = errors.New("urlstore is not enabled")

//文件存储通过组合标准块存储来实现块存储
//存储常规块和一个名为
//文件管理器存储外部文件中存在的数据块。
type Filestore struct {
	fm *FileManager
	bs blockstore.Blockstore
}

//文件管理器返回文件存储中的文件管理器。
func (f *Filestore) FileManager() *FileManager {
	return f.fm
}

//MainBlockStore返回文件存储中的标准块存储。
func (f *Filestore) MainBlockstore() blockstore.Blockstore {
	return f.bs
}

//newfilestore使用给定的blockstore和filemanager创建一个。
func NewFilestore(bs blockstore.Blockstore, fm *FileManager) *Filestore {
	return &Filestore{fm, bs}
}

//allkeyschan返回一个用于读取存储在其中的密钥的通道
//街区商店。如果取消给定的上下文，通道将关闭。
func (f *Filestore) AllKeysChan(ctx context.Context) (<-chan cid.Cid, error) {
	ctx, cancel := context.WithCancel(ctx)

	a, err := f.bs.AllKeysChan(ctx)
	if err != nil {
		cancel()
		return nil, err
	}

	out := make(chan cid.Cid, dsq.KeysOnlyBufSize)
	go func() {
		defer cancel()
		defer close(out)

		var done bool
		for !done {
			select {
			case c, ok := <-a:
				if !ok {
					done = true
					continue
				}
				select {
				case out <- c:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}

//不能同时做这些，因为周围的抽象
//LEVELDB使我们为这两个操作查询LEVELDB。我们显然
//无法同时查询级别数据库
		b, err := f.fm.AllKeysChan(ctx)
		if err != nil {
			log.Error("error querying filestore: ", err)
			return
		}

		done = false
		for !done {
			select {
			case c, ok := <-b:
				if !ok {
					done = true
					continue
				}
				select {
				case out <- c:
				case <-ctx.Done():
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return out, nil
}

//删除块从
//连锁店。如预期，在文件管理器块的情况下，只有
//引用被删除，而不是其内容。它可能会回来
//未存储块时出错。
func (f *Filestore) DeleteBlock(c cid.Cid) error {
	err1 := f.bs.DeleteBlock(c)
	if err1 != nil && err1 != blockstore.ErrNotFound {
		return err1
	}

	err2 := f.fm.DeleteBlock(c)
//如果我们成功地从blockstore中删除了一些内容，但是
//文件存储没有，返回成功

	switch err2 {
	case nil:
		return nil
	case blockstore.ErrNotFound:
		if err1 == blockstore.ErrNotFound {
			return blockstore.ErrNotFound
		}
		return nil
	default:
		return err2
	}
}

//get使用给定的cid检索块。它可能会回来
//未存储块时出错。
func (f *Filestore) Get(c cid.Cid) (blocks.Block, error) {
	blk, err := f.bs.Get(c)
	switch err {
	case nil:
		return blk, nil
	case blockstore.ErrNotFound:
		return f.fm.Get(c)
	default:
		return nil, err
	}
}

//GetSize返回请求块的大小。它可能返回errnotfound
//当块未存储时。
func (f *Filestore) GetSize(c cid.Cid) (int, error) {
	size, err := f.bs.GetSize(c)
	switch err {
	case nil:
		return size, nil
	case blockstore.ErrNotFound:
		return f.fm.GetSize(c)
	default:
		return -1, err
	}
}

//如果具有给定cid的块为
//存储在文件存储中。
func (f *Filestore) Has(c cid.Cid) (bool, error) {
	has, err := f.bs.Has(c)
	if err != nil {
		return false, err
	}

	if has {
		return true, nil
	}

	return f.fm.Has(c)
}

//在文件存储中放置存储块。对于街区
//基础类型filestorenode，操作是
//委托给文件管理器，而其他块
//由常规BlockStore处理。
func (f *Filestore) Put(b blocks.Block) error {
	has, err := f.Has(b.Cid())
	if err != nil {
		return err
	}

	if has {
		return nil
	}

	switch b := b.(type) {
	case *posinfo.FilestoreNode:
		return f.fm.Put(b)
	default:
		return f.bs.Put(b)
	}
}

//putmany与put（）类似，但它接受一个块切片，允许
//执行批处理事务的基础块存储区。
func (f *Filestore) PutMany(bs []blocks.Block) error {
	var normals []blocks.Block
	var fstores []*posinfo.FilestoreNode

	for _, b := range bs {
		has, err := f.Has(b.Cid())
		if err != nil {
			return err
		}

		if has {
			continue
		}

		switch b := b.(type) {
		case *posinfo.FilestoreNode:
			fstores = append(fstores, b)
		default:
			normals = append(normals, b)
		}
	}

	if len(normals) > 0 {
		err := f.bs.PutMany(normals)
		if err != nil {
			return err
		}
	}

	if len(fstores) > 0 {
		err := f.fm.PutMany(fstores)
		if err != nil {
			return err
		}
	}
	return nil
}

//hashonread调用blockstore.hashonread。
func (f *Filestore) HashOnRead(enabled bool) {
	f.bs.HashOnRead(enabled)
}

var _ blockstore.Blockstore = (*Filestore)(nil)
