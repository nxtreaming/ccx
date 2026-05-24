package singleinstance

import "errors"

// ErrAlreadyRunning 表示检测到已有实例在运行。
var ErrAlreadyRunning = errors.New("CCX Desktop 已经在运行中")

// Lock 代表一个单实例锁。调用 Release() 释放。
type Lock struct {
	release func() error
}

// Release 释放锁资源。
func (l *Lock) Release() error {
	if l != nil && l.release != nil {
		return l.release()
	}
	return nil
}
