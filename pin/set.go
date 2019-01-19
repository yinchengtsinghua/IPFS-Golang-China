
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package pin

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/fnv"
	"sort"

	"github.com/ipfs/go-ipfs/pin/internal/pb"
	"gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	"gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"
)

const (
//default fan out指定每层的默认扇出链接数
	defaultFanout = 256

//maxitems是单个存储桶中最多可容纳的项目数。
	maxItems = 8192
)

func hash(seed uint32, c cid.Cid) uint32 {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], seed)
	h := fnv.New32a()
	_, _ = h.Write(buf[:])
	_, _ = h.Write(c.Bytes())
	return h.Sum32()
}

type itemIterator func() (c cid.Cid, ok bool)

type keyObserver func(cid.Cid)

type sortByHash struct {
	links []*ipld.Link
}

func (s sortByHash) Len() int {
	return len(s.links)
}

func (s sortByHash) Less(a, b int) bool {
	return bytes.Compare(s.links[a].Cid.Bytes(), s.links[b].Cid.Bytes()) == -1
}

func (s sortByHash) Swap(a, b int) {
	s.links[a], s.links[b] = s.links[b], s.links[a]
}

func storeItems(ctx context.Context, dag ipld.DAGService, estimatedLen uint64, depth uint32, iter itemIterator, internalKeys keyObserver) (*merkledag.ProtoNode, error) {
	links := make([]*ipld.Link, 0, defaultFanout+maxItems)
	for i := 0; i < defaultFanout; i++ {
		links = append(links, &ipld.Link{Cid: emptyKey})
	}

//将EmptyKey添加到内部pinset对象集
	n := &merkledag.ProtoNode{}
	n.SetLinks(links)

	internalKeys(emptyKey)

	hdr := &pb.Set{
		Version: 1,
		Fanout:  defaultFanout,
		Seed:    depth,
	}
	if err := writeHdr(n, hdr); err != nil {
		return nil, err
	}

	if estimatedLen < maxItems {
//它很可能适合
		links := n.Links()
		for i := 0; i < maxItems; i++ {
			k, ok := iter()
			if !ok {
//都做了
				break
			}

			links = append(links, &ipld.Link{Cid: k})
		}

		n.SetLinks(links)

//按哈希排序，同时交换项数据
		s := sortByHash{
			links: n.Links()[defaultFanout:],
		}
		sort.Stable(s)
	}

	hashed := make([][]cid.Cid, defaultFanout)
	for {
//此循环基本上枚举集合中的每个项
//并把它们映射到一组桶中。每个存储桶将以递归方式
//变成了自己的子集，等等下链。每个子集
//添加到DAGService，并将其放置在一个集合节点中
//链接数组。
//
//以前，bucket是通过从
//输入键+种子。这是错误的，因为我们以后会指定
//创建的子集合按
//Int32哈希值，256。这导致覆盖现有子集
//以及失去销子。修复方法（从这个评论往下几行）是
//在创建
//桶。这样，我们以后就可以避免任何重叠。
		k, ok := iter()
		if !ok {
			break
		}
		h := hash(depth, k) % defaultFanout
		hashed[h] = append(hashed[h], k)
	}

	for h, items := range hashed {
		if len(items) == 0 {
//递归基本情况
			continue
		}

		childIter := getCidListIterator(items)

//从该bucket索引的项中递归创建pinset
		child, err := storeItems(ctx, dag, uint64(len(items)), depth+1, childIter, internalKeys)
		if err != nil {
			return nil, err
		}

		size, err := child.Size()
		if err != nil {
			return nil, err
		}

		err = dag.Add(ctx, child)
		if err != nil {
			return nil, err
		}
		childKey := child.Cid()

		internalKeys(childKey)

//覆盖现有链接数组中的“空键”
		n.Links()[h] = &ipld.Link{
			Cid:  childKey,
			Size: size,
		}
	}
	return n, nil
}

