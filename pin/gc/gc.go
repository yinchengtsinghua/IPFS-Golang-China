
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//包GC为Go IPF提供垃圾收集。
package gc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	pin "github.com/ipfs/go-ipfs/pin"
	dag "gx/ipfs/QmTQdH4848iTVCJmKXYyRiK72HufWTLYQQ8iN3JaQ8K1Hq/go-merkledag"
	bserv "gx/ipfs/QmYPZzd9VqmJDwxUnThfeSbV1Y5o53aVPDijTB7j7rS9Ep/go-blockservice"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	bstore "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	"gx/ipfs/QmYMQuypUbgsdNHmuCBSUJV6wdQVsBHRivNAp3efHJwZJD/go-verifcid"
	offline "gx/ipfs/QmYZwey1thDTynSrvd6qQkX24UpTka6TFhQ2v569UpoqxD/go-ipfs-exchange-offline"
	ipld "gx/ipfs/QmcKKBwfz6FyQdHR2jsXrrF6XeSBXYL86anmWNewpFpoF5/go-ipld-format"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	dstore "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
)

var log = logging.Logger("gc")

//结果表示垃圾收集的增量输出
//跑。它包含一个错误，或者一个被删除对象的cid。
type Result struct {
	KeyRemoved cid.Cid
	Error      error
}

//GC对块存储区中的块执行标记和清理垃圾收集
//首先，它创建一个“已标记”集，并向其中添加以下内容：
//-所有递归固定的块，加上它们的所有子代（递归）
//-BestEffortRoots及其所有子代（递归）
//-所有直接固定块
//-Pinner内部使用的所有块
//
//然后，该例程迭代块存储区中的每个块，并
//删除标记集中找不到的任何块。
func GC(ctx context.Context, bs bstore.GCBlockstore, dstor dstore.Datastore, pn pin.Pinner, bestEffortRoots []cid.Cid) <-chan Result {

	elock := log.EventBegin(ctx, "GC.lockWait")
	unlocker := bs.GCLock()
	elock.Done()
	elock = log.EventBegin(ctx, "GC.locked")
	emark := log.EventBegin(ctx, "GC.mark")

	bsrv := bserv.New(bs, offline.Exchange(bs))
	ds := dag.NewDAGService(bsrv)

	output := make(chan Result, 128)

	go func() {
		defer close(output)
		defer unlocker.Unlock()
		defer elock.Done()

		gcs, err := ColoredSet(ctx, pn, ds, bestEffortRoots, output)
		if err != nil {
			select {
			case output <- Result{Error: err}:
			case <-ctx.Done():
			}
			return
		}
		emark.Append(logging.LoggableMap{
			"blackSetSize": fmt.Sprintf("%d", gcs.Len()),
		})
		emark.Done()
		esweep := log.EventBegin(ctx, "GC.sweep")

		keychan, err := bs.AllKeysChan(ctx)
		if err != nil {
			select {
			case output <- Result{Error: err}:
			case <-ctx.Done():
			}
			return
		}

		errors := false
		var removed uint64

	loop:
		for {
			select {
			case k, ok := <-keychan:
				if !ok {
					break loop
				}
				if !gcs.Has(k) {
					err := bs.DeleteBlock(k)
					removed++
					if err != nil {
						errors = true
						output <- Result{Error: &CannotDeleteBlockError{k, err}}
//log.errorf（“从块存储中删除密钥时出错：%s”，err）
//继续，因为错误不是致命的
						continue loop
					}
					select {
					case output <- Result{KeyRemoved: k}:
					case <-ctx.Done():
						break loop
					}
				}
			case <-ctx.Done():
				break loop
			}
		}
		esweep.Append(logging.LoggableMap{
			"whiteSetSize": fmt.Sprintf("%d", removed),
		})
		esweep.Done()
		if errors {
			select {
			case output <- Result{Error: ErrCannotDeleteSomeBlocks}:
			case <-ctx.Done():
				return
			}
		}

		defer log.EventBegin(ctx, "GC.datastore").Done()
		gds, ok := dstor.(dstore.GCDatastore)
		if !ok {
			return
		}

		err = gds.CollectGarbage()
		if err != nil {
			select {
			case output <- Result{Error: err}:
			case <-ctx.Done():
			}
			return
		}
	}()

	return output
}

