
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package common

import (
	"fmt"
	"strings"
)

func MapGetKV(v map[string]interface{}, key string) (interface{}, error) {
	var ok bool
	var mcursor map[string]interface{}
	var cursor interface{} = v

	parts := strings.Split(key, ".")
	for i, part := range parts {
		sofar := strings.Join(parts[:i], ".")

		mcursor, ok = cursor.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("%s key is not a map", sofar)
		}

		cursor, ok = mcursor[part]
		if !ok {
			return nil, fmt.Errorf("%s key has no attributes", sofar)
		}
	}
	return cursor, nil
}

func MapSetKV(v map[string]interface{}, key string, value interface{}) error {
	var ok bool
	var mcursor map[string]interface{}
	var cursor interface{} = v

	parts := strings.Split(key, ".")
	for i, part := range parts {
		mcursor, ok = cursor.(map[string]interface{})
		if !ok {
			sofar := strings.Join(parts[:i], ".")
			return fmt.Errorf("%s key is not a map", sofar)
		}

//最后一部分？设置在这里
		if i == (len(parts) - 1) {
			mcursor[part] = value
			break
		}

		cursor, ok = mcursor[part]
if !ok || cursor == nil { //如果为空或为空，则创建映射
			mcursor[part] = map[string]interface{}{}
			cursor = mcursor[part]
		}
	}
	return nil
}
