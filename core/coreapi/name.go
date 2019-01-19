
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package coreapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	coreiface "github.com/ipfs/go-ipfs/core/coreapi/interface"
	caopts "github.com/ipfs/go-ipfs/core/coreapi/interface/options"
	"github.com/ipfs/go-ipfs/keystore"
	"github.com/ipfs/go-ipfs/namesys"

	ipath "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	"gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	"gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
)

type NameAPI CoreAPI

type ipnsEntry struct {
	name  string
	value coreiface.Path
}

//name返回ipnsentry名称。
func (e *ipnsEntry) Name() string {
	return e.name
}

//值返回ipnsentry值。
func (e *ipnsEntry) Value() coreiface.Path {
	return e.value
}

//publish发布新的IPN名称并返回新的IPN条目。
func (api *NameAPI) Publish(ctx context.Context, p coreiface.Path, opts ...caopts.NamePublishOption) (coreiface.IpnsEntry, error) {
	if err := api.checkPublishAllowed(); err != nil {
		return nil, err
	}

	options, err := caopts.NamePublishOptions(opts...)
	if err != nil {
		return nil, err
	}

	err = api.checkOnline(options.AllowOffline)
	if err != nil {
		return nil, err
	}

	pth, err := ipath.ParsePath(p.String())
	if err != nil {
		return nil, err
	}

	k, err := keylookup(api.privateKey, api.repo.Keystore(), options.Key)
	if err != nil {
		return nil, err
	}

	if options.TTL != nil {
		ctx = context.WithValue(ctx, "ipns-publish-ttl", *options.TTL)
	}

	eol := time.Now().Add(options.ValidTime)
	err = api.namesys.PublishWithEOL(ctx, k, pth, eol)
	if err != nil {
		return nil, err
	}

	pid, err := peer.IDFromPrivateKey(k)
	if err != nil {
		return nil, err
	}

	return &ipnsEntry{
		name:  pid.Pretty(),
		value: p,
	}, nil
}

func (api *NameAPI) Search(ctx context.Context, name string, opts ...caopts.NameResolveOption) (<-chan coreiface.IpnsResult, error) {
	options, err := caopts.NameResolveOptions(opts...)
	if err != nil {
		return nil, err
	}

	err = api.checkOnline(true)
	if err != nil {
		return nil, err
	}

	var resolver namesys.Resolver = api.namesys

	if !options.Cache {
		resolver = namesys.NewNameSystem(api.routing, api.repo.Datastore(), 0)
	}

	if !strings.HasPrefix(name, "/ipns/") {
		name = "/ipns/" + name
	}

	out := make(chan coreiface.IpnsResult)
	go func() {
		defer close(out)
		for res := range resolver.ResolveAsync(ctx, name, options.ResolveOpts...) {
			p, _ := coreiface.ParsePath(res.Path.String())

			select {
			case out <- coreiface.IpnsResult{Path: p, Err: res.Err}:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

//解析尝试解析指定名称的最新版本和
//返回其路径。
func (api *NameAPI) Resolve(ctx context.Context, name string, opts ...caopts.NameResolveOption) (coreiface.Path, error) {
	results, err := api.Search(ctx, name, opts...)
	if err != nil {
		return nil, err
	}

	err = coreiface.ErrResolveFailed
	var p coreiface.Path

	for res := range results {
		p, err = res.Path, res.Err
		if err != nil {
			break
		}
	}

	return p, err
}

func keylookup(self ci.PrivKey, kstore keystore.Keystore, k string) (crypto.PrivKey, error) {
	if k == "self" {
		return self, nil
	}

	res, err := kstore.Get(k)
	if res != nil {
		return res, nil
	}

	if err != nil && err != keystore.ErrNoSuchKey {
		return nil, err
	}

	keys, err := kstore.List()
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		privKey, err := kstore.Get(key)
		if err != nil {
			return nil, err
		}

		pubKey := privKey.GetPublic()

		pid, err := peer.IDFromPublicKey(pubKey)
		if err != nil {
			return nil, err
		}

		if pid.Pretty() == k {
			return privKey, nil
		}
	}

	return nil, fmt.Errorf("no key by the given name or PeerID was found")
}
