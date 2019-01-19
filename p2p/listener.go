
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package p2p

import (
	"errors"
	"sync"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	net "gx/ipfs/QmNgLg1NTw37iWbYPKcyK85YJ9Whs1MkPtJwhfqbNYAyKg/go-libp2p-net"
	"gx/ipfs/QmZNkThpqfVXs9GNbexPrfBbXSLNYeKrE7jwFM2oqHbyqN/go-libp2p-protocol"
	p2phost "gx/ipfs/QmaoXrM4Z41PD48JY36YqQGKQpLGjyLA2cKcLsES7YddAq/go-libp2p-host"
)

//侦听器侦听连接并将其代理到目标
type Listener interface {
	Protocol() protocol.ID
	ListenAddress() ma.Multiaddr
	TargetAddress() ma.Multiaddr

	key() string

//关闭关闭侦听器。不影响子流
	close()
}

//侦听器管理一组侦听器实现，
//检查冲突并可选地分派连接
type Listeners struct {
	sync.RWMutex

	Listeners map[string]Listener
}

func newListenersLocal() *Listeners {
	return &Listeners{
		Listeners: map[string]Listener{},
	}
}

func newListenersP2P(host p2phost.Host) *Listeners {
	reg := &Listeners{
		Listeners: map[string]Listener{},
	}

	host.SetStreamHandlerMatch("/x/", func(p string) bool {
		reg.RLock()
		defer reg.RUnlock()

		_, ok := reg.Listeners[p]
		return ok
	}, func(stream net.Stream) {
		reg.RLock()
		defer reg.RUnlock()

		l := reg.Listeners[string(stream.Protocol())]
		if l != nil {
			go l.(*remoteListener).handleStream(stream)
		}
	})

	return reg
}

//将listenerinfo注册到这个注册表并启动它
func (r *Listeners) Register(l Listener) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.Listeners[l.key()]; ok {
		return errors.New("listener already registered")
	}

	r.Listeners[l.key()] = l
	return nil
}

func (r *Listeners) Close(matchFunc func(listener Listener) bool) int {
	todo := make([]Listener, 0)
	r.Lock()
	for _, l := range r.Listeners {
		if !matchFunc(l) {
			continue
		}

		if _, ok := r.Listeners[l.key()]; ok {
			delete(r.Listeners, l.key())
			todo = append(todo, l)
		}
	}
	r.Unlock()

	for _, l := range todo {
		l.close()
	}

	return len(todo)
}
