
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//package pin实现跟踪的结构和方法
//用户希望将哪些对象保存在本地。
package pin

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/ipfs/go-ipfs/dagutils"
	mdag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
)

var log = logging.Logger("pin")

var pinDatastoreKey = ds.NewKey("/local/pins")

var emptyKey cid.Cid

func init() {
	e, err := cid.Decode("QmdfTbBqBPQ7VNxZEYEj14VmRuZBkqFbiwReogJgS1zR1n")
	if err != nil {
		log.Error("failed to decode empty key constant")
		os.Exit(1)
	}
	emptyKey = e
}

const (
	linkRecursive = "recursive"
	linkDirect    = "direct"
	linkIndirect  = "indirect"
	linkInternal  = "internal"
	linkNotPinned = "not pinned"
	linkAny       = "any"
	linkAll       = "all"
)

//模式允许指定不同类型的管脚（递归、直接等）。
//有关完整列表，请参阅pin模式常量。
type Mode int

//引脚模式
const (
//递归管脚将目标CID与任何可到达的子级固定在一起。
	Recursive Mode = iota

//直接插脚只是目标CID。
	Direct

//间接管脚是具有递归固定祖先的CID。
	Indirect

//内部管脚是用于保持管脚内部状态的CID。
	Internal

//未被钉住的
	NotPinned

//any指任何固定的cid
	Any
)

//modetostring返回模式的可读名称。
func ModeToString(mode Mode) (string, bool) {
	m := map[Mode]string{
		Recursive: linkRecursive,
		Direct:    linkDirect,
		Indirect:  linkIndirect,
		Internal:  linkInternal,
		NotPinned: linkNotPinned,
		Any:       linkAny,
	}
	s, ok := m[mode]
	return s, ok
}

//StringToMode将modetToString（）的结果解析回模式。
//它返回一个布尔值，如果模式未知，则设置为false。
func StringToMode(s string) (Mode, bool) {
	m := map[string]Mode{
		linkRecursive: Recursive,
		linkDirect:    Direct,
		linkIndirect:  Indirect,
		linkInternal:  Internal,
		linkNotPinned: NotPinned,
		linkAny:       Any,
linkAll:       Any, //“全部”和“任何”的意思相同
	}
	mode, ok := m[s]
	return mode, ok
}

//Pinner提供了跟踪以下节点的必要方法：
//本地保存，根据pin模式。在实践中，一个别针在
//负责保存本地仓库的物品清单
//不要被垃圾收集。
type Pinner interface {
//is pinned返回给定的cid是否被固定
//解释了它为什么会被钉住
	IsPinned(cid.Cid) (string, bool, error)

//ispinnedwithType返回给定的cid是否与
//给定的pin类型，以及返回其固定的pin类型。
	IsPinnedWithType(cid.Cid, Mode) (string, bool, error)

//固定给定节点（可选递归）。
	Pin(ctx context.Context, node ipld.Node, recursive bool) error

//解锁给定的CID。如果recursive为true，则移除递归或
//直接销如果recursive为false，则只删除直接pin。
	Unpin(ctx context.Context, cid cid.Cid, recursive bool) error

//更新将递归pin从一个cid更新到另一个cid
//这比简单地固定新的和取消固定
//旧的
	Update(ctx context.Context, from, to cid.Cid, unpin bool) error

//检查一组键是否固定，是否比
//为每个键调用ispinned
	CheckIfPinned(cids ...cid.Cid) ([]Pinned, error)

//pinwithMode用于手动编辑pin结构。使用与
//小心！如果使用不当，垃圾收集可能不会
//成功。
	PinWithMode(cid.Cid, Mode)

//removePinWithMode用于手动编辑销结构。
//小心使用！如果使用不当，垃圾收集可能不会
//成功。
	RemovePinWithMode(cid.Cid, Mode)

//flush将pin状态写入备份数据存储
	Flush() error

//DirectKeys返回所有直接固定的CID
	DirectKeys() []cid.Cid

//DirectKeys返回所有递归固定的CID
	RecursiveKeys() []cid.Cid

//InternalPins返回为
//佩内尔
	InternalPins() []cid.Cid
}

//pinned表示用pinning策略固定的cid。
//VIA字段允许在
//如果该项不是直接固定的（而是递归固定的）
//以某种优势）。
type Pinned struct {
	Key  cid.Cid
	Mode Mode
	Via  cid.Cid
}

