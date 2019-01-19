
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//package blockstoreutil为blockstores提供实用程序功能。
package blockstoreutil

import (
	"fmt"
	"io"

	"github.com/ipfs/go-ipfs/pin"

	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	bs "gx/ipfs/QmS2aqUZLJp8kF1ihE5rvDGE5LvmKDPnx32w9Z1BW9xLV5/go-ipfs-blockstore"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
)

//REMOVEDBLOCK用于重新显示删除块的结果。
//如果成功删除块，则错误字符串将
//空的。如果无法删除块，则错误将包含
//无法删除块的原因。如果删除被中止
//由于致命错误，哈希将为空，错误将包含
//原因，将不再发送结果。
type RemovedBlock struct {
	Hash  string `json:",omitempty"`
	Error string `json:",omitempty"`
}

//RMBlocksOpts用于包装RMBlocks（）的选项。
type RmBlocksOpts struct {
	Prefix string
	Quiet  bool
	Force  bool
}

//人民币锁可以移除CIDS切片中提供的块。
//当
//不使用安静选项。块移除是异步的，将
//跳过任何固定块。
func RmBlocks(blocks bs.GCBlockstore, pins pin.Pinner, cids []cid.Cid, opts RmBlocksOpts) (<-chan interface{}, error) {
//使通道足够大以容纳任何结果以避免
//保持GClock时阻塞
	out := make(chan interface{}, len(cids))
	go func() {
		defer close(out)

		unlocker := blocks.GCLock()
		defer unlocker.Unlock()

		stillOkay := FilterPinned(pins, out, cids)

		for _, c := range stillOkay {
			err := blocks.DeleteBlock(c)
			if err != nil && opts.Force && (err == bs.ErrNotFound || err == ds.ErrNotFound) {
//忽略不存在的块
			} else if err != nil {
				out <- &RemovedBlock{Hash: c.String(), Error: err.Error()}
			} else if !opts.Quiet {
				out <- &RemovedBlock{Hash: c.String()}
			}
		}
	}()
	return out, nil
}

//filterpinned获取一个CID切片并将其与pinned CID一起返回
//远离的。如果固定了一个cid，它将在给定的
//输出通道，错误指示CID已固定。
//此函数在RMBLOCK中用于筛选任何未
//移除（因为它们是固定的）。
func FilterPinned(pins pin.Pinner, out chan<- interface{}, cids []cid.Cid) []cid.Cid {
	stillOkay := make([]cid.Cid, 0, len(cids))
	res, err := pins.CheckIfPinned(cids...)
	if err != nil {
		out <- &RemovedBlock{Error: fmt.Sprintf("pin check failed: %s", err)}
		return nil
	}
	for _, r := range res {
		if !r.Pinned() {
			stillOkay = append(stillOkay, r.Key)
		} else {
			out <- &RemovedBlock{
				Hash:  r.Key.String(),
				Error: r.String(),
			}
		}
	}
	return stillOkay
}

//procroutput接受一个函数，如果没有输入，则返回来自人民币锁或eof的结果。
//然后，它根据从函数返回的removedBlock对象写入stdout/stderr。
func ProcRmOutput(next func() (interface{}, error), sout io.Writer, serr io.Writer) error {
	someFailed := false
	for {
		res, err := next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		r := res.(*RemovedBlock)
		if r.Hash == "" && r.Error != "" {
			return fmt.Errorf("aborted: %s", r.Error)
		} else if r.Error != "" {
			someFailed = true
			fmt.Fprintf(serr, "cannot remove %s: %s\n", r.Hash, r.Error)
		} else {
			fmt.Fprintf(sout, "removed %s\n", r.Hash)
		}
	}
	if someFailed {
		return fmt.Errorf("some blocks not removed")
	}
	return nil
}
