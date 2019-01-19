
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package namesys

import (
	"context"
	"strings"
	"sync"
	"time"

	pin "github.com/ipfs/go-ipfs/pin"
	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	ft "gx/ipfs/QmQXze9tG878pa4Euya4rrDpyTNX3kQe4dhCaBzBozGgpe/go-unixfs"

	ci "gx/ipfs/QmNiJiXwWE3kRhZrC5ej3kSjWHm337pYfhjLGSCDNKJP2s/go-libp2p-crypto"
	routing "gx/ipfs/QmTiRqrF5zkdZyrdsL5qndG1UbeWi8k8N2pYxCtXWrahR2/go-libp2p-routing"
	ipns "gx/ipfs/QmWPFehHmySCdaGttQ48iwF7M6mBRrGE5GSPWKCuMWqJDR/go-ipns"
	pb "gx/ipfs/QmWPFehHmySCdaGttQ48iwF7M6mBRrGE5GSPWKCuMWqJDR/go-ipns/pb"
	peer "gx/ipfs/QmY5Grm8pJdiSSVsYxx4uNRgweY72EmYwuSDbRnbFok3iY/go-libp2p-peer"
	proto "gx/ipfs/QmdxUuburamoF6zF9qjeQC4WYcWGbWuRmdLacMEsW8ioD8/gogo-protobuf/proto"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
	dsquery "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore/query"
	base32 "gx/ipfs/QmfVj3x4D6Jkq9SEoi5n2NmoUomLwoeiwnYz2KQa15wRw6/base32"
)

const ipnsPrefix = "/ipns/"

const PublishPutValTimeout = time.Minute
const DefaultRecordTTL = 24 * time.Hour

//ipnsPublisher能够向ipfs发布和解析名称。
//路由系统。
type IpnsPublisher struct {
	routing routing.ValueStore
	ds      ds.Datastore

//用于确保分配IPN记录*顺序*序列号。
	mu sync.Mutex
}

//newipnspublisher为ipfs路由名称系统构造发布服务器。
func NewIpnsPublisher(route routing.ValueStore, ds ds.Datastore) *IpnsPublisher {
	if ds == nil {
		panic("nil datastore")
	}
	return &IpnsPublisher{routing: route, ds: ds}
}

//发布实现发布服务器。接受密钥对和值，
//并将其发布到路由系统
func (p *IpnsPublisher) Publish(ctx context.Context, k ci.PrivKey, value path.Path) error {
	log.Debugf("Publish %s", value)
	return p.PublishWithEOL(ctx, k, value, time.Now().Add(DefaultRecordTTL))
}

func IpnsDsKey(id peer.ID) ds.Key {
	return ds.NewKey("/ipns/" + base32.RawStdEncoding.EncodeToString([]byte(id)))
}

