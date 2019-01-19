
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//cmd/ipfs为ipfs实现主cli二进制文件
package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"time"

	util "github.com/ipfs/go-ipfs/cmd/ipfs/util"
	oldcmds "github.com/ipfs/go-ipfs/commands"
	core "github.com/ipfs/go-ipfs/core"
	corecmds "github.com/ipfs/go-ipfs/core/commands"
	corehttp "github.com/ipfs/go-ipfs/core/corehttp"
	loader "github.com/ipfs/go-ipfs/plugin/loader"
	repo "github.com/ipfs/go-ipfs/repo"
	fsrepo "github.com/ipfs/go-ipfs/repo/fsrepo"

	ma "gx/ipfs/QmNTCey11oxhb1AxDnQBRHtdhap6Ctud872NjAYPYYXPuc/go-multiaddr"
	u "gx/ipfs/QmNohiVssaPw3KVLZik59DBVGTSm2dGvYT9eoXt5DQ36Yz/go-ipfs-util"
	madns "gx/ipfs/QmQc7jbDUsxUJZyFJzxVrnrWeECCct6fErEpMqtjyWvCX8/go-multiaddr-dns"
	"gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	"gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds/cli"
	"gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds/http"
	loggables "gx/ipfs/QmWZnkipEJbsdUDhgTGBnKAN1iHM2WDMNqsXHyAuaAiCgo/go-libp2p-loggables"
	osh "gx/ipfs/QmXuBJ7DR6k3rmUEKtvVMhwjmXDuJgXXPUt4LQXKBMsU93/go-os-helper"
	manet "gx/ipfs/QmZcLBXKaFe8ND5YHPkJRAwmhJGrVsi1JqDZNyJ4nRK5Mj/go-multiaddr-net"
	"gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
	logging "gx/ipfs/QmcuXC5cxs79ro2cUuHs4HQ2bkDLJUYokwL8aivcX6HW3C/go-log"
)

//日志是命令记录器
var log = logging.Logger("cmd/ipfs")

var errRequestCanceled = errors.New("request canceled")

//为测试目的声明为VaR
var dnsResolver = madns.DefaultResolver

const (
	EnvEnableProfiling = "IPFS_PROF"
	cpuProfile         = "ipfs.cpuprof"
	heapProfile        = "ipfs.memprof"
)

func loadPlugins(repoPath string) (*loader.PluginLoader, error) {
	pluginpath := filepath.Join(repoPath, "plugins")

//在加载插件之前检查repo是否可访问
	var plugins *loader.PluginLoader
	ok, err := checkPermissions(repoPath)
	if err != nil {
		return nil, err
	}
	if !ok {
		pluginpath = ""
	}
	plugins, err = loader.NewPluginLoader(pluginpath)
	if err != nil {
		log.Error("error loading plugins: ", err)
	}

	if err := plugins.Initialize(); err != nil {
		log.Error("error initializing plugins: ", err)
	}

	if err := plugins.Run(); err != nil {
		log.Error("error running plugins: ", err)
	}
	return plugins, nil
}

//主要路线图：
//-分析命令行以获取命令调用
//-如果用户请求帮助，请打印并退出。
//-运行命令调用
//-输出响应
//-如果有任何问题，打印错误，可能需要帮助。
func main() {
	os.Exit(mainRet())
}

