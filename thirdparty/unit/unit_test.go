
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package unit

import "testing"

//大多数meta奖授予…

func TestByteSizeUnit(t *testing.T) {
	if 1*KB != 1*1024 {
		t.Fatal(1 * KB)
	}
	if 1*MB != 1*1024*1024 {
		t.Fail()
	}
	if 1*GB != 1*1024*1024*1024 {
		t.Fail()
	}
	if 1*TB != 1*1024*1024*1024*1024 {
		t.Fail()
	}
	if 1*PB != 1*1024*1024*1024*1024*1024 {
		t.Fail()
	}
	if 1*EB != 1*1024*1024*1024*1024*1024*1024 {
		t.Fail()
	}
}