//pinned返回给定的cid是否被pinned
func (p Pinned) Pinned() bool {
	return p.Mode != NotPinned
}

//字符串以字符串形式返回pin状态
func (p Pinned) String() string {
	switch p.Mode {
	case NotPinned:
		return "not pinned"
	case Indirect:
		return fmt.Sprintf("pinned via %s", p.Via)
	default:
		modeStr, _ := ModeToString(p.Mode)
		return fmt.Sprintf("pinned: %s", modeStr)
	}
}

//Pinner实现Pinner接口
type pinner struct {
	lock       sync.RWMutex
	recursePin *cid.Set
	directPin  *cid.Set

//跟踪用于存储固定状态的键，所以gc跟踪
//不要删除它们。
	internalPin *cid.Set
	dserv       ipld.DAGService
internal    ipld.DAGService //用于存储内部对象的DAGService
	dstore      ds.Datastore
}

//NewPinner使用给定的数据存储作为后端创建一个新的Pinner
func NewPinner(dstore ds.Datastore, serv, internal ipld.DAGService) Pinner {

	rcset := cid.NewSet()
	dirset := cid.NewSet()

	return &pinner{
		recursePin:  rcset,
		directPin:   dirset,
		dserv:       serv,
		dstore:      dstore,
		internal:    internal,
		internalPin: cid.NewSet(),
	}
}

//固定给定节点（可选递归）
func (p *pinner) Pin(ctx context.Context, node ipld.Node, recurse bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	err := p.dserv.Add(ctx, node)
	if err != nil {
		return err
	}

	c := node.Cid()

	if recurse {
		if p.recursePin.Has(c) {
			return nil
		}

		if p.directPin.Has(c) {
			p.directPin.Remove(c)
		}
		p.lock.Unlock()
//获取整个图形
		err := mdag.FetchGraph(ctx, c, p.dserv)
		p.lock.Lock()
		if err != nil {
			return err
		}

		if p.recursePin.Has(c) {
			return nil
		}

		if p.directPin.Has(c) {
			p.directPin.Remove(c)
		}

		p.recursePin.Add(c)
	} else {
		p.lock.Unlock()
		_, err := p.dserv.Get(ctx, c)
		p.lock.Lock()
		if err != nil {
			return err
		}

		if p.recursePin.Has(c) {
			return fmt.Errorf("%s already pinned recursively", c.String())
		}

		p.directPin.Add(c)
	}
	return nil
}

//尝试取消固定未固定的项目时返回errnotpined。
var ErrNotPinned = fmt.Errorf("not pinned")

//解锁给定的密钥
func (p *pinner) Unpin(ctx context.Context, c cid.Cid, recursive bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	reason, pinned, err := p.isPinnedWithType(c, Any)
	if err != nil {
		return err
	}
	if !pinned {
		return ErrNotPinned
	}
	switch reason {
	case "recursive":
		if recursive {
			p.recursePin.Remove(c)
			return nil
		}
		return fmt.Errorf("%s is pinned recursively", c)
	case "direct":
		p.directPin.Remove(c)
		return nil
	default:
		return fmt.Errorf("%s is pinned indirectly under %s", c, reason)
	}
}

func (p *pinner) isInternalPin(c cid.Cid) bool {
	return p.internalPin.Has(c)
}

//is pinned返回给定密钥是否被固定
//解释了它为什么会被钉住
func (p *pinner) IsPinned(c cid.Cid) (string, bool, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.isPinnedWithType(c, Any)
}

//ispinnedwithType返回给定的cid是否与
//给定的pin类型，以及返回其固定的pin类型。
func (p *pinner) IsPinnedWithType(c cid.Cid, mode Mode) (string, bool, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.isPinnedWithType(c, mode)
}

