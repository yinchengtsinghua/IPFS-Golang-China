
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package iface

import (
	"context"
	"io"

	options "github.com/ipfs/go-ipfs/core/coreapi/interface/options"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
)

//ObjectStat提供有关DAG节点的信息
type ObjectStat struct {
//cid是节点的cid
	Cid cid.Cid

//numlinks是节点包含的链接数
	NumLinks int

//BlockSize是原始序列化节点的大小
	BlockSize int

//linksize是链接块节的大小
	LinksSize int

//data size是数据块节的大小
	DataSize int

//累积大小是树的大小（块大小+链接大小）
	CumulativeSize int
}

//ChangeType表示ObjectChange中的更改类型
type ChangeType int

const (
//将链接添加到图表时设置diffAdd
	DiffAdd ChangeType = iota

//从图表中删除链接时设置diffremove
	DiffRemove

//在图表中更改链接时设置diffmod
	DiffMod
)

//ObjectChange表示图形中的更改
type ObjectChange struct {
//更改类型，可以是：
//*diffadd-添加了一个链接
//*diffremove-删除了一个链接
//*diffmod-修改了链接
	Type ChangeType

//更改链接的路径
	Path string

//在更改前保持链接路径。注意，当一个链接
//加上，这将是零。
	Before ResolvedPath

//更改后保持链接路径。注意，当一个链接
//删除后，将为零。
	After ResolvedPath
}

//objectapi指定merkledag的接口并包含有用的实用程序
//用于操作Merkledag数据结构。
type ObjectAPI interface {
//新建创建新的空（默认情况下）DAG节点。
	New(context.Context, ...options.ObjectNewOption) (ipld.Node, error)

//将导入的数据放入Merkledag
	Put(context.Context, io.Reader, ...options.ObjectPutOption) (ResolvedPath, error)

//get返回路径的节点
	Get(context.Context, Path) (ipld.Node, error)

//数据返回节点数据的读卡器
	Data(context.Context, Path) (io.Reader, error)

//链接返回lint或节点包含的链接
	Links(context.Context, Path) ([]*ipld.Link, error)

//stat返回有关节点的信息
	Stat(context.Context, Path) (*ObjectStat, error)

//addlink在指定路径下添加链接。子路径可以指向
//必须存在的专利中的子目录（可以重写
//使用WITHCREATE选项）。
	AddLink(ctx context.Context, base Path, name string, child Path, opts ...options.ObjectAddLinkOption) (ResolvedPath, error)

//rmlink从节点中删除链接
	RmLink(ctx context.Context, base Path, link string) (ResolvedPath, error)

//AppendData将数据追加到节点
	AppendData(context.Context, Path, io.Reader) (ResolvedPath, error)

//setdata设置节点中包含的数据
	SetData(context.Context, Path, io.Reader) (ResolvedPath, error)

//diff返回将第一个对象转换为
//第二。
	Diff(context.Context, Path, Path) ([]ObjectChange, error)
}
