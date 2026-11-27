package vault

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

var errInjected = errors.New("injected write failure")

type failingBackend struct {
	*FileBackend
	fail bool
}

func (b *failingBackend) Update(fn func([]Record) ([]Record, error)) error {
	if !b.fail {
		return b.FileBackend.Update(fn)
	}
	if err := b.FileBackend.Update(func(newer []Record) ([]Record, error) {
		_, err := fn(newer)
		return nil, err
	}); err != nil {
		return err
	}
	return errInjected
}

func TestFailedWriteLeavesNoRecordAndRetriesFormatMark(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	fb, err := NewFileBackend(path, testKey(3))
	if err != nil {
		t.Fatal(err)
	}
	b := &failingBackend{FileBackend: fb, fail: true}
	v, err := Open(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Assign("person.given", "Wilhelmina", func(i int) string { return fmt.Sprintf("First%d", i) }); !errors.Is(err, errInjected) {
		t.Fatalf("expected injected error, got %v", err)
	}
	if s, ok := v.Get("person.given", "Wilhelmina"); ok {
		t.Fatalf("unpersisted record visible: %q", s)
	}
	if v.IsSurrogate("First0") || v.IsReal("Wilhelmina") || v.Len() != 0 {
		t.Fatal("unpersisted record left in the index")
	}
	if v.marked {
		t.Fatal("format mark set without being written")
	}

	b.fail = false
	s1, err := v.Assign("person.given", "Wilhelmina", func(i int) string { return fmt.Sprintf("Second%d", i) })
	if err != nil || s1 != "Second0" {
		t.Fatalf("%q %v", s1, err)
	}
	if again, _ := v.Assign("person.given", "Wilhelmina", func(int) string { return "Other" }); again != s1 {
		t.Fatalf("unstable surrogate %q", again)
	}
	if v.Legacy("person.given", "Wilhelmina") {
		t.Fatal("new record read as legacy")
	}

	_, recs, err := (&FileBackend{path: path, aead: fb.aead}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || recs[0] != formatRecord || recs[1].Surrogate != "Second0" {
		t.Fatalf("persisted records %+v", recs)
	}

	v2, err := OpenFile(path, testKey(3))
	if err != nil {
		t.Fatal(err)
	}
	if got, ok := v2.Get("person.given", "Wilhelmina"); !ok || got != s1 {
		t.Fatalf("reopen: %q %v", got, ok)
	}
	if v2.Legacy("person.given", "Wilhelmina") {
		t.Fatal("record read back as legacy")
	}
}

func TestFailedWriteDoesNotBlockLaterAssignsInOtherVault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	fb, err := NewFileBackend(path, testKey(4))
	if err != nil {
		t.Fatal(err)
	}
	b := &failingBackend{FileBackend: fb, fail: true}
	v, err := Open(b)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Assign("email", "a@corp.test", func(int) string { return "x@example.com" }); err == nil {
		t.Fatal("expected error")
	}
	other, err := OpenFile(path, testKey(4))
	if err != nil {
		t.Fatal(err)
	}
	s, err := other.Assign("email", "b@corp.test", func(int) string { return "x@example.com" })
	if err != nil || s != "x@example.com" {
		t.Fatalf("%q %v", s, err)
	}
	b.fail = false
	got, err := v.Assign("email", "a@corp.test", func(i int) string { return fmt.Sprintf("x%d@example.com", i) })
	if err != nil || got == "x@example.com" {
		t.Fatalf("collided with other writer: %q %v", got, err)
	}
}

func TestCreateLeavesNoTempFilesAndIsPrivate(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub")
	path := filepath.Join(dir, "v.vault")
	v, err := OpenFile(path, testKey(5))
	if err != nil {
		t.Fatal(err)
	}
	v.Close()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "v.vault" && e.Name() != "v.vault.lock" {
			t.Fatalf("stray file %q", e.Name())
		}
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perms %v", st.Mode().Perm())
	}
	if _, err := OpenFile(path, testKey(5)); err != nil {
		t.Fatal(err)
	}
}
