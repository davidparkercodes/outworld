package vault

import (
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func frameStarts(t *testing.T, raw []byte) []int {
	t.Helper()
	var out []int
	for at := len(fileMagic); at+4 <= len(raw); {
		out = append(out, at)
		at += 4 + int(binary.BigEndian.Uint32(raw[at:]))
	}
	return out
}

func TestReadErrorDoesNotSkipOtherWritersFrames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	a, err := OpenFile(path, testKey(6))
	if err != nil {
		t.Fatal(err)
	}
	b, err := OpenFile(path, testKey(6))
	if err != nil {
		t.Fatal(err)
	}
	for i, real := range []string{"Initech", "Globex", "Hooli"} {
		if _, err := b.Assign("org", real, func(int) string { return fmt.Sprintf("Org%d", i) }); err != nil {
			t.Fatal(err)
		}
	}
	raw, _ := os.ReadFile(path)
	starts := frameStarts(t, raw)
	if len(starts) != 5 {
		t.Fatalf("want header, format and 3 record frames, got %d", len(starts))
	}
	bad := starts[3] + 10
	raw[bad] ^= 0xff
	os.WriteFile(path, raw, 0o600)
	if _, err := a.Assign("org", "Umbrella", func(int) string { return "Org9" }); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("expected authentication error, got %v", err)
	}
	raw[bad] ^= 0xff
	os.WriteFile(path, raw, 0o600)
	got, err := a.Assign("org", "Umbrella", func(i int) string { return fmt.Sprintf("Org%d", i) })
	if err != nil || got != "Org3" {
		t.Fatalf("%q %v", got, err)
	}
	for i, real := range []string{"Initech", "Globex", "Hooli"} {
		if s, ok := a.Get("org", real); !ok || s != fmt.Sprintf("Org%d", i) {
			t.Fatalf("%s: %q %v", real, s, ok)
		}
	}
}

func TestZeroFilledLastFrameIsDroppedLikeATruncatedTail(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	v, _ := OpenFile(path, testKey(7))
	if _, err := v.Assign("org", "Initech", func(int) string { return "Globex" }); err != nil {
		t.Fatal(err)
	}
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write(append([]byte{0, 0, 0, 60}, make([]byte, 60)...))
	f.Close()
	v2, err := OpenFile(path, testKey(7))
	if err != nil {
		t.Fatalf("load after torn tail: %v", err)
	}
	if s, ok := v2.Get("org", "Initech"); !ok || s != "Globex" {
		t.Fatalf("%q %v", s, ok)
	}
	if _, err := v2.Assign("org", "Hooli", func(int) string { return "Umbrella" }); err != nil {
		t.Fatalf("update after torn tail: %v", err)
	}
	v3, err := OpenFile(path, testKey(7))
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := v3.Get("org", "Hooli"); !ok || s != "Umbrella" || v3.Len() != 2 {
		t.Fatalf("%q %v %d", s, ok, v3.Len())
	}
}

func TestZeroFilledFrameBeforeValidDataIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	v, _ := OpenFile(path, testKey(8))
	v.Assign("org", "Initech", func(int) string { return "Globex" })
	v.Assign("org", "Hooli", func(int) string { return "Umbrella" })
	raw, _ := os.ReadFile(path)
	starts := frameStarts(t, raw)
	s := starts[2]
	n := int(binary.BigEndian.Uint32(raw[s:]))
	for k := s + 4; k < s+4+n; k++ {
		raw[k] = 0
	}
	os.WriteFile(path, raw, 0o600)
	if _, err := OpenFile(path, testKey(8)); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("expected authentication error, got %v", err)
	}
}

func TestSyncFailureFramesArePickedUpWithoutDuplicate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	v, err := OpenFile(path, testKey(9))
	if err != nil {
		t.Fatal(err)
	}
	errSync := errors.New("injected sync failure")
	syncFile = func(*os.File) error { return errSync }
	_, err = v.Assign("org", "Initech", func(int) string { return "Globex" })
	syncFile = (*os.File).Sync
	if !errors.Is(err, errSync) {
		t.Fatalf("expected sync error, got %v", err)
	}
	if _, ok := v.Get("org", "Initech"); ok {
		t.Fatal("record visible after failed sync")
	}
	got, err := v.Assign("org", "Initech", func(int) string { return "Other" })
	if err != nil || got != "Globex" {
		t.Fatalf("want the written surrogate back, got %q %v", got, err)
	}
	_, recs, err := (&FileBackend{path: path, aead: v.backend.(*FileBackend).aead}).Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 2 || recs[0] != formatRecord || recs[1] != (Record{Kind: "org", Real: "Initech", Surrogate: "Globex"}) {
		t.Fatalf("records %+v", recs)
	}
}
