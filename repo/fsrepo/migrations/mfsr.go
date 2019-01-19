
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package mfsr

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"
)

const VersionFile = "version"

type RepoPath string

func (rp RepoPath) VersionFile() string {
	return path.Join(string(rp), VersionFile)
}

func (rp RepoPath) Version() (int, error) {
	if rp == "" {
		return 0, fmt.Errorf("invalid repo path \"%s\"", rp)
	}

	fn := rp.VersionFile()
	if _, err := os.Stat(fn); err != nil {
		return 0, err
	}

	c, err := ioutil.ReadFile(fn)
	if err != nil {
		return 0, err
	}

	s := strings.TrimSpace(string(c))
	return strconv.Atoi(s)
}

func (rp RepoPath) CheckVersion(version int) error {
	v, err := rp.Version()
	if err != nil {
		return err
	}

	if v != version {
		return fmt.Errorf("versions differ (expected: %d, actual:%d)", version, v)
	}

	return nil
}

func (rp RepoPath) WriteVersion(version int) error {
	fn := rp.VersionFile()
	return ioutil.WriteFile(fn, []byte(fmt.Sprintf("%d\n", version)), 0644)
}
