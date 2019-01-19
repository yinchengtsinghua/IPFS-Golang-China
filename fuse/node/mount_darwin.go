
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+建设！诺福斯

package node

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	core "github.com/ipfs/go-ipfs/core"

	unix "gx/ipfs/QmVGjyM9i2msKvLXwh9VosCTgP4mL91kC7hDmqnwTTx6Hu/sys/unix"
	"gx/ipfs/QmYRGECuvQnRX73fcvPnGbYijBcGN2HbKZQ7jh26qmLiHG/semver"
)

func init() {
//这是一个黑客，但在我们需要用另一种方式做之前，这是可行的。
	platformFuseChecks = darwinFuseCheckVersion
}

//dontcheckosxfuseconfigkey是一个用来让用户告诉我们
//跳过保险丝检查。
var dontCheckOSXFUSEConfigKey = "DontCheckOSXFUSE"

//fuse version pkg是fuse版本的go-pkg URL
var fuseVersionPkg = "github.com/jbenet/go-fuse-version/fuse-version"

//当我们确定用户没有保险丝时，返回errstrfuserequired。
var errStrFuseRequired = `OSXFUSE not found.

OSXFUSE is required to mount, please install it.
NOTE: Version 2.7.2 or higher required; prior versions are known to kernel panic!
It is recommended you install it from the OSXFUSE website:

http://osxfuse.github.io/

For more help, see:

https://github.com/ipfs/go-ipfs/issues/177（Github.com/ipfs/go-ipfs/issues/177）
`

//errstrnofuseheaders包含在“go get<fuseversionpkg>的输出中，如果存在
//没有保险丝头。这意味着他们没有安装osxfuse。
var errStrNoFuseHeaders = "no such file or directory: '/usr/local/lib/libosxfuse.dylib'"

var errStrUpgradeFuse = `OSXFUSE version %s not supported.

OSXFUSE versions <2.7.2 are known to cause kernel panics!
Please upgrade to the latest OSXFUSE version.
It is recommended you install it from the OSXFUSE website:

http://osxfuse.github.io/

For more help, see:

https://github.com/ipfs/go-ipfs/issues/177（Github.com/ipfs/go-ipfs/issues/177）
`

var errStrNeedFuseVersion = `unable to check fuse version.

Dear User,

Before mounting, we must check your version of OSXFUSE. We are protecting
you from a nasty kernel panic we found in OSXFUSE versions <2.7.2.[1]. To
make matters worse, it's harder than it should be to check whether you have
the right version installed...[2]. We've automated the process with the
help of a little tool. We tried to install it, but something went wrong[3].
Please install it yourself by running:

	go get %s

You can also stop ipfs from running these checks and use whatever OSXFUSE
version you have by running:

	ipfs config %s true

[1]: https://github.com/ipfs/go-ipfs/issues/177（Github.com/ipfs/go-ipfs/issues/177）
[2]: https://Github.com/ipfs/go-ipfs/pull/533（Github.com/ipfs/go-ipfs/pull/533）网站
[3]: %s
`

var errStrFailedToRunFuseVersion = `unable to check fuse version.

Dear User,

Before mounting, we must check your version of OSXFUSE. We are protecting
you from a nasty kernel panic we found in OSXFUSE versions <2.7.2.[1]. To
make matters worse, it's harder than it should be to check whether you have
the right version installed...[2]. We've automated the process with the
help of a little tool. We tried to run it, but something went wrong[3].
Please, try to run it yourself with:

	go get %s
	fuse-version

You should see something like this:

	> fuse-version
	fuse-version -only agent
	OSXFUSE.AgentVersion: 2.7.3

Just make sure the number is 2.7.2 or higher. You can then stop ipfs from
trying to run these checks with:

	ipfs config %s true

[1]: https://github.com/ipfs/go-ipfs/issues/177（Github.com/ipfs/go-ipfs/issues/177）
[2]: https://Github.com/ipfs/go-ipfs/pull/533（Github.com/ipfs/go-ipfs/pull/533）网站
[3]: %s
`

