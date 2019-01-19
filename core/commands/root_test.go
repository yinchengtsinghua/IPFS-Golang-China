
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package commands

import (
	"testing"
)

func TestCommandTree(t *testing.T) {
	printErrors := func(errs map[string][]error) {
		if errs == nil {
			return
		}
		t.Error("In Root command tree:")
		for cmd, err := range errs {
			t.Errorf("  In X command %s:", cmd)
			for _, e := range err {
				t.Errorf("    %s", e)
			}
		}
	}
	printErrors(Root.DebugValidate())
	printErrors(RootRO.DebugValidate())
}
