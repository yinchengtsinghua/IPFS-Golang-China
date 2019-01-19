
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package reprovide

import (
	"context"
	"fmt"
	"time"

	backoff "gx/ipfs/QmPJUtEJsm5YLUWhF6imvyCH8KZXRJa9Wup7FDMwTy5Ufz/backoff"
	cid "gx/ipfs/QmR8BauakNcBa3RbE4nbQu76PDiJgoQgz8AJdhJuiU4TAw/go-cid"
	routing "gx/ipfs/QmTiRqrF5zkdZyrdsL5qndG1UbeWi8k8N2pYxCtXWrahR2/go-libp2p-routing"
	"gx/ipfs/QmYMQuypUbgsdNHmuCBSUJV6wdQVsBHRivNAp3efHJwZJD/go-verifcid"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

var log = logging.Logger("reprovider")

//keychanfunc是传递到内容路由的函数流式CID
type KeyChanFunc func(context.Context) (<-chan cid.Cid, error)
type doneFunc func(error)

type Reprovider struct {
	ctx     context.Context
	trigger chan doneFunc

//路由系统通过
	rsys routing.ContentRouting

	keyProvider KeyChanFunc
}

//NewReprovider创建新的Reprovider实例。
func NewReprovider(ctx context.Context, rsys routing.ContentRouting, keyProvider KeyChanFunc) *Reprovider {
	return &Reprovider{
		ctx:     ctx,
		trigger: make(chan doneFunc),

		rsys:        rsys,
		keyProvider: keyProvider,
	}
}

//运行重新为键提供“勾号”间隔或在触发时
func (rp *Reprovider) Run(tick time.Duration) {
//不要立即重新提供。
//可能刚刚启动了守护程序并立即将其关闭。
//随着正常运行时间的增加（再增加一分钟的正常运行时间）。
	after := time.After(time.Minute)
	var done doneFunc
	for {
		if tick == 0 {
			after = make(chan time.Time)
		}

		select {
		case <-rp.ctx.Done():
			return
		case done = <-rp.trigger:
		case <-after:
		}

//“mute”触发器通道，因此当调用“ipfs bitswap reprove”时
//返回“重新提供程序已在运行”错误
		unmute := rp.muteTrigger()

		err := rp.Reprovide()
		if err != nil {
			log.Debug(err)
		}

		if done != nil {
			done(err)
		}

		unmute()

		after = time.After(tick)
	}
}

//将rp.keyprovider提供的所有密钥重新提供给libp2p内容路由
func (rp *Reprovider) Reprovide() error {
	keychan, err := rp.keyProvider(rp.ctx)
	if err != nil {
		return fmt.Errorf("failed to get key chan: %s", err)
	}
	for c := range keychan {
//散列安全性
		if err := verifcid.ValidateCid(c); err != nil {
			log.Errorf("insecure hash in reprovider, %s (%s)", c, err)
			continue
		}
		op := func() error {
			err := rp.rsys.Provide(rp.ctx, c, true)
			if err != nil {
				log.Debugf("Failed to provide key: %s", err)
			}
			return err
		}

//托多：这个退避库不尊重我们的上下文，我们应该
//最终把工作环境融入其中。低优先级。
		err := backoff.Retry(op, backoff.NewExponentialBackOff())
		if err != nil {
			log.Debugf("Providing failed after number of retries: %s", err)
			return err
		}
	}
	return nil
}

//触发器在rp.run中启动重新设置进程并等待它
func (rp *Reprovider) Trigger(ctx context.Context) error {
	progressCtx, done := context.WithCancel(ctx)

	var err error
	df := func(e error) {
		err = e
		done()
	}

	select {
	case <-rp.ctx.Done():
		return context.Canceled
	case <-ctx.Done():
		return context.Canceled
	case rp.trigger <- df:
		<-progressCtx.Done()
		return err
	}
}

func (rp *Reprovider) muteTrigger() context.CancelFunc {
	ctx, cf := context.WithCancel(rp.ctx)
	go func() {
		defer cf()
		for {
			select {
			case <-ctx.Done():
				return
			case done := <-rp.trigger:
				done(fmt.Errorf("reprovider is already running"))
			}
		}
	}()

	return cf
}