func mainRet() int {
	rand.Seed(time.Now().UnixNano())
	ctx := logging.ContextWithLoggable(context.Background(), loggables.Uuid("session"))
	var err error

//我们将调用这个本地助手来输出错误。
//因此，我们控制如何在一个地方打印错误。
	printErr := func(err error) {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	}

	stopFunc, err := profileIfEnabled()
	if err != nil {
		printErr(err)
		return 1
	}
defer stopFunc() //尽可能晚地执行

	intrh, ctx := util.SetupInterruptHandler(ctx)
	defer intrh.Close()

//处理“ipfs版本”或“ipfs帮助”
	if len(os.Args) > 1 {
//句柄“ipfs--版本”
		if os.Args[1] == "--version" {
			os.Args[1] = "version"
		}

//handle`ipfs help` and`ipfs help<sub command>
		if os.Args[1] == "help" {
			if len(os.Args) > 2 {
				os.Args = append(os.Args[:1], os.Args[2:]...)
//处理“ipfs帮助--帮助”
//append`--help`，当命令不是'ipfs help--help`
				if os.Args[1] != "--help" {
					os.Args = append(os.Args, "--help")
				}
			} else {
				os.Args[1] = "--help"
			}
		}
	}

//输出取决于在os.args中传递的可执行文件名
//所以我们要确保它是稳定的
	os.Args[0] = "ipfs"

	buildEnv := func(ctx context.Context, req *cmds.Request) (cmds.Environment, error) {
		checkDebug(req)
		repoPath, err := getRepoPath(req)
		if err != nil {
			return nil, err
		}
		log.Debugf("config path is %s", repoPath)

		plugins, err := loadPlugins(repoPath)
		if err != nil {
			return nil, err
		}

//这将设置将初始化节点的函数
//这是为了我们可以惰性地构造节点。
		return &oldcmds.Context{
			ConfigRoot: repoPath,
			LoadConfig: loadConfig,
			ReqLog:     &oldcmds.ReqLog{},
			Plugins:    plugins,
			ConstructNode: func() (n *core.IpfsNode, err error) {
				if req == nil {
					return nil, errors.New("constructing node without a request")
				}

				r, err := fsrepo.Open(repoPath)
if err != nil { //repo归节点所有
					return nil, err
				}

//好的，一切都好。在调用时设置（用于所有权）
//然后把它还给我。
				n, err = core.NewNode(ctx, &core.BuildCfg{
					Repo: r,
				})
				if err != nil {
					return nil, err
				}

				n.SetLocal(true)
				return n, nil
			},
		}, nil
	}

	err = cli.Run(ctx, Root, os.Args, os.Stdin, os.Stdout, os.Stderr, buildEnv, makeExecutor)
	if err != nil {
		return 1
	}

//一切都比预期的好：）
	return 0
}

func checkDebug(req *cmds.Request) {
//检查用户是否要调试。期权或环境变量
	debug, _ := req.Options["debug"].(bool)
	if debug || os.Getenv("IPFS_LOGGING") == "debug" {
		u.Debug = true
		logging.SetDebugLogging()
	}
	if u.GetenvBool("DEBUG") {
		u.Debug = true
	}
}

func makeExecutor(req *cmds.Request, env interface{}) (cmds.Executor, error) {
	details := commandDetails(req.Path)
	client, err := commandShouldRunOnDaemon(*details, req, env.(*oldcmds.Context))
	if err != nil {
		return nil, err
	}

	var exctr cmds.Executor
	if client != nil && !req.Command.External {
		exctr = client.(cmds.Executor)
	} else {
		exctr = cmds.NewExecutor(req.Root)
	}

	return exctr, nil
}

func checkPermissions(path string) (bool, error) {
	_, err := os.Open(path)
	if os.IsNotExist(err) {
//repo还不存在-不加载插件，但也不会失败
		return false, nil
	}
	if os.IsPermission(err) {
//无法访问回购。出错。
		return false, fmt.Errorf("error opening repository at %s: permission denied", path)
	}

	return true, nil
}

//command details返回由path提供的命令的详细信息。
func commandDetails(path []string) *cmdDetails {
	var details cmdDetails
//在路径中查找具有CmdDetailsMap项的最后一个命令
	for i := range path {
		if cmdDetails, found := cmdDetailsMap[strings.Join(path[:i+1], "/")]; found {
			details = cmdDetails
		}
	}
	return &details
}

//commandshouldrundaemon根据命令详细信息确定
//命令应该在ipfs守护进程上执行。
//
//如果命令应该在守护进程上执行，则返回客户端；如果
//它应该在客户机上执行。如果命令必须
//也不能在上执行。
func commandShouldRunOnDaemon(details cmdDetails, req *cmds.Request, cctx *oldcmds.Context) (http.Client, error) {
	path := req.Path
//根命令。
	if len(path) < 1 {
		return nil, nil
	}

	if details.cannotRunOnClient && details.cannotRunOnDaemon {
		return nil, fmt.Errorf("command disabled: %s", path[0])
	}

	if details.doesNotUseRepo && details.canRunOnClient() {
		return nil, nil
	}

//此时需要知道API是否正在运行。我们推迟
//到这一点，这样我们就不会不必要地检查

//用户是否指定了用于此命令的API？
	apiAddrStr, _ := req.Options[corecmds.ApiOption].(string)

	client, err := getAPIClient(req.Context, cctx.ConfigRoot, apiAddrStr)
	if err == repo.ErrApiNotRunning {
		if apiAddrStr != "" && req.Command != daemonCmd {
//如果用户指定了一个API，而这个命令不是守护进程
//我们必须使用它。所以出错了。
			return nil, err
		}

//API不运行正常
} else if err != nil { //其他一些API错误
		return nil, err
	}

	if client != nil {
		if details.cannotRunOnDaemon {
//检查守护进程是否锁定。暂时保留错误文本。
			log.Debugf("Command cannot run on daemon. Checking if daemon is locked")
			if daemonLocked, _ := fsrepo.LockedByOtherProcess(cctx.ConfigRoot); daemonLocked {
				return nil, cmds.ClientError("ipfs daemon is running. please stop it to run this command")
			}
			return nil, nil
		}

		return client, nil
	}

	if details.cannotRunOnClient {
		return nil, cmds.ClientError("must run on the ipfs daemon")
	}

	return nil, nil
}

