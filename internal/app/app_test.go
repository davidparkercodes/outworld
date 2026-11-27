package app

import (
	"bytes"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

type closeCounter struct {
	*vault.FileBackend
	closes *int
}

func (c closeCounter) Close() error {
	*c.closes++
	return c.FileBackend.Close()
}

func TestBuildEngineClosesTheVaultOnEveryLaterError(t *testing.T) {
	key, _ := vault.GenerateKey()
	t.Setenv("OUTWORLD_TEST_VAULT_KEY", key)
	closes := 0
	openVault = func(path string, k []byte) (*vault.Vault, error) {
		b, err := vault.NewFileBackend(path, k)
		if err != nil {
			return nil, err
		}
		return vault.Open(closeCounter{FileBackend: b, closes: &closes})
	}
	defer func() { openVault = vault.OpenFile }()
	dir := t.TempDir()
	base := "vault: {path: ./v.vault, key_env: OUTWORLD_TEST_VAULT_KEY}\n"
	cases := map[string]string{
		"missing dictionary": "policy: {dictionary: ./missing.csv}\n",
		"bad ner label":      "ner: {url: \"http://127.0.0.1:1\", labels: {person: not_a_type}}\n",
	}
	for name, extra := range cases {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "_")+".yaml")
		os.WriteFile(path, []byte(base+extra), 0o600)
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatalf("%s: config: %v", name, err)
		}
		before := closes
		if _, err := BuildEngine(cfg); err == nil {
			t.Fatalf("%s: expected an error", name)
		}
		if closes != before+1 {
			t.Errorf("%s: vault closed %d times, want 1", name, closes-before)
		}
	}
	path := filepath.Join(dir, "ok.yaml")
	os.WriteFile(path, []byte(base), 0o600)
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	before := closes
	if _, err := BuildEngine(cfg); err != nil {
		t.Fatal(err)
	}
	if closes != before {
		t.Errorf("a successful build must keep the vault open")
	}
}

func TestWarnDroppedFormsLogsOnlyTheCount(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	WarnDroppedForms(log, 0)
	if buf.Len() != 0 {
		t.Fatalf("no drops must log nothing: %q", buf.String())
	}
	WarnDroppedForms(log, 3)
	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 1 || !strings.Contains(lines[0], "level=WARN") || !strings.Contains(lines[0], "count=3") {
		t.Fatalf("want one warn line with the count: %q", buf.String())
	}
}
