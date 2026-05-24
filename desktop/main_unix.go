//go:build !windows

package main

import (
	"fmt"
	"os"
)

// showErrorDialog 非 Windows 平台仅输出到 stderr。
func showErrorDialog(title, message string) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", title, message)
}

// recoverWithMessageBox 捕获未处理的 panic 并输出到 stderr。
func recoverWithMessageBox() {
	if r := recover(); r != nil {
		fmt.Fprintf(os.Stderr, "[CCX Desktop - 启动失败] 发生未处理的异常: %v\n", r)
		os.Exit(1)
	}
}

// singleInstanceArg 返回 Unix flock 锁文件所在目录。
func singleInstanceArg(dataDir string) string {
	return dataDir
}