//ispinnedwithType是不锁定的ispinnedwithType的实现。
//用于已获取锁的其他固定方法
func (p *pinner) isPinnedWithType(c cid.Cid, mode Mode) (string, bool, error) {
	switch mode {
	case Any, Direct, Indirect, Recursive, Internal:
	default:
		err := fmt.Errorf("invalid Pin Mode '%d', must be one of {%d, %d, %d, %d, %d}",
			mode, Direct, Indirect, Recursive, Internal, Any)
		return "", false, err
	}
	if (mode == Recursive || mode == Any) && p.recursePin.Has(c) {
		return linkRecursive, true, nil
	}
	if mode == Recursive {
		return "", false, nil
	}

	if (mode == Direct || mode == Any) && p.directPin.Has(c) {
		return linkDirect, true, nil
	}
	if mode == Direct {
		return "", false, nil
	}

	if (mode == Internal || mode == Any) && p.isInternalPin(c) {
		return linkInternal, true, nil
	}
	if mode == Internal {
		return "", false, nil
	}

//违约是间接的
	visitedSet := cid.NewSet()
	for _, rc := range p.recursePin.Keys() {
		has, err := hasChild(p.dserv, rc, c, visitedSet.Visit)
		if err != nil {
			return "", false, err
		}
		if has {
			return rc.String(), true, nil
		}
	}
	return "", false, nil
}

//checkifpinned检查一组键是否被固定，其效率比
//为每个键调用ispinned，返回cid的固定状态
func (p *pinner) CheckIfPinned(cids ...cid.Cid) ([]Pinned, error) {
	p.lock.RLock()
	defer p.lock.RUnlock()
	pinned := make([]Pinned, 0, len(cids))
	toCheck := cid.NewSet()

//直接检查非间接销
	for _, c := range cids {
		if p.recursePin.Has(c) {
			pinned = append(pinned, Pinned{Key: c, Mode: Recursive})
		} else if p.directPin.Has(c) {
			pinned = append(pinned, Pinned{Key: c, Mode: Direct})
		} else if p.isInternalPin(c) {
			pinned = append(pinned, Pinned{Key: c, Mode: Internal})
		} else {
			toCheck.Add(c)
		}
	}

//现在遍历所有递归管脚以检查间接管脚
	var checkChildren func(cid.Cid, cid.Cid) error
	checkChildren = func(rk, parentKey cid.Cid) error {
		links, err := ipld.GetLinks(context.TODO(), p.dserv, parentKey)
		if err != nil {
			return err
		}
		for _, lnk := range links {
			c := lnk.Cid

			if toCheck.Has(c) {
				pinned = append(pinned,
					Pinned{Key: c, Mode: Indirect, Via: rk})
				toCheck.Remove(c)
			}

			err := checkChildren(rk, c)
			if err != nil {
				return err
			}

			if toCheck.Len() == 0 {
				return nil
			}
		}
		return nil
	}

	for _, rk := range p.recursePin.Keys() {
		err := checkChildren(rk, rk)
		if err != nil {
			return nil, err
		}
		if toCheck.Len() == 0 {
			break
		}
	}

//任何留在tocheck中的内容都不会被固定
	for _, k := range toCheck.Keys() {
		pinned = append(pinned, Pinned{Key: k, Mode: NotPinned})
	}

	return pinned, nil
}

//removePinWithMode用于手动编辑销结构。
//小心使用！如果使用不当，垃圾收集可能不会
//成功。
func (p *pinner) RemovePinWithMode(c cid.Cid, mode Mode) {
	p.lock.Lock()
	defer p.lock.Unlock()
	switch mode {
	case Direct:
		p.directPin.Remove(c)
	case Recursive:
		p.recursePin.Remove(c)
	default:
//程序错误，恐慌正常
		panic("unrecognized pin type")
	}
}

func cidSetWithValues(cids []cid.Cid) *cid.Set {
	out := cid.NewSet()
	for _, c := range cids {
		out.Add(c)
	}
	return out
}

//LoadPinner从给定的数据存储区加载Pinner及其键集
func LoadPinner(d ds.Datastore, dserv, internal ipld.DAGService) (Pinner, error) {
	p := new(pinner)

	rootKey, err := d.Get(pinDatastoreKey)
	if err != nil {
		return nil, fmt.Errorf("cannot load pin state: %v", err)
	}
	rootCid, err := cid.Cast(rootKey)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Second*5)
	defer cancel()

	root, err := internal.Get(ctx, rootCid)
	if err != nil {
		return nil, fmt.Errorf("cannot find pinning root object: %v", err)
	}

	rootpb, ok := root.(*mdag.ProtoNode)
	if !ok {
		return nil, mdag.ErrNotProtobuf
	}

	internalset := cid.NewSet()
	internalset.Add(rootCid)
	recordInternal := internalset.Add

