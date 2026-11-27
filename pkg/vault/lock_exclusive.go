package vault

import (
	"errors"
	"fmt"
	"os"
)

var ErrVaultInUse = errors.New("vault: in use by another process")

func acquireExclusive(path string) (func() error, error) {
	lock := path + ".lock"
	fh, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil, fmt.Errorf("%w: lock file %s exists; on this platform a vault can be used by one process at a time; the lock file may be left over from a crash: delete it only once no other Outworld process is running", ErrVaultInUse, lock)
	}
	if err != nil {
		return nil, fmt.Errorf("vault: create lock file %s: %w", lock, err)
	}
	fh.Close()
	return func() error {
		if err := os.Remove(lock); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("vault: remove lock file %s: %w", lock, err)
		}
		return nil
	}, nil
}
