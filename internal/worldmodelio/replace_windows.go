//go:build windows

package worldmodelio

import (
	"syscall"
	"unsafe"
)

const (
	moveFileReplaceExisting = 0x1
	moveFileWriteThrough    = 0x8
)

var (
	kernel32MoveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")
)

func replaceCompletedFile(source, target string) error {
	src, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	dst, err := syscall.UTF16PtrFromString(target)
	if err != nil {
		return err
	}
	r1, _, callErr := kernel32MoveFileEx.Call(
		uintptr(unsafe.Pointer(src)),
		uintptr(unsafe.Pointer(dst)),
		uintptr(moveFileReplaceExisting|moveFileWriteThrough),
	)
	if r1 == 0 {
		if callErr != nil && callErr != syscall.Errno(0) {
			return callErr
		}
		return syscall.EINVAL
	}
	return nil
}
