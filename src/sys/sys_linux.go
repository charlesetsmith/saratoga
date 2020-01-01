// +build linux

package sys

// For Linux systems

import (
	"os"
	"syscall"
	"time"
)

// DiskUsage structure
type DiskUsage struct {
	stat *syscall.Statfs_t
}

// NewDiskUsage - Returns an object holding the disk usage of volumePath
// This function assumes volumePath is a valid path
func NewDiskUsage(volumePath string) *DiskUsage {

	var stat syscall.Statfs_t
	syscall.Statfs(volumePath, &stat)
	return &DiskUsage{&stat}
}

// Free - Total free bytes on file system
func (du *DiskUsage) Free() uint64 {
	return du.stat.Bfree * uint64(du.stat.Bsize)
}

// Available - Total available bytes on file system to an unpriveleged user
func (du *DiskUsage) Available() uint64 {
	return du.stat.Bavail * uint64(du.stat.Bsize)
}

// Size - Total size of the file system
func (du *DiskUsage) Size() uint64 {
	return du.stat.Blocks * uint64(du.stat.Bsize)
}

// Used - Total bytes used in file system
func (du *DiskUsage) Used() uint64 {
	return du.Size() - du.Free()
}

// Usage - Percentage of use on the file system
func (du *DiskUsage) Usage() float32 {
	return float32(du.Used()) / float32(du.Size())
}

// FIleTime - Holds Amend, Modification & Creation Times
type FileTime struct {
	Atime time.Time
	Mtime time.Time
	Ctime time.Time
}

// NewTime - times of file on linux
func (ft *FileTime) NewTime(fi os.FileInfo) time.Time {
	stat := fi.Sys().(*syscall.Stat_t)
	ft.Atime = time.Unix(int64(stat_t.Atim.Sec), int64(stat_t.Atim.Nsec))
	ft.Mtime = time.Unix(int64(stat_t.Mtim.Sec), int64(stat_t.Mtim.Nsec))
	ft.Ctime = time.Unix(int64(stat_t.Ctim.Sec), int64(stat_t.Ctim.Nsec))
	return
}
