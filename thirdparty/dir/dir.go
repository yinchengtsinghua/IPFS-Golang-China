
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package dir

//TODO移动到一般位置

import (
	"errors"
	"os"
	"path/filepath"
)

//可写确保目录存在且可写
func Writable(path string) error {
//如果缺少，则构造路径
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}
//检查目录是否可写
	if f, err := os.Create(filepath.Join(path, "._check_writable")); err == nil {
		f.Close()
		os.Remove(f.Name())
	} else {
		return errors.New("'" + path + "' is not writable")
	}
	return nil
}
