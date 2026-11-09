package vault

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func testKey(b byte) []byte { return bytes.Repeat([]byte{b}, 32) }

func TestFileVaultPersistsEncrypted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	v, err := OpenFile(path, testKey(1))
	if err != nil {
		t.Fatal(err)
	}
	s1, err := v.Assign("person.given", "Wilhelmina", func(i int) string { return fmt.Sprintf("Cand%d", i) })
	if err != nil || s1 != "Cand0" {
		t.Fatalf("%q %v", s1, err)
	}
	s2, _ := v.Assign("email", "secret.person@corp.test", func(i int) string { return "a@example.com" })
	again, _ := v.Assign("person.given", "wilhelmina", func(i int) string { return "Other" })
	if again != s1 {
		t.Fatalf("case-insensitive lookup failed: %q", again)
	}
	secret := v.Secret()
	v.Close()

	raw, _ := os.ReadFile(path)
	for _, needle := range []string{"Wilhelmina", "secret.person", "Cand0", "example.com"} {
		if bytes.Contains(raw, []byte(needle)) {
			t.Fatalf("vault file contains plaintext %q", needle)
		}
	}
	st, _ := os.Stat(path)
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("vault perms %v", st.Mode().Perm())
	}

	v2, err := OpenFile(path, testKey(1))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(v2.Secret(), secret) {
		t.Fatal("secret not persisted")
	}
	if got, ok := v2.Get("email", "SECRET.PERSON@corp.test"); !ok || got != s2 {
		t.Fatalf("reload lost record: %q %v", got, ok)
	}
	if r, ok := v2.Reverse("cand0"); !ok || r.Real != "Wilhelmina" {
		t.Fatalf("reverse: %+v %v", r, ok)
	}
	if _, err := OpenFile(path, testKey(2)); !errors.Is(err, ErrWrongKey) {
		t.Fatalf("expected wrong key error, got %v", err)
	}
}

func TestFileVaultDetectsTampering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	v, _ := OpenFile(path, testKey(3))
	v.Assign("org", "Initech", func(int) string { return "Globex" })
	raw, _ := os.ReadFile(path)
	raw[len(raw)-3] ^= 0xff
	os.WriteFile(path, raw, 0o600)
	if _, err := OpenFile(path, testKey(3)); err == nil || !strings.Contains(err.Error(), "authentication") {
		t.Fatalf("expected authentication failure, got %v", err)
	}
}

func TestFileVaultTruncatedTailRecovers(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	v, _ := OpenFile(path, testKey(4))
	v.Assign("org", "Initech", func(int) string { return "Globex" })
	f, _ := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o600)
	f.Write([]byte{0, 0, 0, 90, 1, 2, 3})
	f.Close()
	v2, err := OpenFile(path, testKey(4))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v2.Assign("org", "Umbrella", func(int) string { return "Hooli" }); err != nil {
		t.Fatal(err)
	}
	v3, err := OpenFile(path, testKey(4))
	if err != nil || v3.Len() != 2 {
		t.Fatalf("len=%d err=%v", v3.Len(), err)
	}
}

func TestTwoWritersStayConsistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v.vault")
	a, _ := OpenFile(path, testKey(5))
	b, _ := OpenFile(path, testKey(5))
	sa, _ := a.Assign("person.given", "Alpha", func(i int) string { return fmt.Sprintf("N%d", i) })
	sb, _ := b.Assign("person.given", "Beta", func(i int) string { return fmt.Sprintf("N%d", i) })
	if sa == sb {
		t.Fatalf("two writers produced the same surrogate %q", sa)
	}
	sa2, _ := b.Assign("person.given", "Alpha", func(i int) string { return "Z" })
	if sa2 != sa {
		t.Fatalf("writer b did not see writer a's record: %q vs %q", sa2, sa)
	}
}

func TestSurrogateNeverEqualsReal(t *testing.T) {
	v, _ := Open(NewMemoryBackend(testKey(6)))
	v.Reserve("Dalia")
	v.Assign("person.given", "Felix", func(int) string { return "Otto" })
	got, _ := v.Assign("person.given", "Ramona", func(i int) string {
		return []string{"Ramona", "Dalia", "Otto", "Felix", "Tamsin"}[i]
	})
	if got != "Tamsin" {
		t.Fatalf("got %q", got)
	}
	if _, err := v.Assign("x", "y", func(int) string { return "Otto" }); !errors.Is(err, ErrExhausted) {
		t.Fatalf("expected exhaustion, got %v", err)
	}
}

func TestConcurrentAssign(t *testing.T) {
	v, _ := Open(NewMemoryBackend(testKey(7)))
	var wg sync.WaitGroup
	results := make([]string, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			results[i], _ = v.Assign("person.given", fmt.Sprintf("Real%d", i%10), func(a int) string { return fmt.Sprintf("S%d", a) })
		}(i)
	}
	wg.Wait()
	seen := map[string]string{}
	for i, s := range results {
		real := fmt.Sprintf("Real%d", i%10)
		if prev, ok := seen[s]; ok && prev != real {
			t.Fatalf("surrogate %q shared by %q and %q", s, prev, real)
		}
		seen[s] = real
	}
	if len(seen) != 10 {
		t.Fatalf("expected 10 surrogates, got %d", len(seen))
	}
}

func TestFindRealsAndSurrogates(t *testing.T) {
	v, _ := Open(NewMemoryBackend(testKey(8)))
	v.Assign("person.given", "Bartholomew", func(int) string { return "Isolde" })
	v.Assign("email", "bart@corp.test", func(int) string { return "isolde.x@example.org" })
	hits := v.FindReals("Ask BARTHOLOMEW or bart@corp.test.")
	if len(hits) != 2 {
		t.Fatalf("reals: %+v", hits)
	}
	hits = v.FindSurrogates("isolde.x@example.org wrote to Isolde's friend")
	if len(hits) != 2 || hits[0].Record.Kind != "email" || hits[1].Record.Real != "Bartholomew" {
		t.Fatalf("surrogates: %+v", hits)
	}
}

func TestKeys(t *testing.T) {
	k, _ := GenerateKey()
	if b, err := ParseKey(k); err != nil || len(b) != 32 {
		t.Fatal(err)
	}
	if _, err := ParseKey("dG9vc2hvcnQ="); err == nil {
		t.Fatal("short key accepted")
	}
	t.Setenv("OW_TEST_KEY", "")
	if _, err := LoadKey("OW_TEST_KEY", ""); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("expected refusal, got %v", err)
	}
	t.Setenv("OW_TEST_KEY", k)
	if _, err := LoadKey("OW_TEST_KEY", ""); err != nil {
		t.Fatal(err)
	}
	kf := filepath.Join(t.TempDir(), "key")
	os.WriteFile(kf, []byte(k+"\n"), 0o600)
	if _, err := LoadKey("", kf); err != nil {
		t.Fatal(err)
	}
}
