
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package filestore

import (
	"fmt"
	"sort"

	pb "github.com/ipfs/go-ipfs/filestore/pb"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	blockstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	dshelp "gx/ipfs/QmauEMWPoSqggfpSDHMMXuDn12DTd7TaFBvn39eeurzKT2/go-ipfs-ds-help"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
	dsq "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/query"
)

//状态用于标识引用的块数据的状态
//由文件存储。在其他地方，它被corruptreferenceerror使用。
type Status int32

//这些是支持的状态代码。
const (
	StatusOk           Status = 0
StatusFileError    Status = 10 //备份文件错误
StatusFileNotFound Status = 11 //找不到备份文件
StatusFileChanged  Status = 12 //文件内容已更改
StatusOtherError   Status = 20 //内部错误，可能损坏条目
	StatusKeyNotFound  Status = 30
)

//字符串为状态代码提供了人类可读的表示形式。
func (s Status) String() string {
	switch s {
	case StatusOk:
		return "ok"
	case StatusFileError:
		return "error"
	case StatusFileNotFound:
		return "no-file"
	case StatusFileChanged:
		return "changed"
	case StatusOtherError:
		return "ERROR"
	case StatusKeyNotFound:
		return "missing"
	default:
		return "???"
	}
}

//FORMAT返回格式化为字符串的状态
//以0开头。
func (s Status) Format() string {
	return fmt.Sprintf("%-7s", s.String())
}

//listres包装list*（）函数的响应，其中
//允许获取和验证由文件管理器存储的块
//文件存储。它包括有关参考文献的信息
//块。
type ListRes struct {
	Status   Status
	ErrorMsg string
	Key      cid.Cid
	FilePath string
	Offset   uint64
	Size     uint64
}

//Formatlong返回ListRes对象的可读字符串。
func (r *ListRes) FormatLong() string {
	switch {
	case !r.Key.Defined():
		return "<corrupt key>"
	case r.FilePath == "":
		return r.Key.String()
	default:
		return fmt.Sprintf("%-50s %6d %s %d", r.Key, r.Size, r.FilePath, r.Offset)
	}
}

//列表从文件管理器中提取具有给定键的块
//并返回带有信息的listres对象。
//列表不验证引用是否有效，也不验证
//原始数据是可访问的。参见VIFIFY（）。
func List(fs *Filestore, key cid.Cid) *ListRes {
	return list(fs, false, key)
}

//listall以迭代器的形式返回函数，一旦调用该迭代器，它将返回
//在文件存储的文件管理器中，每个块一个接一个。
//listall不验证引用是否有效，或者
//原始数据是可访问的。请参见verifyall（）。
func ListAll(fs *Filestore, fileOrder bool) (func() *ListRes, error) {
	if fileOrder {
		return listAllFileOrder(fs, false)
	}
	return listAll(fs, false)
}

//验证从文件管理器中获取具有给定密钥的块
//并返回带有信息的listres对象。
//验证以确保引用有效，并且块数据可以
//读。
func Verify(fs *Filestore, key cid.Cid) *ListRes {
	return list(fs, true, key)
}

//verifyall返回一个作为迭代器的函数，一旦调用，
//在文件存储的文件管理器中，逐个返回每个块。
//verifyall检查引用是否有效以及块数据
//可以阅读。
func VerifyAll(fs *Filestore, fileOrder bool) (func() *ListRes, error) {
	if fileOrder {
		return listAllFileOrder(fs, true)
	}
	return listAll(fs, true)
}

func list(fs *Filestore, verify bool, key cid.Cid) *ListRes {
	dobj, err := fs.fm.getDataObj(key)
	if err != nil {
		return mkListRes(key, nil, err)
	}
	if verify {
		_, err = fs.fm.readDataObj(key, dobj)
	}
	return mkListRes(key, dobj, err)
}

