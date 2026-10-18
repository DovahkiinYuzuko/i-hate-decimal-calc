//go:build windows

package calc

import (
	"syscall"
	"unsafe"
)

var (
	modkernel32            = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpaceEx = modkernel32.NewProc("GetDiskFreeSpaceExW")
)

func getDiskFreeSpace(path string) (uint64, error) {
	var freeBytesAvailableToCaller uint64
	var totalNumberOfBytes uint64
	var totalNumberOfFreeBytes uint64

	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	r1, _, errSys := procGetDiskFreeSpaceEx.Call(
		uintptr(unsafe.Pointer(p)),
		uintptr(unsafe.Pointer(&freeBytesAvailableToCaller)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if r1 == 0 {
		if errSys != nil && errSys != syscall.Errno(0) {
			return 0, errSys
		}
		return 0, syscall.EINVAL
	}
	return freeBytesAvailableToCaller, nil
}
