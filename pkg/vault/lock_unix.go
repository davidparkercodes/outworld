//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package vault

import (
	"os"
	"syscall"
)

var singleProcess = false

func lockPath(path string) (func(), error) {
	fh, err := os.OpenFile(path+".lock", os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(fh.Fd()), syscall.LOCK_EX); err != nil {
		fh.Close()
		return nil, err
	}
	return func() {
		syscall.Flock(int(fh.Fd()), syscall.LOCK_UN)
		fh.Close()
	}, nil
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
