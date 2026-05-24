//go:build windows

package singleinstance

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procCreateMutexW = modkernel32.NewProc("CreateMutexW")
	procCloseHandle  = modkernel32.NewProc("CloseHandle")
)

const errorAlreadyExists = 183

// Acquire 尝试获取 Windows named mutex。若已有实例持有该 mutex 则返回 ErrAlreadyRunning。
func Acquire(name string) (*Lock, error) {
	namePtr, err := syscall.UTF16PtrFromString(name)
	if err != nil {
		return nil, fmt.Errorf("创建互斥锁失败: %w", err)
	}
	handle, _, lastErr := procCreateMutexW.Call(0, 0, uintptr(unsafe.Pointer(namePtr)))
	if handle == 0 {
		return nil, fmt.Errorf("创建互斥锁失败: %v", lastErr)
	}
	if errno, ok := lastErr.(syscall.Errno); ok && errno == errorAlreadyExists {
		procCloseHandle.Call(handle)
		return nil, ErrAlreadyRunning
	}
	return &Lock{
		release: func() error {
			r, _, e := procCloseHandle.Call(handle)
			if r == 0 {
				return fmt.Errorf("释放互斥锁失败: %v", e)
			}
			return nil
		},
	}, nil
}