func getRepoPath(req *cmds.Request) (string, error) {
	repoOpt, found := req.Options["config"].(string)
	if found && repoOpt != "" {
		return repoOpt, nil
	}

	repoPath, err := fsrepo.BestKnownPath()
	if err != nil {
		return "", err
	}
	return repoPath, nil
}

func loadConfig(path string) (*config.Config, error) {
	return fsrepo.ConfigAt(path)
}

//StartProfiling开始CPU分析，并返回一个“stop”函数
//尽可能晚地执行。stop函数捕获memprofile。
func startProfiling() (func(), error) {
//尽早启动CPU分析
	ofi, err := os.Create(cpuProfile)
	if err != nil {
		return nil, err
	}
	pprof.StartCPUProfile(ofi)
	go func() {
		for range time.NewTicker(time.Second * 30).C {
			err := writeHeapProfileToFile()
			if err != nil {
				log.Error(err)
			}
		}
	}()

	stopProfiling := func() {
		pprof.StopCPUProfile()
ofi.Close() //被关闭处抓获
	}
	return stopProfiling, nil
}

func writeHeapProfileToFile() error {
	mprof, err := os.Create(heapProfile)
	if err != nil {
		return err
	}
defer mprof.Close() //在写入堆配置文件之后
	return pprof.WriteHeapProfile(mprof)
}

func profileIfEnabled() (func(), error) {
//Fixme这是一个临时的黑客程序，所以分析异步操作
//按预期工作。
	if os.Getenv(EnvEnableProfiling) != "" {
stopProfilingFunc, err := startProfiling() //Todo可能会将此更改为自己的选项…分析会使速度变慢。
		if err != nil {
			return nil, err
		}
		return stopProfilingFunc, nil
	}
	return func() {}, nil
}

var apiFileErrorFmt string = `Failed to parse '%[1]s/api' file.
	error: %[2]s
If you're sure go-ipfs isn't running, you can just delete it.
`
var checkIPFSUnixFmt = "Otherwise check:\n\tps aux | grep ipfs"
var checkIPFSWinFmt = "Otherwise check:\n\ttasklist | findstr ipfs"

//getapiclient检查repo和给定选项，检查
//正在运行的API服务。如果有，则返回客户机。
//否则，它将返回errapinotrunning或其他错误。
func getAPIClient(ctx context.Context, repoPath, apiAddrStr string) (http.Client, error) {
	var apiErrorFmt string
	switch {
	case osh.IsUnix():
		apiErrorFmt = apiFileErrorFmt + checkIPFSUnixFmt
	case osh.IsWindows():
		apiErrorFmt = apiFileErrorFmt + checkIPFSWinFmt
	default:
		apiErrorFmt = apiFileErrorFmt
	}

	var addr ma.Multiaddr
	var err error
	if len(apiAddrStr) != 0 {
		addr, err = ma.NewMultiaddr(apiAddrStr)
		if err != nil {
			return nil, err
		}
		if len(addr.Protocols()) == 0 {
			return nil, fmt.Errorf("multiaddr doesn't provide any protocols")
		}
	} else {
		addr, err = fsrepo.APIAddr(repoPath)
		if err == repo.ErrApiNotRunning {
			return nil, err
		}

		if err != nil {
			return nil, fmt.Errorf(apiErrorFmt, repoPath, err.Error())
		}
	}
	if len(addr.Protocols()) == 0 {
		return nil, fmt.Errorf(apiErrorFmt, repoPath, "multiaddr doesn't provide any protocols")
	}
	return apiClientForAddr(ctx, addr)
}

func apiClientForAddr(ctx context.Context, addr ma.Multiaddr) (http.Client, error) {
	addr, err := resolveAddr(ctx, addr)
	if err != nil {
		return nil, err
	}

	_, host, err := manet.DialArgs(addr)
	if err != nil {
		return nil, err
	}

	return http.NewClient(host, http.ClientWithAPIPrefix(corehttp.APIPath)), nil
}

func resolveAddr(ctx context.Context, addr ma.Multiaddr) (ma.Multiaddr, error) {
	ctx, cancelFunc := context.WithTimeout(ctx, 10*time.Second)
	defer cancelFunc()

	addrs, err := dnsResolver.Resolve(ctx, addr)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, errors.New("non-resolvable API endpoint")
	}

	return addrs[0], nil
}
