//go:build darwin
// +build darwin

// For Apple Darwin Systems

package sarsys

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

// FileTime - Holds Amend, Modification & Creation Times
type FileTime struct {
	Atime time.Time // Access
	Mtime time.Time // Modification
	Ctime time.Time // Creation
}

// NewTime - times of file on Darwin
func (ft *FileTime) NewTime(fi os.FileInfo) {
	// Special handling to get a ctime
	// THIS IS PLATFORM DEPENDENT!!!!!
	stat := fi.Sys().(*syscall.Stat_t)
	ft.Atime = time.Unix(int64(stat.Atimespec.Sec), int64(stat.Atimespec.Nsec))
	// Mtime
	// ft.Mtime = time.Unix(int64(stat.Mtimespec.Sec), int64(stat.Mtimespec.Nsec))
	ft.Mtime = fi.ModTime()
	ft.Ctime = time.Unix(int64(stat.Ctimespec.Sec), int64(stat.Ctimespec.Nsec))
}

// Windows does not implement the syscall.Stat_t type we
// need, but the *nixes do. We use this
// to get owner/group on file
func GetOwnerAndGroup(finfo os.FileInfo) (int, int) {
	systat := finfo.Sys().(*syscall.Stat_t)
	if systat != nil {
		return int(systat.Uid), int(systat.Gid)
	}
	return 0, 0
}