var errStrFixConfig = `config key invalid: %s %v
You may be able to get this error to go away by setting it again:

	ipfs config %s true

Either way, please tell us at: http://github.com/ipfs/go-ipfs/问题
`

func darwinFuseCheckVersion(node *core.IpfsNode) error {
//在OSX上，检查保险丝版本。
	if runtime.GOOS != "darwin" {
		return nil
	}

	ov, errGFV := tryGFV()
	if errGFV != nil {
//如果我们失败了，用户告诉我们忽略检查，我们
//继续。这是在保险丝版本断开或用户不能
//安装它，但确保它们的保险丝版本可以工作。
		if skip, err := userAskedToSkipFuseCheck(node); err != nil {
			return err
		} else if skip {
return nil //用户告诉我们不要检查版本…好啊。。。。
		} else {
			return errGFV
		}
	}

	log.Debug("mount: osxfuse version:", ov)

	min := semver.MustParse("2.7.2")
	curr, err := semver.Make(ov)
	if err != nil {
		return err
	}

	if curr.LT(min) {
		return fmt.Errorf(errStrUpgradeFuse, ov)
	}
	return nil
}

func tryGFV() (string, error) {
//首先尝试sysctl。可能奏效！
	ov, err := trySysctl()
	if err == nil {
		return ov, nil
	}
	log.Debug(err)

	return tryGFVFromFuseVersion()
}

func trySysctl() (string, error) {
	v, err := unix.Sysctl("osxfuse.version.number")
	if err != nil {
		log.Debug("mount: sysctl osxfuse.version.number:", "failed")
		return "", err
	}
	log.Debug("mount: sysctl osxfuse.version.number:", v)
	return v, nil
}

func tryGFVFromFuseVersion() (string, error) {
	if err := ensureFuseVersionIsInstalled(); err != nil {
		return "", err
	}

	cmd := exec.Command("fuse-version", "-q", "-only", "agent", "-s", "OSXFUSE")
	out := new(bytes.Buffer)
	cmd.Stdout = out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf(errStrFailedToRunFuseVersion, fuseVersionPkg, dontCheckOSXFUSEConfigKey, err)
	}

	return out.String(), nil
}

func ensureFuseVersionIsInstalled() error {
//
	if _, err := exec.LookPath("fuse-version"); err == nil {
return nil //
	}

//尝试安装它…
	log.Debug("fuse-version: no fuse-version. attempting to install.")
	cmd := exec.Command("go", "get", "github.com/jbenet/go-fuse-version/fuse-version")
	cmdout := new(bytes.Buffer)
	cmd.Stdout = cmdout
	cmd.Stderr = cmdout
	if err := cmd.Run(); err != nil {
//好，安装保险丝版本失败。是他们没有保险丝吗？
		cmdoutstr := cmdout.String()
		if strings.Contains(cmdoutstr, errStrNoFuseHeaders) {
//对！它是！他们没有保险丝！
			return fmt.Errorf(errStrFuseRequired)
		}

		log.Debug("fuse-version: failed to install.")
		s := err.Error() + "\n" + cmdoutstr
		return fmt.Errorf(errStrNeedFuseVersion, fuseVersionPkg, dontCheckOSXFUSEConfigKey, s)
	}

//好的，再试一次…
	if _, err := exec.LookPath("fuse-version"); err != nil {
		log.Debug("fuse-version: failed to install?")
		return fmt.Errorf(errStrNeedFuseVersion, fuseVersionPkg, dontCheckOSXFUSEConfigKey, err)
	}

	log.Debug("fuse-version: install success")
	return nil
}

func userAskedToSkipFuseCheck(node *core.IpfsNode) (skip bool, err error) {
	val, err := node.Repo.GetConfigKey(dontCheckOSXFUSEConfigKey)
	if err != nil {
return false, nil //未能获取配置值。不要跳过检查。
	}

	switch val := val.(type) {
	case string:
		return val == "true", nil
	case bool:
		return val, nil
	default:
//有配置值，但它无效…不要跳过检查，请用户修复它…
		return false, fmt.Errorf(errStrFixConfig, dontCheckOSXFUSEConfigKey, val,
			dontCheckOSXFUSEConfigKey)
	}
}
