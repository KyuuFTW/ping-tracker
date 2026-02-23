//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
)

func checkPrivileges() {
	if !isRunningAsAdmin() {
		fmt.Fprintln(os.Stderr, "Warning: running without Administrator. Some process names may not resolve.")
		fmt.Fprintln(os.Stderr, "Run as Administrator for full functionality.")
		fmt.Fprintln(os.Stderr, "")
	}
}

func isRunningAsAdmin() bool {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	isUserAnAdmin := shell32.NewProc("IsUserAnAdmin")

	ret, _, _ := isUserAnAdmin.Call()
	return ret != 0
}
