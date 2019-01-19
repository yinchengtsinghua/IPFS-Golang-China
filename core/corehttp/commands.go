
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package corehttp

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	version "github.com/ipfs/go-ipfs"
	oldcmds "github.com/ipfs/go-ipfs/commands"
	"github.com/ipfs/go-ipfs/core"
	corecommands "github.com/ipfs/go-ipfs/core/commands"

	path "gx/ipfs/QmNYPETsdAu2uQ1k9q9S1jYEGURaLHV6cbYRSVFVRftpF8/go-path"
	cmds "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds"
	cmdsHttp "gx/ipfs/QmWGm4AbZEbnmdgVTza52MSNpEmBdFVqzmAysRbjrRyGbH/go-ipfs-cmds/http"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
)

var (
	errAPIVersionMismatch = errors.New("api version mismatch")
)

const originEnvKey = "API_ORIGIN"
const originEnvKeyDeprecate = `You are using the ` + originEnvKey + `ENV Variable.
This functionality is deprecated, and will be removed in future versions.
Instead, try either adding headers to the config, or passing them via
cli arguments:

	ipfs config API.HTTPHeaders --json '{"Access-Control-Allow-Origin": ["*"]}'
	ipfs daemon
`

//api path是安装api的路径。
const APIPath = "/api/v0"

var defaultLocalhostOrigins = []string{
"http://127.0.0.1:<port>“，
"https://127.0.0.1:<port>“，
"http://本地主机：<port>“，
"https://本地主机：<port>“，
}

func addCORSFromEnv(c *cmdsHttp.ServerConfig) {
	origin := os.Getenv(originEnvKey)
	if origin != "" {
		log.Warning(originEnvKeyDeprecate)
		c.AppendAllowedOrigins(origin)
	}
}

func addHeadersFromConfig(c *cmdsHttp.ServerConfig, nc *config.Config) {
	log.Info("Using API.HTTPHeaders:", nc.API.HTTPHeaders)

	if acao := nc.API.HTTPHeaders[cmdsHttp.ACAOrigin]; acao != nil {
		c.SetAllowedOrigins(acao...)
	}
	if acam := nc.API.HTTPHeaders[cmdsHttp.ACAMethods]; acam != nil {
		c.SetAllowedMethods(acam...)
	}
	if acac := nc.API.HTTPHeaders[cmdsHttp.ACACredentials]; acac != nil {
		for _, v := range acac {
			c.SetAllowCredentials(strings.ToLower(v) == "true")
		}
	}

	c.Headers = make(map[string][]string, len(nc.API.HTTPHeaders)+1)

//复制这些，因为配置是共享的，并且调用了此函数
//同时在多个地方。在适当的地方更新这些*是*令人兴奋的。
	for h, v := range nc.API.HTTPHeaders {
		h = http.CanonicalHeaderKey(h)
		switch h {
		case cmdsHttp.ACAOrigin, cmdsHttp.ACAMethods, cmdsHttp.ACACredentials:
//这些由CORS库处理。
		default:
			c.Headers[h] = v
		}
	}
	c.Headers["Server"] = []string{"go-ipfs/" + version.CurrentVersionNumber}
}

func addCORSDefaults(c *cmdsHttp.ServerConfig) {
//默认情况下使用localhost origins
	if len(c.AllowedOrigins()) == 0 {
		c.SetAllowedOrigins(defaultLocalhostOrigins...)
	}

//默认情况下，使用get、put、post
	if len(c.AllowedMethods()) == 0 {
		c.SetAllowedMethods("GET", "POST", "PUT")
	}
}

func patchCORSVars(c *cmdsHttp.ServerConfig, addr net.Addr) {

//我们必须从地址中获取端口，地址可能是IP6地址。
//TODO:这应该采用多加法器并从中派生端口。
	port := ""
	if tcpaddr, ok := addr.(*net.TCPAddr); ok {
		port = strconv.Itoa(tcpaddr.Port)
	} else if udpaddr, ok := addr.(*net.UDPAddr); ok {
		port = strconv.Itoa(udpaddr.Port)
	}

//我们正在用端口监听TCP/UDP。（UDP！“你说呢？是啊。。。它发生了……）
	oldOrigins := c.AllowedOrigins()
	newOrigins := make([]string, len(oldOrigins))
	for i, o := range oldOrigins {
//TODO:允许替换<host>。棘手，IP4和IP6以及主机名…
		if port != "" {
			o = strings.Replace(o, "<port>", port, -1)
		}
		newOrigins[i] = o
	}
	c.SetAllowedOrigins(newOrigins...)
}

func commandsOption(cctx oldcmds.Context, command *cmds.Command) ServeOption {
	return func(n *core.IpfsNode, l net.Listener, mux *http.ServeMux) (*http.ServeMux, error) {

		cfg := cmdsHttp.NewServerConfig()
		cfg.SetAllowedMethods("GET", "POST", "PUT")
		cfg.APIPath = APIPath
		rcfg, err := n.Repo.Config()
		if err != nil {
			return nil, err
		}

		addHeadersFromConfig(cfg, rcfg)
		addCORSFromEnv(cfg)
		addCORSDefaults(cfg)
		patchCORSVars(cfg, l.Addr())

		cmdHandler := cmdsHttp.NewHandler(&cctx, command, cfg)
		mux.Handle(APIPath+"/", cmdHandler)
		return mux, nil
	}
}

//commandsOption构造用于将命令挂接到
//HTTP服务器。
func CommandsOption(cctx oldcmds.Context) ServeOption {
	return commandsOption(cctx, corecommands.Root)
}

//commandsRooption构造用于挂接只读命令的serverOption
//进入HTTP服务器。
func CommandsROOption(cctx oldcmds.Context) ServeOption {
	return commandsOption(cctx, corecommands.RootRO)
}

//checkversionOption返回一个检查客户端IPFS版本是否匹配的服务操作。如果用户代理字符串不包含`/go ipfs/`，则不执行任何操作。
func CheckVersionOption() ServeOption {
	daemonVersion := version.ApiVersion

	return ServeOption(func(n *core.IpfsNode, l net.Listener, parent *http.ServeMux) (*http.ServeMux, error) {
		mux := http.NewServeMux()
		parent.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, APIPath) {
				cmdqry := r.URL.Path[len(APIPath):]
				pth := path.SplitList(cmdqry)

//与以前版本的向后兼容性检查
				if len(pth) >= 2 && pth[1] != "version" {
					clientVersion := r.UserAgent()
//跳过检查客户端是否不进行IPF
					if strings.Contains(clientVersion, "/go-ipfs/") && daemonVersion != clientVersion {
						http.Error(w, fmt.Sprintf("%s (%s != %s)", errAPIVersionMismatch, daemonVersion, clientVersion), http.StatusBadRequest)
						return
					}
				}
			}

			mux.ServeHTTP(w, r)
		})

		return mux, nil
	})
}
