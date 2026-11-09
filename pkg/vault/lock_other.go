//go:build !unix

package vault

func lockPath(string) (func(), error) { return func() {}, nil }
