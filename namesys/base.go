
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package namesys

import (
	"context"
	"strings"
	"time"

	opts "github.com/ipfs/go-ipfs/namesys/opts"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
)

type onceResult struct {
	value path.Path
	ttl   time.Duration
	err   error
}

type resolver interface {
	resolveOnceAsync(ctx context.Context, name string, options opts.ResolveOpts) <-chan onceResult
}

//resolve是使用resolveonce实现resolver.resolven的帮助程序。
func resolve(ctx context.Context, r resolver, name string, options opts.ResolveOpts) (path.Path, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	err := ErrResolveFailed
	var p path.Path

	resCh := resolveAsync(ctx, r, name, options)

	for res := range resCh {
		p, err = res.Path, res.Err
		if err != nil {
			break
		}
	}

	return p, err
}

func resolveAsync(ctx context.Context, r resolver, name string, options opts.ResolveOpts) <-chan Result {
	resCh := r.resolveOnceAsync(ctx, name, options)
	depth := options.Depth
	outCh := make(chan Result, 1)

	go func() {
		defer close(outCh)
		var subCh <-chan Result
		var cancelSub context.CancelFunc
		defer func() {
			if cancelSub != nil {
				cancelSub()
			}
		}()

		for {
			select {
			case res, ok := <-resCh:
				if !ok {
					resCh = nil
					break
				}

				if res.err != nil {
					emitResult(ctx, outCh, Result{Err: res.err})
					return
				}
				log.Debugf("resolved %s to %s", name, res.value.String())
				if !strings.HasPrefix(res.value.String(), ipnsPrefix) {
					emitResult(ctx, outCh, Result{Path: res.value})
					break
				}

				if depth == 1 {
					emitResult(ctx, outCh, Result{Path: res.value, Err: ErrResolveRecursion})
					break
				}

				subopts := options
				if subopts.Depth > 1 {
					subopts.Depth--
				}

				var subCtx context.Context
				if cancelSub != nil {
//取消以前的递归解析，因为无论如何都不会使用它
					cancelSub()
				}
				subCtx, cancelSub = context.WithCancel(ctx)
				_ = cancelSub

				p := strings.TrimPrefix(res.value.String(), ipnsPrefix)
				subCh = resolveAsync(subCtx, r, p, subopts)
			case res, ok := <-subCh:
				if !ok {
					subCh = nil
					break
				}

//在上下文超时的情况下，我们不需要返回这里
//没有很好的理由这样做，我们可能仍然能够发出一个结果
				emitResult(ctx, outCh, res)
			case <-ctx.Done():
				return
			}
			if resCh == nil && subCh == nil {
				return
			}
		}
	}()
	return outCh
}

func emitResult(ctx context.Context, outCh chan<- Result, r Result) {
	select {
	case outCh <- r:
	case <-ctx.Done():
	}
}
