package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func vaultWithOneRecord(t *testing.T, key byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "v.vault")
	v, err := OpenFile(path, testKey(key))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Assign("org", "Initech", func(int) string { return "Globex" }); err != nil {
		t.Fatal(err)
	}
	v.Close()
	return path
}

func appendBytes(t *testing.T, path string, b []byte) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	f.Write(b)
	f.Close()
}

func TestTornZeroTailsAreDropped(t *testing.T) {
	frame := func(n int) []byte { return append([]byte{0, 0, 0, byte(n)}, make([]byte, n)...) }
	tails := map[string][]byte{
		"76 zero bytes":         make([]byte, 76),
		"4096 zero bytes":       make([]byte, 4096),
		"two zero-body frames":  append(frame(40), frame(40)...),
		"zero-body frame":       frame(60),
		"three zero bytes only": make([]byte, 3),
	}
	for name, tail := range tails {
		path := vaultWithOneRecord(t, 10)
		before, _ := os.Stat(path)
		appendBytes(t, path, tail)
		v, err := OpenFile(path, testKey(10))
		if err != nil {
			t.Errorf("%s: open: %v", name, err)
			continue
		}
		if s, ok := v.Get("org", "Initech"); !ok || s != "Globex" {
			t.Errorf("%s: lost record", name)
		}
		if _, err := v.Assign("org", "Hooli", func(int) string { return "Umbrella" }); err != nil {
			t.Errorf("%s: assign: %v", name, err)
			continue
		}
		v.Close()
		v2, err := OpenFile(path, testKey(10))
		if err != nil || v2.Len() != 2 {
			t.Errorf("%s: reopen: %v", name, err)
			continue
		}
		v2.Close()
		after, _ := os.Stat(path)
		if after.Size() <= before.Size() || after.Size() >= before.Size()+int64(len(tail))+200 {
			t.Errorf("%s: zero tail not truncated before the append: %d -> %d", name, before.Size(), after.Size())
		}
	}
}

func TestMostlyZeroTailWithDataIsAnError(t *testing.T) {
	path := vaultWithOneRecord(t, 11)
	body := make([]byte, 60)
	body[30] = 1
	appendBytes(t, path, append(append([]byte{0, 0, 0, 60}, body...), 0, 0, 0, 0))
	if _, err := OpenFile(path, testKey(11)); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("expected authentication error, got %v", err)
	}
	path = vaultWithOneRecord(t, 11)
	appendBytes(t, path, append(make([]byte, 8), 7))
	if _, err := OpenFile(path, testKey(11)); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("zeros followed by data: expected authentication error, got %v", err)
	}
}

func TestFailedSyncIsRetriedOnTheNextUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	v, err := OpenFile(path, testKey(12))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { syncFile = (*os.File).Sync }()
	syncFile = func(*os.File) error { return errors.New("injected sync failure") }
	if _, err := v.Assign("org", "Initech", func(int) string { return "Globex" }); err == nil {
		t.Fatal("expected sync error")
	}
	calls := 0
	syncFile = func(fh *os.File) error { calls++; return fh.Sync() }
	got, err := v.Assign("org", "Initech", func(int) string { return "Other" })
	if err != nil || got != "Globex" {
		t.Fatalf("%q %v", got, err)
	}
	if calls != 1 {
		t.Fatalf("want one fsync of the unsynced frames, got %d", calls)
	}
	if _, err := v.Assign("org", "Hooli", func(int) string { return "Umbrella" }); err != nil || calls != 2 {
		t.Fatalf("next write: %v, %d syncs", err, calls)
	}
	other, _ := OpenFile(path, testKey(12))
	other.Assign("org", "Acme", func(int) string { return "Vandelay" })
	calls = 0
	if _, err := v.Assign("org", "Acme", func(int) string { return "x" }); err != nil || calls != 0 {
		t.Fatalf("frames from another writer need no extra fsync: %v, %d", err, calls)
	}
}

func TestExclusiveLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	release, err := acquireExclusive(path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = acquireExclusive(path)
	if !errors.Is(err, ErrVaultInUse) {
		t.Fatalf("want ErrVaultInUse, got %v", err)
	}
	msg := err.Error()
	for _, want := range []string{path + ".lock", "crash", "delete it only once no other Outworld process is running", "one process"} {
		if !strings.Contains(msg, want) {
			t.Errorf("error %q lacks %q", msg, want)
		}
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("lock file not removed: %v", err)
	}
	again, err := acquireExclusive(path)
	if err != nil {
		t.Fatal(err)
	}
	again()
}

func TestSingleProcessVaultHoldsLockUntilClose(t *testing.T) {
	old := singleProcess
	singleProcess = true
	defer func() { singleProcess = old }()
	path := filepath.Join(t.TempDir(), "sub", "v.vault")
	v, err := OpenFile(path, testKey(13))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".lock"); err != nil {
		t.Fatalf("lock file missing while open: %v", err)
	}
	if _, err := OpenFile(path, testKey(13)); !errors.Is(err, ErrVaultInUse) || !strings.Contains(err.Error(), path+".lock") {
		t.Fatalf("second open: %v", err)
	}
	if _, err := v.Assign("org", "Initech", func(int) string { return "Globex" }); err != nil {
		t.Fatal(err)
	}
	if err := v.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("Close did not remove the lock file: %v", err)
	}
	v2, err := OpenFile(path, testKey(13))
	if err != nil {
		t.Fatal(err)
	}
	if s, _ := v2.Get("org", "Initech"); s != "Globex" {
		t.Fatal(fmt.Sprintf("lost record: %q", s))
	}
	v2.Close()
	if _, err := OpenFile(path, testKey(99)); err == nil {
		t.Fatal("wrong key opened")
	}
	if _, err := os.Stat(path + ".lock"); !os.IsNotExist(err) {
		t.Fatalf("failed open left the lock file: %v", err)
	}
}
