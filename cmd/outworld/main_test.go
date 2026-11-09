package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/outworld"
)

func setup(t *testing.T) string {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "people.csv"), []byte("value,type,aliases\nJohn Smith,person,\n"), 0o600)
	cfg := filepath.Join(dir, "outworld.yaml")
	os.WriteFile(cfg, []byte("vault: {path: ./v.vault}\npolicy: {dictionary: ./people.csv}\n"), 0o600)
	t.Setenv("OUTWORLD_VAULT_KEY", strings.Repeat("cd", 32))
	return cfg
}

func TestMaskUnmaskCLI(t *testing.T) {
	cfg := setup(t)
	in := "John Smith (john@corp.test) met on 2024-03-15. OTP code 991234."
	var masked bytes.Buffer
	if err := maskCmd([]string{"--config", cfg}, strings.NewReader(in), &masked); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(masked.String(), "John") || strings.Contains(masked.String(), "2024-03-15") || strings.Contains(masked.String(), "991234") {
		t.Fatalf("masked: %s", masked.String())
	}
	var back bytes.Buffer
	if err := unmaskCmd([]string{"--config", cfg}, strings.NewReader(masked.String()), &back); err != nil {
		t.Fatal(err)
	}
	if back.String() != "John Smith (john@corp.test) met on 2024-03-15. OTP code [redacted code]." {
		t.Fatalf("unmasked: %s", back.String())
	}
	var other bytes.Buffer
	unmaskCmd([]string{"--config", cfg, "--scope", "elsewhere"}, strings.NewReader(masked.String()), &other)
	if other.String() == back.String() {
		t.Fatal("a different scope must use a different date offset")
	}
}

func TestEnvConfigAndMissingKey(t *testing.T) {
	cfg := setup(t)
	t.Setenv("OUTWORLD_CONFIG", cfg)
	var out bytes.Buffer
	if err := maskCmd(nil, strings.NewReader("hi John"), &out); err != nil || strings.Contains(out.String(), "John") {
		t.Fatalf("%v %q", err, out.String())
	}
	t.Setenv("OUTWORLD_VAULT_KEY", "")
	if err := maskCmd(nil, strings.NewReader("hi"), &out); err == nil || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("expected refusal without key, got %v", err)
	}
	if !errors.Is(&outworld.LeakError{}, outworld.ErrLeak) {
		t.Fatal("leak error must match ErrLeak for exit code 3")
	}
}
