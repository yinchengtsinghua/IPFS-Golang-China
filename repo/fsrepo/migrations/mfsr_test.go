
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package mfsr

import (
	"io/ioutil"
	"os"
	"strconv"
	"testing"

	"github.com/ipfs/go-ipfs/thirdparty/assert"
)

func testVersionFile(v string, t *testing.T) (rp RepoPath) {
	name, err := ioutil.TempDir("", v)
	if err != nil {
		t.Fatal(err)
	}
	rp = RepoPath(name)
	return rp
}

func TestVersion(t *testing.T) {
	rp := RepoPath("")
	_, err := rp.Version()
	assert.Err(err, t, "Should throw an error when path is bad,")

	rp = RepoPath("/path/to/nowhere")
	_, err = rp.Version()
	if !os.IsNotExist(err) {
		t.Fatalf("Should throw an `IsNotExist` error when file doesn't exist: %v", err)
	}

	fsrepoV := 5

	rp = testVersionFile(strconv.Itoa(fsrepoV), t)
	_, err = rp.Version()
	assert.Err(err, t, "Bad VersionFile")

	assert.Nil(rp.WriteVersion(fsrepoV), t, "Trouble writing version")

	assert.Nil(rp.CheckVersion(fsrepoV), t, "Trouble checking the version")

	assert.Err(rp.CheckVersion(1), t, "Should throw an error for the wrong version.")
}