//子代递归查找给定根的所有子代，并
//使用提供的dag.getlinks函数将它们添加到给定的cid.set中
//走在树上。
func Descendants(ctx context.Context, getLinks dag.GetLinks, set *cid.Set, roots []cid.Cid) error {
	verifyGetLinks := func(ctx context.Context, c cid.Cid) ([]*ipld.Link, error) {
		err := verifcid.ValidateCid(c)
		if err != nil {
			return nil, err
		}

		return getLinks(ctx, c)
	}

	verboseCidError := func(err error) error {
		if strings.Contains(err.Error(), verifcid.ErrBelowMinimumHashLength.Error()) ||
			strings.Contains(err.Error(), verifcid.ErrPossiblyInsecureHashFunction.Error()) {
			err = fmt.Errorf("\"%s\"\nPlease run 'ipfs pin verify'"+
				" to list insecure hashes. If you want to read them,"+
				" please downgrade your go-ipfs to 0.4.13\n", err)
			log.Error(err)
		}
		return err
	}

	for _, c := range roots {
		set.Add(c)

//EnumerateChildren递归地遍历DAG并向给定集添加键
		err := dag.EnumerateChildren(ctx, verifyGetLinks, c, set.Visit)

		if err != nil {
			err = verboseCidError(err)
			return err
		}
	}

	return nil
}

//coloredset计算图表中由
//在给定的销中的销。
func ColoredSet(ctx context.Context, pn pin.Pinner, ng ipld.NodeGetter, bestEffortRoots []cid.Cid, output chan<- Result) (*cid.Set, error) {
//目前在内存中实现的键集将来可能是Bloom过滤器或
//磁盘已备份以节省内存。
	errors := false
	gcs := cid.NewSet()
	getLinks := func(ctx context.Context, cid cid.Cid) ([]*ipld.Link, error) {
		links, err := ipld.GetLinks(ctx, ng, cid)
		if err != nil {
			errors = true
			select {
			case output <- Result{Error: &CannotFetchLinksError{cid, err}}:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return links, nil
	}
	err := Descendants(ctx, getLinks, gcs, pn.RecursiveKeys())
	if err != nil {
		errors = true
		select {
		case output <- Result{Error: err}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	bestEffortGetLinks := func(ctx context.Context, cid cid.Cid) ([]*ipld.Link, error) {
		links, err := ipld.GetLinks(ctx, ng, cid)
		if err != nil && err != ipld.ErrNotFound {
			errors = true
			select {
			case output <- Result{Error: &CannotFetchLinksError{cid, err}}:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
		return links, nil
	}
	err = Descendants(ctx, bestEffortGetLinks, gcs, bestEffortRoots)
	if err != nil {
		errors = true
		select {
		case output <- Result{Error: err}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	for _, k := range pn.DirectKeys() {
		gcs.Add(k)
	}

	err = Descendants(ctx, getLinks, gcs, pn.InternalPins())
	if err != nil {
		errors = true
		select {
		case output <- Result{Error: err}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	if errors {
		return nil, ErrCannotFetchAllLinks
	}

	return gcs, nil
}

//errcanotfetchallinks作为gc输出的最后一个结果返回
//创建标记集时出错，原因是
//查找后代时出现问题。
var ErrCannotFetchAllLinks = errors.New("garbage collection aborted: could not retrieve some links")

//删除标记为的块时返回errCanNotDeleteSomBlocks
//删除失败，这是GC输出通道的最后一个结果。
var ErrCannotDeleteSomeBlocks = errors.New("garbage collection incomplete: could not delete some blocks")

//cannotfetchlinkserror提供有关哪些链接的详细信息
//无法获取，因此可能出现在GC输出通道中。
type CannotFetchLinksError struct {
	Key cid.Cid
	Err error
}

//错误实现此类型的错误接口
//消息。
func (e *CannotFetchLinksError) Error() string {
	return fmt.Sprintf("could not retrieve links for %s: %s", e.Key, e.Err)
}

//CanNotDeleteBlockError提供有关以下内容的详细信息：
//无法删除块，因此可能出现在gc输出中
//通道。
type CannotDeleteBlockError struct {
	Key cid.Cid
	Err error
}

//错误实现此类型的错误接口
//有用的信息。
func (e *CannotDeleteBlockError) Error() string {
	return fmt.Sprintf("could not remove %s: %s", e.Key, e.Err)
}