//publishedNames返回此节点发布的最新IPN记录，以及
//它们的过期时间。
//
//此方法不会在路由系统中搜索其他人发布的记录
//节点。
func (p *IpnsPublisher) ListPublished(ctx context.Context) (map[peer.ID]*pb.IpnsEntry, error) {
	query, err := p.ds.Query(dsquery.Query{
		Prefix: ipnsPrefix,
	})
	if err != nil {
		return nil, err
	}
	defer query.Close()

	records := make(map[peer.ID]*pb.IpnsEntry)
	for {
		select {
		case result, ok := <-query.Next():
			if !ok {
				return records, nil
			}
			if result.Error != nil {
				return nil, result.Error
			}
			e := new(pb.IpnsEntry)
			if err := proto.Unmarshal(result.Value, e); err != nil {
//不如把我们能得到的还回去。
				log.Error("found an invalid IPNS entry:", err)
				continue
			}
			if !strings.HasPrefix(result.Key, ipnsPrefix) {
				log.Errorf("datastore query for keys with prefix %s returned a key: %s", ipnsPrefix, result.Key)
				continue
			}
			k := result.Key[len(ipnsPrefix):]
			pid, err := base32.RawStdEncoding.DecodeString(k)
			if err != nil {
				log.Errorf("ipns ds key invalid: %s", result.Key)
				continue
			}
			records[peer.ID(pid)] = e
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
}

//GetPublished返回此节点发布的与
//给定对等体ID.
//
//如果“checkrouting”为真并且我们没有现有记录，则此方法将
//检查路由系统是否有任何现有记录。
func (p *IpnsPublisher) GetPublished(ctx context.Context, id peer.ID, checkRouting bool) (*pb.IpnsEntry, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	value, err := p.ds.Get(IpnsDsKey(id))
	switch err {
	case nil:
	case ds.ErrNotFound:
		if !checkRouting {
			return nil, nil
		}
		ipnskey := ipns.RecordKey(id)
		value, err = p.routing.GetValue(ctx, ipnskey)
		if err != nil {
//找不到或其他网络问题。真的不能做
//关于这个案子的任何事情。
			if err != routing.ErrNotFound {
				log.Debugf("error when determining the last published IPNS record for %s: %s", id, err)
			}

			return nil, nil
		}
	default:
		return nil, err
	}
	e := new(pb.IpnsEntry)
	if err := proto.Unmarshal(value, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (p *IpnsPublisher) updateRecord(ctx context.Context, k ci.PrivKey, value path.Path, eol time.Time) (*pb.IpnsEntry, error) {
	id, err := peer.IDFromPrivateKey(k)
	if err != nil {
		return nil, err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

//获取以前的记录序列号
	rec, err := p.GetPublished(ctx, id, true)
	if err != nil {
		return nil, err
	}

seqno := rec.GetSequence() //如果rec为零，则返回0
	if rec != nil && value != path.Path(rec.GetValue()) {
//不要费心增加序列号，除非
//价值变化。
		seqno++
	}

//创建记录
	entry, err := ipns.Create(k, []byte(value), seqno, eol)
	if err != nil {
		return nil, err
	}

//设置TTL
//托多：别那么刻薄。
	ttl, ok := checkCtxTTL(ctx)
	if ok {
		entry.Ttl = proto.Uint64(uint64(ttl.Nanoseconds()))
	}

	data, err := proto.Marshal(entry)
	if err != nil {
		return nil, err
	}

//把新记录放进去。
	if err := p.ds.Put(IpnsDsKey(id), data); err != nil {
		return nil, err
	}
	return entry, nil
}

//publishwitheol是IPN记录实现的临时代理
//有关详细信息，请参阅此处：https://github.com/ipfs/specs/tree/master/records
func (p *IpnsPublisher) PublishWithEOL(ctx context.Context, k ci.PrivKey, value path.Path, eol time.Time) error {
	record, err := p.updateRecord(ctx, k, value, eol)
	if err != nil {
		return err
	}

	return PutRecordToRouting(ctx, p.routing, k.GetPublic(), record)
}

//在已发布的记录上设置TTL是一项实验性功能。
//因此，我使用上下文来连接它，以避免发生变化。
//一路上有很多代码。
func checkCtxTTL(ctx context.Context) (time.Duration, bool) {
	v := ctx.Value("ipns-publish-ttl")
	if v == nil {
		return 0, false
	}

	d, ok := v.(time.Duration)
	return d, ok
}

func PutRecordToRouting(ctx context.Context, r routing.ValueStore, k ci.PubKey, entry *pb.IpnsEntry) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

errs := make(chan error, 2) //最多两个错误（IPN和公钥）

	if err := ipns.EmbedPublicKey(k, entry); err != nil {
		return err
	}

	id, err := peer.IDFromPublicKey(k)
	if err != nil {
		return err
	}

	go func() {
		errs <- PublishEntry(ctx, r, ipns.RecordKey(id), entry)
	}()

//如果无法从ID中提取公钥，则发布公钥
//托多：一旦v0.4.16足够广泛，我们就可以停止这样做了。
//在这一点上，我们甚至可以取消预测DHT中的/pk/名称空间
//
//注意：此检查实际上检查公钥是否已嵌入
//在IPN条目中。这张支票已经足够了，因为我们将
//如果无法从ID中提取IPN项中的公钥。
	if entry.PubKey != nil {
		go func() {
			errs <- PublishPublicKey(ctx, r, PkKeyForID(id), k)
		}()

		if err := waitOnErrChan(ctx, errs); err != nil {
			return err
		}
	}

	return waitOnErrChan(ctx, errs)
}

func waitOnErrChan(ctx context.Context, errs chan error) error {
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func PublishPublicKey(ctx context.Context, r routing.ValueStore, k string, pubk ci.PubKey) error {
	log.Debugf("Storing pubkey at: %s", k)
	pkbytes, err := pubk.Bytes()
	if err != nil {
		return err
	}

//存储关联的公钥
	timectx, cancel := context.WithTimeout(ctx, PublishPutValTimeout)
	defer cancel()
	return r.PutValue(timectx, k, pkbytes)
}

func PublishEntry(ctx context.Context, r routing.ValueStore, ipnskey string, rec *pb.IpnsEntry) error {
	timectx, cancel := context.WithTimeout(ctx, PublishPutValTimeout)
	defer cancel()

	data, err := proto.Marshal(rec)
	if err != nil {
		return err
	}

	log.Debugf("Storing ipns entry at: %s", ipnskey)
//将IPN条目存储在“/ipns/”+h（pubkey）
	return r.PutValue(timectx, ipnskey, data)
}

//InitializeKeyspace将给定密钥的IPN记录设置为
//指向空目录。
//托多：这感觉不属于这里
func InitializeKeyspace(ctx context.Context, pub Publisher, pins pin.Pinner, key ci.PrivKey) error {
	emptyDir := ft.EmptyDirNode()

//以递归方式固定，因为它可能已经被固定
//在这种情况下，做一个直接的插针会引发一个错误。
	err := pins.Pin(ctx, emptyDir, true)
	if err != nil {
		return err
	}

	err = pins.Flush()
	if err != nil {
		return err
	}

	return pub.Publish(ctx, key, path.FromCid(emptyDir.Cid()))
}

//pkkeyorid返回给定对等ID的公钥路由密钥。
func PkKeyForID(id peer.ID) string {
	return "/pk/" + string(id)
}
