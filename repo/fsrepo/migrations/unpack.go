
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
package mfsr

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
)

func unpackArchive(dist, binnom, path, out, atype string) error {
	switch atype {
	case "zip":
		return unpackZip(dist, binnom, path, out)
	case "tar.gz":
		return unpackTgz(dist, binnom, path, out)
	default:
		return fmt.Errorf("unrecognized archive type: %s", atype)
	}
}

func unpackTgz(dist, binnom, path, out string) error {
	fi, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fi.Close()

	gzr, err := gzip.NewReader(fi)
	if err != nil {
		return err
	}

	defer gzr.Close()

	var bin io.Reader
	tarr := tar.NewReader(gzr)

loop:
	for {
		th, err := tarr.Next()
		switch err {
		default:
			return err
		case io.EOF:
			break loop
		case nil:
//持续
		}

		if th.Name == dist+"/"+binnom {
			bin = tarr
			break
		}
	}

	if bin == nil {
		return fmt.Errorf("no binary found in downloaded archive")
	}

	return writeToPath(bin, out)
}

func writeToPath(rc io.Reader, out string) error {
	binfi, err := os.Create(out)
	if err != nil {
		return fmt.Errorf("error opening tmp bin path '%s': %s", out, err)
	}
	defer binfi.Close()

	_, err = io.Copy(binfi, rc)

	return err
}

func unpackZip(dist, binnom, path, out string) error {
	zipr, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("error opening zipreader: %s", err)
	}

	defer zipr.Close()

	var bin io.ReadCloser
	for _, fis := range zipr.File {
		if fis.Name == dist+"/"+binnom {
			rc, err := fis.Open()
			if err != nil {
				return fmt.Errorf("error extracting binary from archive: %s", err)
			}

			bin = rc
		}
	}

	return writeToPath(bin, out)
}
