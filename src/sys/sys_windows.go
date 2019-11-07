// +build windows

package sys

// For Windows Systems

import (
	"syscall"
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
