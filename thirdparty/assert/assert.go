
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package assert

import "testing"

func Nil(err error, t *testing.T, msgs ...string) {
	if err != nil {
		t.Fatal(msgs, "error:", err)
	}
}

func True(v bool, t *testing.T, msgs ...string) {
	if !v {
		t.Fatal(msgs)
	}
}

func False(v bool, t *testing.T, msgs ...string) {
	True(!v, t, msgs...)
}

func Err(err error, t *testing.T, msgs ...string) {
	if err == nil {
		t.Fatal(msgs, "error:", err)
	}
}
