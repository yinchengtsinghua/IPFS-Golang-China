
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+构建FreeBSD

package util

import (
	"errors"
	"math"

	unix "gx/ipfs/QmVGjyM9i2msKvLXwh9VosCTgP4mL91kC7hDmqnwTTx6Hu/sys/unix"
)

func init() {
	supportsFDManagement = true
	getLimit = freebsdGetLimit
	setLimit = freebsdSetLimit
}

func freebsdGetLimit() (uint64, uint64, error) {
	rlimit := unix.Rlimit{}
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &rlimit)
	if (rlimit.Cur < 0) || (rlimit.Max < 0) {
		return 0, 0, errors.New("invalid rlimits")
	}
	return uint64(rlimit.Cur), uint64(rlimit.Max), err
}

func freebsdSetLimit(soft uint64, max uint64) error {
	if (soft > math.MaxInt64) || (max > math.MaxInt64) {
		return errors.New("invalid rlimits")
	}
	rlimit := unix.Rlimit{
		Cur: int64(soft),
		Max: int64(max),
	}
	return unix.Setrlimit(unix.RLIMIT_NOFILE, &rlimit)
}
