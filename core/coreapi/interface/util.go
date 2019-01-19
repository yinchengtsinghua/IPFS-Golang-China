
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
)

type Reader interface {
	ReadSeekCloser
	Size() uint64
	CtxReadFull(context.Context, []byte) (int, error)
}

//readseekCloser实现读取、复制、查找和关闭的接口。
type ReadSeekCloser interface {
	io.Reader
	io.Seeker
	io.Closer
	io.WriterTo
}
