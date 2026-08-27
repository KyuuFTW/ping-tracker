//go:build !windows && !linux

package tracker

import "time"

func measureICMP(addr string, count int, timeout time.Duration) (time.Duration, float64) {
	return 0, 100.0
}
