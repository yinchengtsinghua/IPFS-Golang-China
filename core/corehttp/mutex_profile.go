
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corehttp

import (
	"net"
	"net/http"
	"runtime"
	"strconv"

	core "github.com/ipfs/go-ipfs/core"
)

//mutexfractionoption允许通过http设置runtime.setmutexprofilefraction
//使用参数“fraction”的post请求。
func MutexFractionOption(path string) ServeOption {
	return func(_ *core.IpfsNode, _ net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			if err := r.ParseForm(); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}

			asfr := r.Form.Get("fraction")
			if len(asfr) == 0 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			fr, err := strconv.Atoi(asfr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(err.Error()))
				return
			}
			log.Infof("Setting MutexProfileFraction to %d", fr)
			runtime.SetMutexProfileFraction(fr)
		})

		return mux, nil
	}
}
