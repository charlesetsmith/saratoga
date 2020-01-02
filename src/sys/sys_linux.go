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
	Atime time.Time // Access
	Mtime time.Time // Modification
	Ctime time.Time // Creation
}

// NewTime - times of file on linux
func (ft *FileTime) NewTime(fi os.FileInfo) {
	stat := fi.Sys().(*syscall.Stat_t)
	ft.Atime = time.Unix(int64(stat.Atim.Sec), int64(stat.Atim.Nsec))
	ft.Mtime = fi.ModTime()
	// ft.Mtime = time.Unix(int64(stat.Mtim.Sec), int64(stat_t.Mtim.Nsec))
	ft.Ctime = time.Unix(int64(stat.Ctim.Sec), int64(stat.Ctim.Nsec))
	return
}
