
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

func main() {
	when := make(chan time.Time, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	for _, port := range []string{"5001", "8080"} {
		go func(port string) {
			defer wg.Done()
			for {
r, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s”，端口）
				if err != nil {
					continue
				}
				t := time.Now()
				when <- t
				log.Println(port, t, r.StatusCode)
				break
			}
		}(port)
	}
	wg.Wait()
	first := <-when
	second := <-when
	log.Println(second.Sub(first))
}
