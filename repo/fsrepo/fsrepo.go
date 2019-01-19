
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package fsrepo

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	filestore "github.com/ipfs/go-ipfs/filestore"
	keystore "github.com/ipfs/go-ipfs/keystore"
	repo "github.com/ipfs/go-ipfs/repo"
	"github.com/ipfs/go-ipfs/repo/common"
	mfsr "github.com/ipfs/go-ipfs/repo/fsrepo/migrations"
	dir "github.com/ipfs/go-ipfs/thirdparty/dir"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	util "gx/ipfs/QmNohiVssaPw3KVLZik59DBVGTSm2dGvYT9eoXt5DQ36Yz/go-ipfs-util"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
	serialize "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config/serialize"
	lockfile "gx/ipfs/QmcWjZkQxyPMkgZRpda4hqWwaD6E1yqCvcxZfxbt98CEAK/go-fs-lock"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
	measure "gx/ipfs/QmdCQgMgoMjur6D15ZB3z1LodiSP3L6EBHMyVx4ekqzRWA/go-ds-measure"
	homedir "gx/ipfs/QmdcULN1WCzgoQmcCaUAmEhwcxHYsDrbZ2LvRJKCL8dMrK/go-homedir"
	ds "gx/ipfs/Qmf4xQhNomPNhrtZc67qSnfJSjxjXs9LWvknJtSXwimPrM/go-datastore"
)

//lockfile是repo lock的文件名，相对于config dir
//TODO重命名repo lock并隐藏名称
const LockFile = "repo.lock"

var log = logging.Logger("fsrepo")

//我们当前希望看到的版本号
var RepoVersion = 7

var migrationInstructions = `See https://github.com/ipfs/fs-repo-migrations/blob/master/run.md（Github.com/ipfs/fs-repo-migrations/blob/master/run.md）
Sorry for the inconvenience. In the future, these will run automatically.`

