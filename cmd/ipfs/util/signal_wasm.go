
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package util

import (
	"context"
	"io"
)

type ctxCloser context.CancelFunc

func (c ctxCloser) Close() error {
	c()
	return nil
}

func SetupInterruptHandler(ctx context.Context) (io.Closer, context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	return ctxCloser(cancel), ctx
}
