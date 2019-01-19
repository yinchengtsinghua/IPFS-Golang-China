
//此源码被清华学神尹成大魔王专业翻译分析并修改
//尹成QQ77025077
//尹成微信18510341407
//尹成所在QQ群721929980
//尹成邮箱 yinc13@mails.tsinghua.edu.cn
//尹成毕业于清华大学,微软区块链领域全球最有价值专家
//https://mvp.microsoft.com/zh-cn/PublicProfile/4033620
//+建设！诺福斯
//+建设！窗户

package mount

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"gx/ipfs/QmSF8fPo3jgVBAy8fpdjjYqgG87dkJgUprRBHRd2tmfgpP/goprocess"
	"gx/ipfs/QmSJBsmLP1XMjv8hxYg2rUMdPDB7YUpyBo9idjrJ6Cmq6F/fuse"
	"gx/ipfs/QmSJBsmLP1XMjv8hxYg2rUMdPDB7YUpyBo9idjrJ6Cmq6F/fuse/fs"
)

var ErrNotMounted = errors.New("not mounted")

//安装工具转到IPF/保险丝/安装
type mount struct {
	mpoint   string
	filesys  fs.FS
	fuseConn *fuse.Conn

	active     bool
	activeLock *sync.RWMutex

	proc goprocess.Process
}

//mount在给定位置安装fuse fs.fs，并返回mount实例。
//父级是要将装入的ContextGroup绑定到的ContextGroup。
func NewMount(p goprocess.Process, fsys fs.FS, mountpoint string, allow_other bool) (Mount, error) {
	var conn *fuse.Conn
	var err error

	if allow_other {
		conn, err = fuse.Mount(mountpoint, fuse.AllowOther())
	} else {
		conn, err = fuse.Mount(mountpoint)
	}

	if err != nil {
		return nil, err
	}

	m := &mount{
		mpoint:     mountpoint,
		fuseConn:   conn,
		filesys:    fsys,
		active:     false,
		activeLock: &sync.RWMutex{},
proc:       goprocess.WithParent(p), //将其链接到父级。
	}
	m.proc.SetTeardown(m.unmount)

//启动安装过程。
	if err := m.mount(); err != nil {
m.Unmount() //以防万一。
		return nil, err
	}

	return m, nil
}

func (m *mount) mount() error {
	log.Infof("Mounting %s", m.MountPoint())

	errs := make(chan error, 1)
	go func() {
//fs.service块，直到卸载文件系统。
		err := fs.Serve(m.fuseConn, m.filesys)
		log.Debugf("%s is unmounted", m.MountPoint())
		if err != nil {
			log.Debugf("fs.Serve returned (%s)", err)
			errs <- err
		}
		m.setActive(false)
	}()

//等待装载过程完成或超时。
	select {
	case <-time.After(MountTimeout):
		return fmt.Errorf("mounting %s timed out", m.MountPoint())
	case err := <-errs:
		return err
	case <-m.fuseConn.Ready:
	}

//检查装载过程是否有要报告的错误
	if err := m.fuseConn.MountError; err != nil {
		return err
	}

	m.setActive(true)

	log.Infof("Mounted %s", m.MountPoint())
	return nil
}

//只调用一次umount以卸载此服务。
//请注意，关闭连接并不总是卸载
//适当地。如果发生这种情况，我们就拿出大炮
//（mount.forceunmountmanytimes，exec unmount）。
func (m *mount) unmount() error {
	log.Infof("Unmounting %s", m.MountPoint())

//尝试用保险丝库卸载
	err := fuse.Unmount(m.MountPoint())
	if err == nil {
		m.setActive(false)
		return nil
	}
	log.Warningf("fuse unmount err: %s", err)

//尝试关闭FuseConn
	err = m.fuseConn.Close()
	if err == nil {
		m.setActive(false)
		return nil
	}
	log.Warningf("fuse conn error: %s", err)

//尝试mount.forceunmountmanyTimes
	if err := ForceUnmountManyTimes(m, 10); err != nil {
		return err
	}

	log.Infof("Seemingly unmounted %s", m.MountPoint())
	m.setActive(false)
	return nil
}

func (m *mount) Process() goprocess.Process {
	return m.proc
}

func (m *mount) MountPoint() string {
	return m.mpoint
}

func (m *mount) Unmount() error {
	if !m.IsActive() {
		return ErrNotMounted
	}

//调用process close（），它只调用unmount（）一次。
	return m.proc.Close()
}

func (m *mount) IsActive() bool {
	m.activeLock.RLock()
	defer m.activeLock.RUnlock()

	return m.active
}

func (m *mount) setActive(a bool) {
	m.activeLock.Lock()
	m.active = a
	m.activeLock.Unlock()
}
