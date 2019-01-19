
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package unit

import "fmt"

type Information int64

const (
_  Information = iota //通过分配给空标识符忽略第一个值
	KB             = 1 << (10 * iota)
	MB
	GB
	TB
	PB
	EB
)

func (i Information) String() string {

	tmp := int64(i)

//违约
	var d = tmp
	symbol := "B"

	switch {
	case i > EB:
		d = tmp / EB
		symbol = "EB"
	case i > PB:
		d = tmp / PB
		symbol = "PB"
	case i > TB:
		d = tmp / TB
		symbol = "TB"
	case i > GB:
		d = tmp / GB
		symbol = "GB"
	case i > MB:
		d = tmp / MB
		symbol = "MB"
	case i > KB:
		d = tmp / KB
		symbol = "KB"
	}
	return fmt.Sprintf("%d %s", d, symbol)
}
