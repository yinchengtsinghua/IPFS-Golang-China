
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"testing"

	"github.com/ipfs/go-ipfs/thirdparty/unit"

	random "gx/ipfs/QmSJ9n2s9NUoA9D849W5jj5SJ94nMcZpj1jCgQJieiNqSt/go-random"
	config "gx/ipfs/QmcRKBUqc2p3L1ZraoJjbXfs9E6xzvEuyK9iypb5RGwfsr/go-ipfs-config"
)

var (
	debug  = flag.Bool("debug", false, "direct ipfs output to console")
	online = flag.Bool("online", false, "run the benchmarks with a running daemon")
)

func main() {
	flag.Parse()
	if err := compareResults(); err != nil {
		log.Fatal(err)
	}
}

func compareResults() error {
	var amount unit.Information
	for amount = 10 * unit.MB; amount > 0; amount = amount * 2 {
if results, err := benchmarkAdd(int64(amount)); err != nil { //做比较
			return err
		} else {
			log.Println(amount, "\t", results)
		}
	}
	return nil
}

func benchmarkAdd(amount int64) (*testing.BenchmarkResult, error) {
	var benchmarkError error
	results := testing.Benchmark(func(b *testing.B) {
		b.SetBytes(amount)
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			tmpDir, err := ioutil.TempDir("", "")
			if err != nil {
				benchmarkError = err
				b.Fatal(err)
			}
			defer os.RemoveAll(tmpDir)

			env := append(
[]string{fmt.Sprintf("%s=%s", config.EnvDir, path.Join(tmpDir, config.DefaultPathName))}, //首先要覆盖
				os.Environ()...,
			)
			setupCmd := func(cmd *exec.Cmd) {
				cmd.Env = env
				if *debug {
					cmd.Stdout = os.Stdout
					cmd.Stderr = os.Stderr
				}
			}

			initCmd := exec.Command("ipfs", "init", "-b=1024")
			setupCmd(initCmd)
			if err := initCmd.Run(); err != nil {
				benchmarkError = err
				b.Fatal(err)
			}

			const seed = 1
			f, err := ioutil.TempFile("", "")
			if err != nil {
				benchmarkError = err
				b.Fatal(err)
			}
			defer os.Remove(f.Name())

			random.WritePseudoRandomBytes(amount, f, seed)
			if err := f.Close(); err != nil {
				benchmarkError = err
				b.Fatal(err)
			}

			func() {
//Fixme联机模式不工作。客户端抱怨它无法打开级别数据库
				if *online {
					daemonCmd := exec.Command("ipfs", "daemon")
					setupCmd(daemonCmd)
					if err := daemonCmd.Start(); err != nil {
						benchmarkError = err
						b.Fatal(err)
					}
					defer daemonCmd.Wait()
					defer daemonCmd.Process.Signal(os.Interrupt)
				}

				b.StartTimer()
				addCmd := exec.Command("ipfs", "add", f.Name())
				setupCmd(addCmd)
				if err := addCmd.Run(); err != nil {
					benchmarkError = err
					b.Fatal(err)
				}
				b.StopTimer()
			}()
		}
	})
	if benchmarkError != nil {
		return nil, benchmarkError
	}
	return &results, nil
}
