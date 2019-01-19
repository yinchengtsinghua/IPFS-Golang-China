
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package notifier

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

//测试数据结构
type Router struct {
	queue    chan Packet
	notifier Notifier
}

type Packet struct{}

type RouterNotifiee interface {
	Enqueued(*Router, Packet)
	Forwarded(*Router, Packet)
	Dropped(*Router, Packet)
}

func (r *Router) Notify(n RouterNotifiee) {
	r.notifier.Notify(n)
}

func (r *Router) StopNotify(n RouterNotifiee) {
	r.notifier.StopNotify(n)
}

func (r *Router) notifyAll(notify func(n RouterNotifiee)) {
	r.notifier.NotifyAll(func(n Notifiee) {
		notify(n.(RouterNotifiee))
	})
}

func (r *Router) Receive(p Packet) {

	select {
case r.queue <- p: //
		r.notifyAll(func(n RouterNotifiee) {
			n.Enqueued(r, p)
		})

default: //滴
		r.notifyAll(func(n RouterNotifiee) {
			n.Dropped(r, p)
		})
	}
}

func (r *Router) Forward() {
	p := <-r.queue
	r.notifyAll(func(n RouterNotifiee) {
		n.Forwarded(r, p)
	})
}

type Metrics struct {
	enqueued  int
	forwarded int
	dropped   int
	received  chan struct{}
	sync.Mutex
}

func (m *Metrics) Enqueued(*Router, Packet) {
	m.Lock()
	m.enqueued++
	m.Unlock()
	if m.received != nil {
		m.received <- struct{}{}
	}
}

func (m *Metrics) Forwarded(*Router, Packet) {
	m.Lock()
	m.forwarded++
	m.Unlock()
	if m.received != nil {
		m.received <- struct{}{}
	}
}

func (m *Metrics) Dropped(*Router, Packet) {
	m.Lock()
	m.dropped++
	m.Unlock()
	if m.received != nil {
		m.received <- struct{}{}
	}
}

func (m *Metrics) String() string {
	m.Lock()
	defer m.Unlock()
	return fmt.Sprintf("%d enqueued, %d forwarded, %d in queue, %d dropped",
		m.enqueued, m.forwarded, m.enqueued-m.forwarded, m.dropped)
}

func TestNotifies(t *testing.T) {

	m := Metrics{received: make(chan struct{})}
	r := Router{queue: make(chan Packet, 10)}
	r.Notify(&m)

	for i := 0; i < 10; i++ {
		r.Receive(Packet{})
		<-m.received
		if m.enqueued != (1 + i) {
			t.Error("not notifying correctly", m.enqueued, 1+i)
		}

	}

	for i := 0; i < 10; i++ {
		r.Receive(Packet{})
		<-m.received
		if m.enqueued != 10 {
			t.Error("not notifying correctly", m.enqueued, 10)
		}
		if m.dropped != (1 + i) {
			t.Error("not notifying correctly", m.dropped, 1+i)
		}
	}
}

func TestStopsNotifying(t *testing.T) {
	m := Metrics{received: make(chan struct{})}
	r := Router{queue: make(chan Packet, 10)}
	r.Notify(&m)

	for i := 0; i < 5; i++ {
		r.Receive(Packet{})
		<-m.received
		if m.enqueued != (1 + i) {
			t.Error("not notifying correctly")
		}
	}

	r.StopNotify(&m)

	for i := 0; i < 5; i++ {
		r.Receive(Packet{})
		select {
		case <-m.received:
			t.Error("did not stop notifying")
		default:
		}
		if m.enqueued != 5 {
			t.Error("did not stop notifying")
		}
	}
}

func TestThreadsafe(t *testing.T) {
	N := 1000
	r := Router{queue: make(chan Packet, 10)}
	m1 := Metrics{received: make(chan struct{})}
	m2 := Metrics{received: make(chan struct{})}
	m3 := Metrics{received: make(chan struct{})}
	r.Notify(&m1)
	r.Notify(&m2)
	r.Notify(&m3)

	var n int
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		n++
		wg.Add(1)
		go func() {
			defer wg.Done()
			r.Receive(Packet{})
		}()

		if i%3 == 0 {
			n++
			wg.Add(1)
			go func() {
				defer wg.Done()
				r.Forward()
			}()
		}
	}

//排水队列
	for i := 0; i < (n * 3); i++ {
		select {
		case <-m1.received:
		case <-m2.received:
		case <-m3.received:
		}
	}

	wg.Wait()

//
//在'go test-race-cpu=5'下运行正常

	t.Log("m1", m1.String())
	t.Log("m2", m2.String())
	t.Log("m3", m3.String())

	if m1.String() != m2.String() || m2.String() != m3.String() {
		t.Error("counts disagree")
	}
}

type highwatermark struct {
	mu    sync.Mutex
	mark  int
	limit int
	errs  chan error
}

func (m *highwatermark) incr() {
	m.mu.Lock()
	m.mark++
//fmt.println（“增量”，m.mark）
	if m.mark > m.limit {
		m.errs <- fmt.Errorf("went over limit: %d/%d", m.mark, m.limit)
	}
	m.mu.Unlock()
}

func (m *highwatermark) decr() {
	m.mu.Lock()
	m.mark--
//fmt.println（“decr”，M.mark）
	if m.mark < 0 {
		m.errs <- fmt.Errorf("went under zero: %d/%d", m.mark, m.limit)
	}
	m.mu.Unlock()
}

func TestLimited(t *testing.T) {
timeout := 10 * time.Second //巨大的超时时间。
	limit := 9

	hwm := highwatermark{limit: limit, errs: make(chan error, 100)}
n := RateLimited(limit) //3轮后停止
	n.Notify(1)
	n.Notify(2)
	n.Notify(3)

	entr := make(chan struct{})
	exit := make(chan struct{})
	done := make(chan struct{})
	go func() {
		for i := 0; i < 10; i++ {
//fmt.printf（“圆形：%d\n”，i）
			n.NotifyAll(func(e Notifiee) {
				hwm.incr()
				entr <- struct{}{}
<-exit //等待
				hwm.decr()
			})
		}
		done <- struct{}{}
	}()

	for i := 0; i < 30; {
		select {
		case <-entr:
continue //让尽可能多的人进入
		case <-time.After(1 * time.Millisecond):
		}

//让一个出口
		select {
		case <-entr:
continue //如果出现时间问题。
		case exit <- struct{}{}:
		case <-time.After(timeout):
			t.Error("got stuck")
		}
		i++
	}

	select {
case <-done: //两部分完成
	case <-time.After(timeout):
		t.Error("did not finish")
	}

	close(hwm.errs)
	for err := range hwm.errs {
		t.Error(err)
	}
}
