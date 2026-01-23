//go:build linux

package system

import (
	"syscall"
)

type StatFS struct {
	Bsize  int64
	Blocks uint64
	Bfree  uint64
}

func Statfs(path string, stat *StatFS) error {
	var sysstat syscall.Statfs_t
	if err := syscall.Statfs(path, &sysstat); err != nil {
		return err
	}
	stat.Bsize = sysstat.Bsize
	stat.Blocks = sysstat.Blocks
	stat.Bfree = sysstat.Bfree
	return nil
}