var programTooLowMessage = `Your programs version (%d) is lower than your repos (%d).
Please update ipfs to a version that supports the existing repo, or run
a migration in reverse.

See https://有关详细信息，请访问github.com/ipfs/fs-repo-migrations/blob/master/run.md。

var (
	ErrNoVersion     = errors.New("no version file found, please run 0-to-1 migration tool.\n" + migrationInstructions)
	ErrOldRepo       = errors.New("ipfs repo found in old '~/.go-ipfs' location, please run migration tool.\n" + migrationInstructions)
	ErrNeedMigration = errors.New("ipfs repo needs migration")
)

type NoRepoError struct {
	Path string
}

var _ error = NoRepoError{}

func (err NoRepoError) Error() string {
	return fmt.Sprintf("no IPFS repo found in %s.\nplease run: 'ipfs init'", err.Path)
}

const apiFile = "api"
const swarmKeyFile = "swarm.key"

const specFn = "datastore_spec"

var (

//执行任何修改
//fsrepo的状态字段。这包括init、open、close和remove。
	packageLock sync.Mutex

//只有OnlyOne跟踪打开的fsrepo实例。
//
//TODO:一旦清除了命令上下文/repo集成，
//这个可以移除。现在，这使configCmd.run
//函数尝试打开回购两次：
//
//IPFS守护进程
//$ipfs配置foo
//
//上述原因是，在独立模式下，没有
//daemon，`ipfs config`试图通过不构建
//完整的ipfsnode，但直接访问repo。
	onlyOne repo.OnlyOne
)

//
//呼叫者。
type FSRepo struct {
//已调用Close
	closed bool
//路径是文件系统路径
	path string
//lock file是防止其他人打开的文件系统锁。
//同时使用相同的fsrepo路径
	lockfile io.Closer
	config   *config.Config
	ds       repo.Datastore
	keystore keystore.Keystore
	filemgr  *filestore.FileManager
}

var _ repo.Repo = (*FSRepo)(nil)

//打开路径处的fsrepo。如果repo不是
//初始化。
func Open(repoPath string) (repo.Repo, error) {
	fn := func() (repo.Repo, error) {
		return open(repoPath)
	}
	return onlyOne.Open(repoPath, fn)
}

func open(repoPath string) (repo.Repo, error) {
	packageLock.Lock()
	defer packageLock.Unlock()

	r, err := newFSRepo(repoPath)
	if err != nil {
		return nil, err
	}

//
	if err := checkInitialized(r.path); err != nil {
		return nil, err
	}

	r.lockfile, err = lockfile.Lock(r.path, LockFile)
	if err != nil {
		return nil, err
	}
	keepLocked := false
	defer func() {
//出错时解锁，成功时保持锁定
		if !keepLocked {
			r.lockfile.Close()
		}
	}()

//检查版本，如果不匹配则出错
	ver, err := mfsr.RepoPath(r.path).Version()
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNoVersion
		}
		return nil, err
	}

	if RepoVersion > ver {
		return nil, ErrNeedMigration
	} else if ver > RepoVersion {
//现有回购的程序版本太低
		return nil, fmt.Errorf(programTooLowMessage, RepoVersion, ver)
	}

//检查回购路径，然后检查所有组成部分。
	if err := dir.Writable(r.path); err != nil {
		return nil, err
	}

	if err := r.openConfig(); err != nil {
		return nil, err
	}

	if err := r.openDatastore(); err != nil {
		return nil, err
	}

	if err := r.openKeystore(); err != nil {
		return nil, err
	}

	if r.config.Experimental.FilestoreEnabled || r.config.Experimental.UrlstoreEnabled {
		r.filemgr = filestore.NewFileManager(r.ds, filepath.Dir(r.path))
		r.filemgr.AllowFiles = r.config.Experimental.FilestoreEnabled
		r.filemgr.AllowUrls = r.config.Experimental.UrlstoreEnabled
	}

	keepLocked = true
	return r, nil
}

func newFSRepo(rpath string) (*FSRepo, error) {
	expPath, err := homedir.Expand(filepath.Clean(rpath))
	if err != nil {
		return nil, err
	}

	return &FSRepo{path: expPath}, nil
}

func checkInitialized(path string) error {
	if !isInitializedUnsynced(path) {
		alt := strings.Replace(path, ".ipfs", ".go-ipfs", 1)
		if isInitializedUnsynced(alt) {
			return ErrOldRepo
		}
		return NoRepoError{Path: path}
	}
	return nil
}

//如果给定路径的fsrepo不是
//初始化。此函数允许调用方读取配置文件，即使在
//
func ConfigAt(repoPath string) (*config.Config, error) {

//必须保持PackageLock以确保读取是原子的。
	packageLock.Lock()
	defer packageLock.Unlock()

	configFilename, err := config.Filename(repoPath)
	if err != nil {
		return nil, err
	}
	return serialize.Load(configFilename)
}

//
//提供路径。
func configIsInitialized(path string) bool {
	configFilename, err := config.Filename(path)
	if err != nil {
		return false
	}
	if !util.FileExists(configFilename) {
		return false
	}
	return true
}

func initConfig(path string, conf *config.Config) error {
	if configIsInitialized(path) {
		return nil
	}
	configFilename, err := config.Filename(path)
	if err != nil {
		return err
	}
//初始化是可以写入配置文件的一次
//不从磁盘读取配置，也不合并任何用户提供的密钥
//
	if err := serialize.WriteConfigFile(configFilename, conf); err != nil {
		return err
	}

	return nil
}

func initSpec(path string, conf map[string]interface{}) error {
	fn, err := config.Path(path, specFn)
	if err != nil {
		return err
	}

	if util.FileExists(fn) {
		return nil
	}

	dsc, err := AnyDatastoreConfig(conf)
	if err != nil {
		return err
	}
	bytes := dsc.DiskSpec().Bytes()

	return ioutil.WriteFile(fn, bytes, 0600)
}

//init使用提供的配置在给定路径上初始化新的fsrepo。
//TODO添加对自定义数据存储的支持。
func Init(repoPath string, conf *config.Config) error {

//必须保持PackageLock以确保不再对repo进行更多初始化。
//不止一次。
	packageLock.Lock()
	defer packageLock.Unlock()

	if isInitializedUnsynced(repoPath) {
		return nil
	}

	if err := initConfig(repoPath, conf); err != nil {
		return err
	}

	if err := initSpec(repoPath, conf.Datastore.Spec); err != nil {
		return err
	}

	if err := mfsr.RepoPath(repoPath).WriteVersion(RepoVersion); err != nil {
		return err
	}

	return nil
}

//如果fsrepo被另一个进程锁定，则lockedbyotherprocess返回true
//过程。如果为真，则此进程无法打开回购。
func LockedByOtherProcess(repoPath string) (bool, error) {
	repoPath = filepath.Clean(repoPath)
	locked, err := lockfile.Locked(repoPath, LockFile)
	if locked {
		log.Debugf("(%t)<->Lock is held at %s", locked, repoPath)
	}
	return locked, err
}

//api addr根据api文件返回已注册的api addr
//
//进程可以读取此文件。因此，修改此文件应该
//使用“mv”替换整个文件，避免交错读/写。
func APIAddr(repoPath string) (ma.Multiaddr, error) {
	repoPath = filepath.Clean(repoPath)
	apiFilePath := filepath.Join(repoPath, apiFile)

//如果没有文件，假设没有api地址。
	f, err := os.Open(apiFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, repo.ErrApiNotRunning
		}
		return nil, err
	}
	defer f.Close()

//最多读取2048字节。io.readall是一个漏洞，如
//有人可以通过在那里放一个大文件来破坏这个过程。
//
//注意（@stebalien）：@jbenet可能没有在思考的时候
//写了那条评论，但我要在这里留下限制，以防
//一些隐藏的智慧。但是，我正在修复它，以便：
//1。我们读书不多。
//2。我们不会截短和成功。
	buf, err := ioutil.ReadAll(io.LimitReader(f, 2048))
	if err != nil {
		return nil, err
	}
	if len(buf) == 2048 {
		return nil, fmt.Errorf("API file too large, must be <2048 bytes long: %s", apiFilePath)
	}

	s := string(buf)
	s = strings.TrimSpace(s)
	return ma.NewMultiaddr(s)
}

func (r *FSRepo) Keystore() keystore.Keystore {
	return r.keystore
}

func (r *FSRepo) Path() string {
	return r.path
}

//setapiaddr将api addr写入/api文件。
func (r *FSRepo) SetAPIAddr(addr ma.Multiaddr) error {
	f, err := os.Create(filepath.Join(r.path, apiFile))
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(addr.String())
	return err
}

//如果配置文件不存在，则openconfig返回错误。
func (r *FSRepo) openConfig() error {
	configFilename, err := config.Filename(r.path)
	if err != nil {
		return err
	}
	conf, err := serialize.Load(configFilename)
	if err != nil {
		return err
	}
	r.config = conf
	return nil
}

func (r *FSRepo) openKeystore() error {
	ksp := filepath.Join(r.path, "keystore")
	ks, err := keystore.NewFSKeystore(ksp)
	if err != nil {
		return err
	}

	r.keystore = ks

	return nil
}

//如果配置文件不存在，则opendatastore返回一个错误。
func (r *FSRepo) openDatastore() error {
	if r.config.Datastore.Type != "" || r.config.Datastore.Path != "" {
		return fmt.Errorf("old style datatstore config detected")
	} else if r.config.Datastore.Spec == nil {
		return fmt.Errorf("required Datastore.Spec entry missing from config file")
	}
	if r.config.Datastore.NoSync {
log.Warning("NoSync is now deprecated in favor of datastore specific settings. If you want to disable fsync on flatfs set 'sync' to false. See https://github.com/ipfs/go ipfs/blob/master/docs/datastores.md flatfs.”）
	}

	dsc, err := AnyDatastoreConfig(r.config.Datastore.Spec)
	if err != nil {
		return err
	}
	spec := dsc.DiskSpec()

	oldSpec, err := r.readSpec()
	if err != nil {
		return err
	}
	if oldSpec != spec.String() {
		return fmt.Errorf("datastore configuration of '%s' does not match what is on disk '%s'",
			oldSpec, spec.String())
	}

	d, err := dsc.Create(r.path)
	if err != nil {
		return err
	}
	r.ds = d

//用度量收集来包装它
	prefix := "ipfs.fsrepo.datastore"
	r.ds = measure.New(prefix, r.ds)

	return nil
}

func (r *FSRepo) readSpec() (string, error) {
	fn, err := config.Path(r.path, specFn)
	if err != nil {
		return "", err
	}
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

//关闭关闭fsrepo，释放保留的资源。
func (r *FSRepo) Close() error {
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return errors.New("repo is closed")
	}

	err := os.Remove(filepath.Join(r.path, apiFile))
	if err != nil && !os.IsNotExist(err) {
		log.Warning("error removing api file: ", err)
	}

	if err := r.ds.Close(); err != nil {
		return err
	}

//此代码存在于以前的版本中，但是
//从未调用EventLogComponent.Close。保存在这里
//等待进一步讨论。
//
//Todo它不是当前合同的一部分，但呼叫者可能会喜欢我们
//关闭组件后禁用日志记录。
//logging.configure（logging.output（os.stderr））。

	r.closed = true
	return r.lockfile.Close()
}

//配置当前配置。此函数不复制配置。呼叫者
//必须先调用“clone”才能修改它。
//
//未打开时的结果未定义。如果愿意，这种方法可能会惊慌失措。
func (r *FSRepo) Config() (*config.Config, error) {
//由于回购处于
//打开状态。程序包锁不是用来确保回购
//线程安全。包装锁仅用于防止拆卸和
//协调锁定文件。但是，我们提供线程安全保护
//事情很简单。
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return nil, errors.New("cannot access config, repo not open")
	}
	return r.config, nil
}

func (r *FSRepo) FileManager() *filestore.FileManager {
	return r.filemgr
}

func (r *FSRepo) BackupConfig(prefix string) (string, error) {
	temp, err := ioutil.TempFile(r.path, "config-"+prefix)
	if err != nil {
		return "", err
	}
	defer temp.Close()

	configFilename, err := config.Filename(r.path)
	if err != nil {
		return "", err
	}

	orig, err := os.OpenFile(configFilename, os.O_RDONLY, 0600)
	if err != nil {
		return "", err
	}
	defer orig.Close()

	_, err = io.Copy(temp, orig)
	if err != nil {
		return "", err
	}

	return orig.Name(), nil
}

//setconfigunsynched仅供私人使用。
func (r *FSRepo) setConfigUnsynced(updated *config.Config) error {
	configFilename, err := config.Filename(r.path)
	if err != nil {
		return err
	}
//为了避免删除用户提供的密钥，必须从磁盘读取配置。
//作为映射，将更新的结构值写入映射并写入映射
//到磁盘。
	var mapconf map[string]interface{}
	if err := serialize.ReadConfigFile(configFilename, &mapconf); err != nil {
		return err
	}
	m, err := config.ToMap(updated)
	if err != nil {
		return err
	}
	for k, v := range m {
		mapconf[k] = v
	}
	if err := serialize.WriteConfigFile(configFilename, mapconf); err != nil {
		return err
	}
//不要使用“*r.config=…”。这将修改*共享*配置
//由“r.config”返回。
	r.config = updated
	return nil
}

//setconfig更新fsrepo的配置。用户不得修改配置
//对象。
func (r *FSRepo) SetConfig(updated *config.Config) error {

//PackageLock用于提供螺纹安全性。
	packageLock.Lock()
	defer packageLock.Unlock()

	return r.setConfigUnsynced(updated)
}

//getconfigkey只检索特定键的值。
func (r *FSRepo) GetConfigKey(key string) (interface{}, error) {
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return nil, errors.New("repo is closed")
	}

	filename, err := config.Filename(r.path)
	if err != nil {
		return nil, err
	}
	var cfg map[string]interface{}
	if err := serialize.ReadConfigFile(filename, &cfg); err != nil {
		return nil, err
	}
	return common.MapGetKV(cfg, key)
}

//setconfigkey写入特定键的值。
func (r *FSRepo) SetConfigKey(key string, value interface{}) error {
	packageLock.Lock()
	defer packageLock.Unlock()

	if r.closed {
		return errors.New("repo is closed")
	}

	filename, err := config.Filename(r.path)
	if err != nil {
		return err
	}
	var mapconf map[string]interface{}
	if err := serialize.ReadConfigFile(filename, &mapconf); err != nil {
		return err
	}

//加载私钥以防止它被覆盖。
//注意：这是在我们移动之前保护此字段的临时措施
//键退出配置文件。
	pkval, err := common.MapGetKV(mapconf, config.PrivKeySelector)
	if err != nil {
		return err
	}

//获取与键关联的值的类型
	oldValue, err := common.MapGetKV(mapconf, key)
	ok := true
	if err != nil {
//键值尚不存在
		switch v := value.(type) {
		case string:
			value, err = strconv.ParseBool(v)
			if err != nil {
				value, err = strconv.Atoi(v)
				if err != nil {
					value, err = strconv.ParseFloat(v, 32)
					if err != nil {
						value = v
					}
				}
			}
		default:
		}
	} else {
		switch oldValue.(type) {
		case bool:
			value, ok = value.(bool)
		case int:
			value, ok = value.(int)
		case float32:
			value, ok = value.(float32)
		case string:
			value, ok = value.(string)
		default:
		}
		if !ok {
			return fmt.Errorf("wrong config type, expected %T", oldValue)
		}
	}

	if err := common.MapSetKV(mapconf, key, value); err != nil {
		return err
	}

//替换私钥，以防它被覆盖。
	if err := common.MapSetKV(mapconf, config.PrivKeySelector, pkval); err != nil {
		return err
	}

//此步骤加倍以根据结构验证映射
//序列化前
	conf, err := config.FromMap(mapconf)
	if err != nil {
		return err
	}
	if err := serialize.WriteConfigFile(filename, mapconf); err != nil {
		return err
	}
return r.setConfigUnsynced(conf) //TODO将此转入此方法
}

//数据存储返回回购拥有的数据存储。如果fsrepo关闭，返回值
//是未定义的。
func (r *FSRepo) Datastore() repo.Datastore {
	packageLock.Lock()
	d := r.ds
	packageLock.Unlock()
	return d
}

//GetStorageUsage计算repo所占用的存储空间（字节）
func (r *FSRepo) GetStorageUsage() (uint64, error) {
	return ds.DiskUsage(r.Datastore())
}

func (r *FSRepo) SwarmKey() ([]byte, error) {
	repoPath := filepath.Clean(r.path)
	spath := filepath.Join(repoPath, swarmKeyFile)

	f, err := os.Open(spath)
	if err != nil {
		if os.IsNotExist(err) {
			err = nil
		}
		return nil, err
	}
	defer f.Close()

	return ioutil.ReadAll(f)
}

var _ io.Closer = &FSRepo{}
var _ repo.Repo = &FSRepo{}

//如果在提供的路径处初始化repo，则IsInitialized返回true。
func IsInitialized(path string) bool {
//PackageLock用于确保另一个调用方不会尝试
//在进行此调用时初始化或删除repo。
	packageLock.Lock()
	defer packageLock.Unlock()

	return isInitializedUnsynced(path)
}

//低于此点的私有方法。注意：PackageLock必须由呼叫者持有。

//IsInitializedUnsynced报告是否初始化repo。来电者必须
//握住包装盒。
func isInitializedUnsynced(repoPath string) bool {
	return configIsInitialized(repoPath)
}
