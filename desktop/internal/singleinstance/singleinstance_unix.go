//go:build !windows

package singleinstance

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Acquire 在 dataDir 下创建 .lock 文件并加排他锁。若已有实例持有锁则返回 ErrAlreadyRunning。
func Acquire(dataDir string) (*Lock, error) {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	lockPath := filepath.Join(dataDir, ".lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("打开锁文件失败: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, ErrAlreadyRunning
	}
	// 写入 PID 方便调试
	_ = f.Truncate(0)
	_, _ = fmt.Fprintf(f, "%d\n", os.Getpid())
	return &Lock{
		release: func() error {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			return f.Close()
		},
	}, nil
}
