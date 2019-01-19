
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package tarfmt

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"strings"

	"github.com/ipfs/go-ipfs/dagutils"
	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	importer "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/importer"
	uio "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs/io"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"

	chunker "gx/ipfs/QmR4QQVkBZsZENRjYFVi8dEtPL3daZRNKk24m4r6WKJHNm/go-ipfs-chunker"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("tarfmt")

var blockSize = 512
var zeroBlock = make([]byte, blockSize)

func marshalHeader(h *tar.Header) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := tar.NewWriter(buf)
	err := w.WriteHeader(h)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

//importtar将tar文件导入给定的dagservice并返回根目录
//节点。
func ImportTar(ctx context.Context, r io.Reader, ds ipld.DAGService) (*dag.ProtoNode, error) {
	tr := tar.NewReader(r)

	root := new(dag.ProtoNode)
	root.SetData([]byte("ipfs/tar"))

	e := dagutils.NewDagEditor(root, ds)

	for {
		h, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		header := new(dag.ProtoNode)

		headerBytes, err := marshalHeader(h)
		if err != nil {
			return nil, err
		}

		header.SetData(headerBytes)

		if h.Size > 0 {
			spl := chunker.NewRabin(tr, uint64(chunker.DefaultBlockSize))
			nd, err := importer.BuildDagFromReader(ds, spl)
			if err != nil {
				return nil, err
			}

			err = header.AddNodeLink("data", nd)
			if err != nil {
				return nil, err
			}
		}

		err = ds.Add(ctx, header)
		if err != nil {
			return nil, err
		}

		path := escapePath(h.Name)
		err = e.InsertNodeAtPath(context.Background(), path, header, func() *dag.ProtoNode { return new(dag.ProtoNode) })
		if err != nil {
			return nil, err
		}
	}

	return e.Finalize(ctx, ds)
}

//在每个path元素的开头添加一个“-”，以便我们可以将“data”用作
//结构中的特殊链接，无需担心
func escapePath(pth string) string {
	elems := path.SplitList(strings.Trim(pth, "/"))
	for i, e := range elems {
		elems[i] = "-" + e
	}
	return path.Join(elems)
}

type tarReader struct {
	links []*ipld.Link
	ds    ipld.DAGService

	childRead *tarReader
	hdrBuf    *bytes.Reader
	fileRead  *countReader
	pad       int

	ctx context.Context
}

func (tr *tarReader) Read(b []byte) (int, error) {
//如果我们有一个标题要读，它会优先
	if tr.hdrBuf != nil {
		n, err := tr.hdrBuf.Read(b)
		if err == io.EOF {
			tr.hdrBuf = nil
			return n, nil
		}
		return n, err
	}

//没有剩余的头，请检查递归
	if tr.childRead != nil {
		n, err := tr.childRead.Read(b)
		if err == io.EOF {
			tr.childRead = nil
			return n, nil
		}
		return n, err
	}

//检查要读取的文件数据
	if tr.fileRead != nil {
		n, err := tr.fileRead.Read(b)
		if err == io.EOF {
			nr := tr.fileRead.n
			tr.pad = (blockSize - (nr % blockSize)) % blockSize
			tr.fileRead.Close()
			tr.fileRead = nil
			return n, nil
		}
		return n, err
	}

//必须将读取的文件数据填充到512字节的偏移量
	if tr.pad > 0 {
		n := copy(b, zeroBlock[:tr.pad])
		tr.pad -= n
		return n, nil
	}

	if len(tr.links) == 0 {
		return 0, io.EOF
	}

	next := tr.links[0]
	tr.links = tr.links[1:]

	headerNd, err := next.GetNode(tr.ctx, tr.ds)
	if err != nil {
		return 0, err
	}

	hndpb, ok := headerNd.(*dag.ProtoNode)
	if !ok {
		return 0, dag.ErrNotProtobuf
	}

	tr.hdrBuf = bytes.NewReader(hndpb.Data())

	dataNd, err := hndpb.GetLinkedProtoNode(tr.ctx, tr.ds, "data")
	if err != nil && err != dag.ErrLinkNotFound {
		return 0, err
	}

	if err == nil {
		dr, err := uio.NewDagReader(tr.ctx, dataNd, tr.ds)
		if err != nil {
			log.Error("dagreader error: ", err)
			return 0, err
		}

		tr.fileRead = &countReader{r: dr}
	} else if len(headerNd.Links()) > 0 {
		tr.childRead = &tarReader{
			links: headerNd.Links(),
			ds:    tr.ds,
			ctx:   tr.ctx,
		}
	}

	return tr.Read(b)
}

//exporttar将传递的DAG导出为tar文件。这个函数是反函数
//进口的
func ExportTar(ctx context.Context, root *dag.ProtoNode, ds ipld.DAGService) (io.Reader, error) {
	if string(root.Data()) != "ipfs/tar" {
		return nil, errors.New("not an IPFS tarchive")
	}
	return &tarReader{
		links: root.Links(),
		ds:    ds,
		ctx:   ctx,
	}, nil
}

type countReader struct {
	r io.ReadCloser
	n int
}

func (r *countReader) Read(b []byte) (int, error) {
	n, err := r.r.Read(b)
	r.n += n
	return n, err
}

func (r *countReader) Close() error {
	return r.r.Close()
}
