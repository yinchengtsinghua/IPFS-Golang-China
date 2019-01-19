
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package keystore

import ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"

//memkeystore是一个内存中的keystore实现，它不持久于
//任何后备存储器。
type MemKeystore struct {
	keys map[string]ci.PrivKey
}

func NewMemKeystore() *MemKeystore {
	return &MemKeystore{make(map[string]ci.PrivKey)}
}

//返回密钥库中是否存在密钥
func (mk *MemKeystore) Has(name string) (bool, error) {
	_, ok := mk.keys[name]
	return ok, nil
}

//在密钥库中存储密钥
func (mk *MemKeystore) Put(name string, k ci.PrivKey) error {
	if err := validateName(name); err != nil {
		return err
	}

	_, ok := mk.keys[name]
	if ok {
		return ErrKeyExists
	}

	mk.keys[name] = k
	return nil
}

//获取从密钥库中检索密钥
func (mk *MemKeystore) Get(name string) (ci.PrivKey, error) {
	if err := validateName(name); err != nil {
		return nil, err
	}

	k, ok := mk.keys[name]
	if !ok {
		return nil, ErrNoSuchKey
	}

	return k, nil
}

//删除从密钥库中删除密钥
func (mk *MemKeystore) Delete(name string) error {
	if err := validateName(name); err != nil {
		return err
	}

	delete(mk.keys, name)
	return nil
}

//list返回键标识符列表
func (mk *MemKeystore) List() ([]string, error) {
	out := make([]string, 0, len(mk.keys))
	for k := range mk.keys {
		out = append(out, k)
	}
	return out, nil
}
