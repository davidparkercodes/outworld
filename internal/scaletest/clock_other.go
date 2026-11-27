//go:build !unix

package scaletest

import "time"

const CPUClock = false

func now() time.Duration { return wallNow() }
