
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package iface

import (
	ipfspath "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	"gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
)

//TODO:与ipfspath合并，这样我们就不依赖它了

//路径是API中使用的路径的通用包装器。可以解析路径
//使用API中的一个resolve函数传递给一个cid。
//
//路径必须以有效前缀作为前缀：
//
//*/ipfs-不可变的unixfs路径（文件）
//*/ipld-不可变的ipld路径（数据）
//*/ipns-可变名称。通常解析为不可变路径之一
//
type Path interface {
//字符串以字符串形式返回路径。
	String() string

//命名空间返回路径的第一个组件。
//
//例如，path“/ipfs/qmhash”，调用namespace（）将返回“ipfs”
	Namespace() string

//如果保证此路径指向的数据
//不要改变。
//
//请注意，解析的可变路径可以是不可变的。
	Mutable() bool
}

//resolved path是解析为最后一个可解析节点的路径
type ResolvedPath interface {
//cid返回路径引用的节点的cid。剩余的
//路径保证在节点内。
//
//实例：
//如果有3个链接对象：qmroot->a->b：
//
//CIDB：=“foo”：“bar”：42
//CIDA：=“B”：“/”：CIDB
//cidroot：=“a”：“/”：cida
//
//并解析路径：
//
//*“/ipfs/$cidroot”
//*调用cid（）将返回'cidroot'
//*调用root（）将返回'cidroot'
//*调用remains（）将返回``
//
//*“/ipfs/$cidroot/a”
//*调用cid（）将返回“cida”
//*调用root（）将返回'cidroot'
//*调用remains（）将返回``
//
//*“/ipfs/$cidroot/a/b/foo”
//*调用cid（）将返回'cidb'
//*调用root（）将返回'cidroot'
//*调用remains（）将返回'foo'
//
//*“/ipfs/$cidroot/a/b/foo/bar”
//*调用cid（）将返回'cidb'
//*调用root（）将返回'cidroot'
//*调用remains（）将返回'foo/bar'
	Cid() cid.Cid

//根返回路径根对象的cid
//
//例子：
//如果有3个链接对象：qmroot->a->b，并解析路径
//“/ipfs/qmroot/a/b”，根方法将返回对象qmroot的cid
//
//有关更多示例，请参见cid（）方法的文档
	Root() cid.Cid

//余数返回路径的未解析部分
//
//例子：
//如果有两个链接对象：qmroot->a，其中a是cbor节点
//包含以下数据：
//
//“foo”：“bar”：42_
//
//解析/ipld/qmroot/a/foo/bar时，余数返回“foo/bar”
//
//有关更多示例，请参见cid（）方法的文档
	Remainder() string

	Path
}

//路径实现coreiface.path
type path struct {
	path ipfspath.Path
}

//resolvedpath实现coreiface.resolvedpath
type resolvedPath struct {
	path
	cid       cid.Cid
	root      cid.Cid
	remainder string
}

//将提供的段附加到基路径
func Join(base Path, a ...string) Path {
	s := ipfspath.Join(append([]string{base.String()}, a...))
	return &path{path: ipfspath.FromString(s)}
}

//ipfs path从提供的cid创建新的/ipfs路径
func IpfsPath(c cid.Cid) ResolvedPath {
	return &resolvedPath{
		path:      path{ipfspath.Path("/ipfs/" + c.String())},
		cid:       c,
		root:      c,
		remainder: "",
	}
}

//ipld path从提供的cid创建新的/ipld路径
func IpldPath(c cid.Cid) ResolvedPath {
	return &resolvedPath{
		path:      path{ipfspath.Path("/ipld/" + c.String())},
		cid:       c,
		root:      c,
		remainder: "",
	}
}

//parsePath将字符串路径解析为路径
func ParsePath(p string) (Path, error) {
	pp, err := ipfspath.ParsePath(p)
	if err != nil {
		return nil, err
	}

	return &path{path: pp}, nil
}

//NewResolvedPath创建新的ResolvedPath。此函数不执行任何检查
//并被解析器实现使用。输入错误可能
//引起恐慌。小心轻放。
func NewResolvedPath(ipath ipfspath.Path, c cid.Cid, root cid.Cid, remainder string) ResolvedPath {
	return &resolvedPath{
		path:      path{ipath},
		cid:       c,
		root:      root,
		remainder: remainder,
	}
}

func (p *path) String() string {
	return p.path.String()
}

func (p *path) Namespace() string {
	if len(p.path.Segments()) < 1 {
panic("path without namespace") //这在任何情况下都不应该发生
	}
	return p.path.Segments()[0]
}

func (p *path) Mutable() bool {
//TODO:mfs:检查/local
	return p.Namespace() == "ipns"
}

func (p *resolvedPath) Cid() cid.Cid {
	return p.cid
}

func (p *resolvedPath) Root() cid.Cid {
	return p.root
}

func (p *resolvedPath) Remainder() string {
	return p.remainder
}
