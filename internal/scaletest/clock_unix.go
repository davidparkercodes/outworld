//go:build unix

package scaletest

import (
	"syscall"
	"time"
)

const CPUClock = true

func now() time.Duration {
	var u syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &u); err != nil {
		return wallNow()
	}
	return time.Duration(u.Utime.Nano() + u.Stime.Nano())
}
