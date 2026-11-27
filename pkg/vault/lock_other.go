//go:build !(linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package vault

var singleProcess = true

func lockPath(string) (func(), error) { return func() {}, nil }

func syncDir(string) error { return nil }
