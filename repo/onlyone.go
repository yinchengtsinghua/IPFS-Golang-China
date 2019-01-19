
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package repo

import (
	"sync"
)

//OnlyOne通过任意键跟踪打开的回购，并返回
//打开一个。
type OnlyOne struct {
	mu     sync.Mutex
	active map[interface{}]*ref
}

//打开由键标识的回购。如果回购尚未打开，则
//调用open函数，并进一步记住结果
//使用。
//
//钥匙必须是可比的，否则打开会恐慌。一定要选钥匙
//在不同的具体回购实施中是独一无二的，
//例如，通过创建本地类型：
//
//类型repokey字符串
//R，错误：=O.open（repokey（path），open）
//
//调用repo。完成后关闭。
func (o *OnlyOne) Open(key interface{}, open func() (Repo, error)) (Repo, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.active == nil {
		o.active = make(map[interface{}]*ref)
	}

	item, found := o.active[key]
	if !found {
		repo, err := open()
		if err != nil {
			return nil, err
		}
		item = &ref{
			parent: o,
			key:    key,
			Repo:   repo,
		}
		o.active[key] = item
	}
	item.refs++
	return item, nil
}

type ref struct {
	parent *OnlyOne
	key    interface{}
	refs   uint32
	Repo
}

var _ Repo = (*ref)(nil)

func (r *ref) Close() error {
	r.parent.mu.Lock()
	defer r.parent.mu.Unlock()

	r.refs--
	if r.refs > 0 {
//其他人把它打开了
		return nil
	}

//最后一个
	delete(r.parent.active, r.key)
	return r.Repo.Close()
}
