//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	moduser32       = syscall.NewLazyDLL("user32.dll")
	procMessageBoxW = moduser32.NewProc("MessageBoxW")
)

const (
	mbOK            = 0x00000000
	mbIconError     = 0x00000010
	mbTopmost       = 0x00040000
	mbSetForeground = 0x00010000
)

// showErrorDialog 在无控制台的 windowsgui 模式下弹出错误对话框。
func showErrorDialog(title, message string) {
	titlePtr, _ := syscall.UTF16PtrFromString(title)
	msgPtr, _ := syscall.UTF16PtrFromString(message)
	procMessageBoxW.Call(
		0,
		uintptr(unsafe.Pointer(msgPtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		mbOK|mbIconError|mbTopmost|mbSetForeground,
	)
}

// recoverWithMessageBox 捕获未处理的 panic 并弹窗提示后退出。
func recoverWithMessageBox() {
	if r := recover(); r != nil {
		showErrorDialog("CCX Desktop - 启动失败", fmt.Sprintf("发生未处理的异常: %v\n\n请检查日志或联系开发者。", r))
		os.Exit(1)
	}
}

// singleInstanceArg 返回 Windows named mutex 名称（忽略 dataDir）。
func singleInstanceArg(_ string) string {
	return "Global\\CCXDesktopMutex"
}
