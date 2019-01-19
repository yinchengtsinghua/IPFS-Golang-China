
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//包装FSRPO
//
//要解释程序包路线图…
//
//IPFS/
//——客户/
//——client.lock<------保护client/+信号自己的PID
//珍——ipfs-client.cpuprof
//│——ipfs-client.memprof
//——配置
//——守护进程/
//珍——daemon.lock<------保护daemon/+信号自己的地址
//珍——ipfs-daemon.cpuprof
//│———ipfs-daemon.memprof
//——数据存储/
//——repo.lock<------保护数据存储/和配置
//破译——版本
package fsrepo

//TODO防止多个守护进程运行