func readHdr(n *merkledag.ProtoNode) (*pb.Set, error) {
	hdrLenRaw, consumed := binary.Uvarint(n.Data())
	if consumed <= 0 {
		return nil, errors.New("invalid Set header length")
	}

	pbdata := n.Data()[consumed:]
	if hdrLenRaw > uint64(len(pbdata)) {
		return nil, errors.New("impossibly large Set header length")
	}
//由于hdrlenraw<=int，我们现在知道它适合int
	hdrLen := int(hdrLenRaw)
	var hdr pb.Set
	if err := proto.Unmarshal(pbdata[:hdrLen], &hdr); err != nil {
		return nil, err
	}

	if v := hdr.GetVersion(); v != 1 {
		return nil, fmt.Errorf("unsupported Set version: %d", v)
	}
	if uint64(hdr.GetFanout()) > uint64(len(n.Links())) {
		return nil, errors.New("impossibly large Fanout")
	}
	return &hdr, nil
}

func writeHdr(n *merkledag.ProtoNode, hdr *pb.Set) error {
	hdrData, err := proto.Marshal(hdr)
	if err != nil {
		return err
	}

//为长度前缀和封送的头数据留出足够的空间
	data := make([]byte, binary.MaxVarintLen64, binary.MaxVarintLen64+len(hdrData))

//写入头数据的uvarint长度
	uvarlen := binary.PutUvarint(data, uint64(len(hdrData)))

//将实际protobuf数据*附加到*我们编写的长度值之后
	data = append(data[:uvarlen], hdrData...)

	n.SetData(data)
	return nil
}

type walkerFunc func(idx int, link *ipld.Link) error

func walkItems(ctx context.Context, dag ipld.DAGService, n *merkledag.ProtoNode, fn walkerFunc, children keyObserver) error {
	hdr, err := readHdr(n)
	if err != nil {
		return err
	}
//readhdr保证fanout是一个安全值
	fanout := hdr.GetFanout()
	for i, l := range n.Links()[fanout:] {
		if err := fn(i, l); err != nil {
			return err
		}
	}
	for _, l := range n.Links()[:fanout] {
		c := l.Cid
		children(c)
		if c.Equals(emptyKey) {
			continue
		}
		subtree, err := l.GetNode(ctx, dag)
		if err != nil {
			return err
		}

		stpb, ok := subtree.(*merkledag.ProtoNode)
		if !ok {
			return merkledag.ErrNotProtobuf
		}

		if err := walkItems(ctx, dag, stpb, fn, children); err != nil {
			return err
		}
	}
	return nil
}

func loadSet(ctx context.Context, dag ipld.DAGService, root *merkledag.ProtoNode, name string, internalKeys keyObserver) ([]cid.Cid, error) {
	l, err := root.GetNodeLink(name)
	if err != nil {
		return nil, err
	}

	lnkc := l.Cid
	internalKeys(lnkc)

	n, err := l.GetNode(ctx, dag)
	if err != nil {
		return nil, err
	}

	pbn, ok := n.(*merkledag.ProtoNode)
	if !ok {
		return nil, merkledag.ErrNotProtobuf
	}

	var res []cid.Cid
	walk := func(idx int, link *ipld.Link) error {
		res = append(res, link.Cid)
		return nil
	}

	if err := walkItems(ctx, dag, pbn, walk, internalKeys); err != nil {
		return nil, err
	}
	return res, nil
}

func getCidListIterator(cids []cid.Cid) itemIterator {
	return func() (c cid.Cid, ok bool) {
		if len(cids) == 0 {
			return cid.Cid{}, false
		}

		first := cids[0]
		cids = cids[1:]
		return first, true
	}
}

func storeSet(ctx context.Context, dag ipld.DAGService, cids []cid.Cid, internalKeys keyObserver) (*merkledag.ProtoNode, error) {
	iter := getCidListIterator(cids)

	n, err := storeItems(ctx, dag, uint64(len(cids)), 0, iter, internalKeys)
	if err != nil {
		return nil, err
	}
	err = dag.Add(ctx, n)
	if err != nil {
		return nil, err
	}
	internalKeys(n.Cid())
	return n, nil
}
