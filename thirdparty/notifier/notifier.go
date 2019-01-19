
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//包通知程序提供了一个简单的通知调度程序
//希望嵌入更大的结构中
//要注册事件通知的客户端。
package notifier

import (
	"sync"

	process "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	ratelimit "gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess/ratelimit"
)

//通知是一个通用接口。客户端实现
//它们自己的notifiee接口以确保类型安全
//通知数：
//
//RocketNotifie接口类型
//倒计时（R火箭，倒计时时间，持续时间）
//升空（火箭）
//雷切多比特（火箭）
//分离的（火箭，太空舱）
//着陆（火箭）
//}
//
type Notifiee interface{}

//通知程序是通知调度程序。它的意思是
//它的零值准备好使用。
//
//火箭式结构
//通知程序。通知程序
//}
//
type Notifier struct {
mu   sync.RWMutex //警卫通知人
	nots map[Notifiee]struct{}
	lim  *ratelimit.RateLimiter
}

//rate limited返回速率限制通知程序。仅限Goroutines
//将产生。如果限制为零，则不会发生速率限制。这个
//与“通知人”相同。
func RateLimited(limit int) *Notifier {
	n := &Notifier{}
	if limit > 0 {
		n.lim = ratelimit.NewRateLimiter(process.Background(), limit)
	}
	return n
}

//通知注册通知人进行通知。这个函数
//是指在您自己的类型安全函数后面调用：
//
////模式跟踪的泛型函数
//func（r*rocket）通知（n notifiee）
//通知人通知
//}
//
////或作为其他函数的一部分
//载（宇航员）上的func（R*火箭）
//R.宇航员=附加（R.Austranuts，A）
//通知人通知（a）
//}
//
func (n *Notifier) Notify(e Notifiee) {
	n.mu.Lock()
if n.nots == nil { //所以零值就可以使用了。
		n.nots = make(map[Notifiee]struct{})
	}
	n.nots[e] = struct{}{}
	n.mu.Unlock()
}

//stopnotify停止通知e。此函数
//是指在您自己的类型安全函数后面调用：
//
////模式跟踪的泛型函数
//func（r*rocket）stopnotify（n notifiee）
//R.notifier.stopnotify（n）
//}
//
////或作为其他函数的一部分
//func（R*火箭）分离（C胶囊）
//
//R.Pix= nIL
//}
//
func (n *Notifier) StopNotify(e Notifiee) {
	n.mu.Lock()
if n.nots != nil { //所以零值就可以使用了。
		delete(n.nots, e)
	}
	n.mu.Unlock()
}

//向通知者的通知者通知所有消息。
//这是通过对每个通知调用给定的函数来完成的。它是
//旨在使用您自己的类型安全通知函数调用：
//
//func（r*rocket）发射（）
//
//N.发射（R）
//}）
//}
//
////将其设为私有，以便只有您可以使用它。这个功能是必要的
////以确保只在一个位置上进行强制转换。你控制你添加的人
////成为通知人。如果Go加上了仿制药，也许我们可以摆脱它。
////方法，但现在它就像用
////类型安全接口。
//func（r*火箭）notifyall（notify func（notifiee））
//r.notifier.notifyall（func（n notifier.notifiee）通知人）
//通知
//}）
//}
//
//注意：每个通知都是在自己的goroutine中启动的，因此它们
//可以同时处理，这样无论通知做什么
//
//钩住不小心挡住你的物体。
func (n *Notifier) NotifyAll(notify func(Notifiee)) {
	n.mu.Lock()
	defer n.mu.Unlock()

if n.nots == nil { //所以零值就可以使用了。
		return
	}

//无速率限制。
	if n.lim == nil {
		for notifiee := range n.nots {
			go notify(notifiee)
		}
		return
	}

//有速率限制。
	n.lim.Go(func(worker process.Process) {
		for notifiee := range n.nots {
notifiee := notifiee //
			n.lim.LimitedGo(func(worker process.Process) {
				notify(notifiee)
			})
		}
	})
}
