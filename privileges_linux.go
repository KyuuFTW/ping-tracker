//go:build linux

package main

import (
	"fmt"
	"os"
)

func checkPrivileges() {
	if os.Geteuid() != 0 {
		fmt.Fprintln(os.Stderr, "Warning: running without root. PID/app resolution may be incomplete.")
		fmt.Fprintln(os.Stderr, "Run with: sudo ping-tracker")
		fmt.Fprintln(os.Stderr, "")
	}
}
