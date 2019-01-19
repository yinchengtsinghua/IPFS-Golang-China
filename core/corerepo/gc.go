
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corerepo

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/ipfs/go-ipfs/core"
	gc "github.com/ipfs/go-ipfs/pin/gc"
	repo "github.com/ipfs/go-ipfs/repo"

	mfs "gx/ipfs/QmP9eu5X5Ax8169jNWqAJcc42mdZgzLR1aKCEzqhNoBLKk/go-mfs"
	humanize "gx/ipfs/QmPSBJL4momYnE7DcUyk2DVhD6rH488ZmHBGLbxNdhU44K/go-humanize"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("corerepo")

var ErrMaxStorageExceeded = errors.New("maximum storage limit exceeded. Try to unpin some files")

type GC struct {
	Node       *core.IpfsNode
	Repo       repo.Repo
	StorageMax uint64
	StorageGC  uint64
	SlackGB    uint64
	Storage    uint64
}

func NewGC(n *core.IpfsNode) (*GC, error) {
	r := n.Repo
	cfg, err := r.Config()
	if err != nil {
		return nil, err
	}

//检查cfg是否初始化了这些字段
//TODO:应该对所有cfg字段进行常规检查
//是否可以区分用户配置文件和默认结构？
	if cfg.Datastore.StorageMax == "" {
		r.SetConfigKey("Datastore.StorageMax", "10GB")
		cfg.Datastore.StorageMax = "10GB"
	}
	if cfg.Datastore.StorageGCWatermark == 0 {
		r.SetConfigKey("Datastore.StorageGCWatermark", 90)
		cfg.Datastore.StorageGCWatermark = 90
	}

	storageMax, err := humanize.ParseBytes(cfg.Datastore.StorageMax)
	if err != nil {
		return nil, err
	}
	storageGC := storageMax * uint64(cfg.Datastore.StorageGCWatermark) / 100

//计算storagemax和storageg水印之间的松弛空间
//用于限制GC持续时间
	slackGB := (storageMax - storageGC) / 10e9
	if slackGB < 1 {
		slackGB = 1
	}

	return &GC{
		Node:       n,
		Repo:       r,
		StorageMax: storageMax,
		StorageGC:  storageGC,
		SlackGB:    slackGB,
	}, nil
}

func BestEffortRoots(filesRoot *mfs.Root) ([]cid.Cid, error) {
	rootDag, err := filesRoot.GetDirectory().GetNode()
	if err != nil {
		return nil, err
	}

	return []cid.Cid{rootDag.Cid()}, nil
}

func GarbageCollect(n *core.IpfsNode, ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
defer cancel() //如果操作过程中发生错误
	roots, err := BestEffortRoots(n.FilesRoot)
	if err != nil {
		return err
	}
	rmed := gc.GC(ctx, n.Blockstore, n.Repo.Datastore(), n.Pinning, roots)

	return CollectResult(ctx, rmed, nil)
}

//CollectResult收集垃圾收集运行的输出并调用
//为删除的每个对象提供回调。它还将所有错误收集到
//GC完成后返回的多错误。
func CollectResult(ctx context.Context, gcOut <-chan gc.Result, cb func(cid.Cid)) error {
	var errors []error
loop:
	for {
		select {
		case res, ok := <-gcOut:
			if !ok {
				break loop
			}
			if res.Error != nil {
				errors = append(errors, res.Error)
			} else if res.KeyRemoved.Defined() && cb != nil {
				cb(res.KeyRemoved)
			}
		case <-ctx.Done():
			errors = append(errors, ctx.Err())
			break loop
		}
	}

	switch len(errors) {
	case 0:
		return nil
	case 1:
		return errors[0]
	default:
		return NewMultiError(errors...)
	}
}

//newmultiError从给定的错误切片创建新的multiError对象。
func NewMultiError(errs ...error) *MultiError {
	return &MultiError{errs[:len(errs)-1], errs[len(errs)-1]}
}

//多错误包含多个错误的结果。
type MultiError struct {
	Errors  []error
	Summary error
}

func (e *MultiError) Error() string {
	var buf bytes.Buffer
	for _, err := range e.Errors {
		buf.WriteString(err.Error())
		buf.WriteString("; ")
	}
	buf.WriteString(e.Summary.Error())
	return buf.String()
}

func GarbageCollectAsync(n *core.IpfsNode, ctx context.Context) <-chan gc.Result {
	roots, err := BestEffortRoots(n.FilesRoot)
	if err != nil {
		out := make(chan gc.Result)
		out <- gc.Result{Error: err}
		close(out)
		return out
	}

	return gc.GC(ctx, n.Blockstore, n.Repo.Datastore(), n.Pinning, roots)
}

func PeriodicGC(ctx context.Context, node *core.IpfsNode) error {
	cfg, err := node.Repo.Config()
	if err != nil {
		return err
	}

	if cfg.Datastore.GCPeriod == "" {
		cfg.Datastore.GCPeriod = "1h"
	}

	period, err := time.ParseDuration(cfg.Datastore.GCPeriod)
	if err != nil {
		return err
	}
	if int64(period) == 0 {
//如果持续时间为0，则表示禁用GC。
		return nil
	}

	gc, err := NewGC(node)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(period):
//private func maybegc不计算storagemax、storagegc、slackgc，因此不会为每个周期重新计算它们。
			if err := gc.maybeGC(ctx, 0); err != nil {
				log.Error(err)
			}
		}
	}
}

func ConditionalGC(ctx context.Context, node *core.IpfsNode, offset uint64) error {
	gc, err := NewGC(node)
	if err != nil {
		return err
	}
	return gc.maybeGC(ctx, offset)
}

func (gc *GC) maybeGC(ctx context.Context, offset uint64) error {
	storage, err := gc.Repo.GetStorageUsage()
	if err != nil {
		return err
	}

	if storage+offset > gc.StorageGC {
		if storage+offset > gc.StorageMax {
			log.Warningf("pre-GC: %s", ErrMaxStorageExceeded)
		}

//在这里做GC
		log.Info("Watermark exceeded. Starting repo GC...")
		defer log.EventBegin(ctx, "repoGC").Done()

		if err := GarbageCollect(gc.Node, ctx); err != nil {
			return err
		}
		log.Infof("Repo GC done. See `ipfs repo stat` to see how much space got freed.\n")
	}
	return nil
}
