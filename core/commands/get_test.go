
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package commands

import (
	"context"
	"fmt"
	"testing"

	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	cmdkit "gx/ipfs/Qmde5VP1qUkyQXKCfmEUA7bP64V2HAptbJ7phuPp7jXWwg/go-ipfs-cmdkit"
)

func TestGetOutputPath(t *testing.T) {
	cases := []struct {
		args    []string
		opts    cmdkit.OptMap
		outPath string
	}{
		{
			args: []string{"/ipns/multiformats.io/"},
			opts: map[string]interface{}{
				"output": "takes-precedence",
			},
			outPath: "takes-precedence",
		},
		{
			args: []string{"/ipns/multiformats.io/", "some-other-arg-to-be-ignored"},
			opts: cmdkit.OptMap{
				"output": "takes-precedence",
			},
			outPath: "takes-precedence",
		},
		{
			args:    []string{"/ipns/multiformats.io/"},
			outPath: "multiformats.io",
			opts:    cmdkit.OptMap{},
		},
		{
			args:    []string{"/ipns/multiformats.io/logo.svg/"},
			outPath: "logo.svg",
			opts:    cmdkit.OptMap{},
		},
		{
			args:    []string{"/ipns/multiformats.io", "some-other-arg-to-be-ignored"},
			outPath: "multiformats.io",
			opts:    cmdkit.OptMap{},
		},
	}

	_, err := GetCmd.GetOptions([]string{})
	if err != nil {
		t.Fatalf("error getting default command options: %v", err)
	}

	for i, tc := range cases {
		t.Run(fmt.Sprintf("%s-%d", t.Name(), i), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			req, err := cmds.NewRequest(ctx, []string{}, tc.opts, tc.args, nil, GetCmd)
			if err != nil {
				t.Fatalf("error creating a command request: %v", err)
			}

			if outPath := getOutPath(req); outPath != tc.outPath {
				t.Errorf("expected outPath %s to be %s", outPath, tc.outPath)
			}
		})
	}
}
