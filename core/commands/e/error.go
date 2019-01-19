
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package e

import (
	"fmt"
	"runtime/debug"
)

//typeerr返回一个错误，其字符串解释了预期的错误和收到的错误。
func TypeErr(expected, actual interface{}) error {
	return fmt.Errorf("expected type %T, got %T", expected, actual)
}

//编译时类型检查handlerError是否为错误
var _ error = New(nil)

//handlerError向错误添加堆栈跟踪
type HandlerError struct {
	Err   error
	Stack []byte
}

//错误使handlerError实现错误
func (err HandlerError) Error() string {
	return fmt.Sprintf("%s in:\n%s", err.Err.Error(), err.Stack)
}

//new返回新的handlerError
func New(err error) HandlerError {
	return HandlerError{Err: err, Stack: debug.Stack()}
}