func listAll(fs *Filestore, verify bool) (func() *ListRes, error) {
	q := dsq.Query{}
	qr, err := fs.fm.ds.Query(q)
	if err != nil {
		return nil, err
	}

	return func() *ListRes {
		cid, dobj, err := next(qr)
		if dobj == nil && err == nil {
			return nil
		} else if err == nil && verify {
			_, err = fs.fm.readDataObj(cid, dobj)
		}
		return mkListRes(cid, dobj, err)
	}, nil
}

func next(qr dsq.Results) (cid.Cid, *pb.DataObj, error) {
	v, ok := qr.NextSync()
	if !ok {
		return cid.Cid{}, nil, nil
	}

	k := ds.RawKey(v.Key)
	c, err := dshelp.DsKeyToCid(k)
	if err != nil {
		return cid.Cid{}, nil, fmt.Errorf("decoding cid from filestore: %s", err)
	}

	dobj, err := unmarshalDataObj(v.Value)
	if err != nil {
		return c, nil, err
	}

	return c, dobj, nil
}

func listAllFileOrder(fs *Filestore, verify bool) (func() *ListRes, error) {
	q := dsq.Query{}
	qr, err := fs.fm.ds.Query(q)
	if err != nil {
		return nil, err
	}

	var entries listEntries

	for {
		v, ok := qr.NextSync()
		if !ok {
			break
		}
		dobj, err := unmarshalDataObj(v.Value)
		if err != nil {
			entries = append(entries, &listEntry{
				dsKey: v.Key,
				err:   err,
			})
		} else {
			entries = append(entries, &listEntry{
				dsKey:    v.Key,
				filePath: dobj.GetFilePath(),
				offset:   dobj.GetOffset(),
				size:     dobj.GetSize_(),
			})
		}
	}
	sort.Sort(entries)

	i := 0
	return func() *ListRes {
		if i >= len(entries) {
			return nil
		}
		v := entries[i]
		i++
//尝试将数据存储键转换为CID，
//存储错误，但不要使用它
		cid, keyErr := dshelp.DsKeyToCid(ds.RawKey(v.dsKey))
//首先，如果listres已经出错，则返回该错误
		if v.err != nil {
			return mkListRes(cid, nil, v.err)
		}
//现在重建数据对象
		dobj := pb.DataObj{
			FilePath: v.filePath,
			Offset:   v.offset,
			Size_:    v.size,
		}
//如果我们不能转换数据存储键，返回
//错误
		if keyErr != nil {
			return mkListRes(cid, &dobj, keyErr)
		}
//如有要求，最后验证dataobj
		var err error
		if verify {
			_, err = fs.fm.readDataObj(cid, &dobj)
		}
		return mkListRes(cid, &dobj, err)
	}, nil
}

type listEntry struct {
	filePath string
	offset   uint64
	dsKey    string
	size     uint64
	err      error
}

type listEntries []*listEntry

func (l listEntries) Len() int      { return len(l) }
func (l listEntries) Swap(i, j int) { l[i], l[j] = l[j], l[i] }
func (l listEntries) Less(i, j int) bool {
	if l[i].filePath == l[j].filePath {
		if l[i].offset == l[j].offset {
			return l[i].dsKey < l[j].dsKey
		}
		return l[i].offset < l[j].offset
	}
	return l[i].filePath < l[j].filePath
}

func mkListRes(c cid.Cid, d *pb.DataObj, err error) *ListRes {
	status := StatusOk
	errorMsg := ""
	if err != nil {
		if err == ds.ErrNotFound || err == blockstore.ErrNotFound {
			status = StatusKeyNotFound
		} else if err, ok := err.(*CorruptReferenceError); ok {
			status = err.Code
		} else {
			status = StatusOtherError
		}
		errorMsg = err.Error()
	}
	if d == nil {
		return &ListRes{
			Status:   status,
			ErrorMsg: errorMsg,
			Key:      c,
		}
	}

	return &ListRes{
		Status:   status,
		ErrorMsg: errorMsg,
		Key:      c,
		FilePath: d.FilePath,
		Size:     d.Size_,
		Offset:   d.Offset,
	}
}
