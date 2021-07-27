// +build windows

package sarsys

// For Windows Systems

import (
	"os"
	"syscall"
	"time"
	"unsafe"
)

// DiskUsage structure
type DiskUsage struct {
	freeBytes  int64
	totalBytes int64
	availBytes int64
}

// NeNwDiskUsage - Returns an object holding the disk usage of volumePath
// This function assumes volumePath is a valid path
func NewDiskUsage(volumePath string) *DiskUsage {

	h := syscall.MustLoadDLL("kernel32.dll")
	c := h.MustFindProc("GetDiskFreeSpaceExW")

	du := &DiskUsage{}

	c.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(volumePath))),
		uintptr(unsafe.Pointer(&du.freeBytes)),
		uintptr(unsafe.Pointer(&du.totalBytes)),
		uintptr(unsafe.Pointer(&du.availBytes)))

	return du
}

// Total free bytes on file system
func (du *DiskUsage) Free() uint64 {
	return uint64(du.freeBytes)
}

// Total available bytes on file system to an unpriveleged user
func (du *DiskUsage) Available() uint64 {
	return uint64(du.availBytes)
}

// Total size of the file system
func (du *DiskUsage) Size() uint64 {
	return uint64(du.totalBytes)
}

// Total bytes used in file system
func (du *DiskUsage) Used() uint64 {
	return du.Size() - du.Free()
}

// Percentage of use on the file system
func (du *DiskUsage) Usage() float32 {
	return float32(du.Used()) / float32(du.Size())
}

// FIleTime - Holds Amend, Modification & Creation Times
type FileTime struct {
	Atime time.Time // Access
	Mtime time.Time // Modification
	Ctime time.Time // Creation
}

// NewTime - Times of file in windows
func (ft *FileTime) NewTime(fi os.FileInfo) {
	stat := fi.Sys().(*syscall.Win32FileAttributeData)
	ft.Atime = time.Since(time.Unix(0, stat.LastAccessTime.Nanoseconds()))
	ft.Ctime = time.Since(time.Unix(0, stat.CreationTime.Nanoseconds()))
	ft.Mtime = fi.ModTime()
	// ft.Mtime = time.Since(time.Unix(0, stat.LastWriteTime.Nanoseconds()))
}
