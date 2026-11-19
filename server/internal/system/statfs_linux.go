//go:build linux

package system

import (
	"syscall"
)

// StatFS holds the filesystem counters read from statfs(2).
type StatFS struct {
	Bsize  int64
	Blocks uint64
	Bfree  uint64
}

// Statfs fills stat with the filesystem counters for path.
func Statfs(path string, stat *StatFS) error {
	var sysstat syscall.Statfs_t
	if err := syscall.Statfs(path, &sysstat); err != nil {
		return err
	}
	stat.Bsize = int64(sysstat.Bsize) // int32 on 32-bit ARM
	stat.Blocks = uint64(sysstat.Blocks)
	stat.Bfree = uint64(sysstat.Bfree)
	return nil
}