{ //加载递归集
		recurseKeys, err := loadSet(ctx, internal, rootpb, linkRecursive, recordInternal)
		if err != nil {
			return nil, fmt.Errorf("cannot load recursive pins: %v", err)
		}
		p.recursePin = cidSetWithValues(recurseKeys)
	}

{ //荷载直接集
		directKeys, err := loadSet(ctx, internal, rootpb, linkDirect, recordInternal)
		if err != nil {
			return nil, fmt.Errorf("cannot load direct pins: %v", err)
		}
		p.directPin = cidSetWithValues(directKeys)
	}

	p.internalPin = internalset

//分配服务
	p.dserv = dserv
	p.dstore = d
	p.internal = internal

	return p, nil
}

//DirectKeys返回一个包含直接固定键的切片
func (p *pinner) DirectKeys() []cid.Cid {
	return p.directPin.Keys()
}

//递归键返回一个包含递归固定键的切片
func (p *pinner) RecursiveKeys() []cid.Cid {
	return p.recursePin.Keys()
}

//更新将递归pin从一个cid更新到另一个cid
//这比简单地固定新的和取消固定
//旧的
func (p *pinner) Update(ctx context.Context, from, to cid.Cid, unpin bool) error {
	p.lock.Lock()
	defer p.lock.Unlock()

	if !p.recursePin.Has(from) {
		return fmt.Errorf("'from' cid was not recursively pinned already")
	}

	err := dagutils.DiffEnumerate(ctx, p.dserv, from, to)
	if err != nil {
		return err
	}

	p.recursePin.Add(to)
	if unpin {
		p.recursePin.Remove(from)
	}
	return nil
}

//flush编码并将pinner密钥集写入数据存储
func (p *pinner) Flush() error {
	p.lock.Lock()
	defer p.lock.Unlock()

	ctx := context.TODO()

	internalset := cid.NewSet()
	recordInternal := internalset.Add

	root := &mdag.ProtoNode{}
	{
		n, err := storeSet(ctx, p.internal, p.directPin.Keys(), recordInternal)
		if err != nil {
			return err
		}
		if err := root.AddNodeLink(linkDirect, n); err != nil {
			return err
		}
	}

	{
		n, err := storeSet(ctx, p.internal, p.recursePin.Keys(), recordInternal)
		if err != nil {
			return err
		}
		if err := root.AddNodeLink(linkRecursive, n); err != nil {
			return err
		}
	}

//添加空节点，该节点由管脚集引用，但从未创建
	err := p.internal.Add(ctx, new(mdag.ProtoNode))
	if err != nil {
		return err
	}

	err = p.internal.Add(ctx, root)
	if err != nil {
		return err
	}

	k := root.Cid()

	internalset.Add(k)
	if err := p.dstore.Put(pinDatastoreKey, k.Bytes()); err != nil {
		return fmt.Errorf("cannot store pin state: %v", err)
	}
	p.internalPin = internalset
	return nil
}

//InternalPins返回为
//佩内尔
func (p *pinner) InternalPins() []cid.Cid {
	p.lock.Lock()
	defer p.lock.Unlock()
	var out []cid.Cid
	out = append(out, p.internalPin.Keys()...)
	return out
}

//pinwithMode允许用户对pin进行细粒度控制
//计数
func (p *pinner) PinWithMode(c cid.Cid, mode Mode) {
	p.lock.Lock()
	defer p.lock.Unlock()
	switch mode {
	case Recursive:
		p.recursePin.Add(c)
	case Direct:
		p.directPin.Add(c)
	}
}

//hasschild递归地在根cid的子代中查找cid。
//访问功能可用于快捷方式已访问的分支。
func hasChild(ng ipld.NodeGetter, root cid.Cid, child cid.Cid, visit func(cid.Cid) bool) (bool, error) {
	links, err := ipld.GetLinks(context.TODO(), ng, root)
	if err != nil {
		return false, err
	}
	for _, lnk := range links {
		c := lnk.Cid
		if lnk.Cid.Equals(child) {
			return true, nil
		}
		if visit(c) {
			has, err := hasChild(ng, c, child, visit)
			if err != nil {
				return false, err
			}

			if has {
				return has, nil
			}
		}
	}
	return false, nil
}
