//go:build !unix

package gateway

import "time"

var cpuStart = time.Now()

func processCPU() time.Duration { return time.Since(cpuStart) }
