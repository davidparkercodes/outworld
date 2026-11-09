package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

const specExample = `listen: 127.0.0.1:8787
vault:
  path: ./outworld.vault
  key_env: OUTWORLD_VAULT_KEY
upstreams:
  cloud: {base_url: https://api.example.com/v1, api_key_env: CLOUD_API_KEY}
  local: {base_url: http://127.0.0.1:11434/v1}
policy:
  default_route: masked
  masked_upstream: cloud
  local_upstream: local
  date_shift: true
  dictionary: ./people.csv
  entities:
    person: mask
    email: mask
    phone: mask
    address: mask
    date: shift
    org: keep
    location: keep
    secret: strip
    gov_id: strip
    financial: strip
  rules:
    - {match_any: ["diagnosis", "lawsuit"], route: local}
`

func TestSpecExampleParses(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "outworld.yaml")
	os.WriteFile(p, []byte(specExample), 0o600)
	c, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if c.Resolve(c.Policy.Dictionary) != filepath.Join(dir, "people.csv") {
		t.Fatalf("relative paths must resolve against the config dir: %s", c.Resolve(c.Policy.Dictionary))
	}
	pol, err := c.EnginePolicy()
	if err != nil || pol.Action(entity.Org) != entity.Keep || pol.Action(entity.Secret) != entity.Strip || !pol.DateShift {
		t.Fatalf("policy %+v %v", pol, err)
	}
	r := c.Policy.Rules[0]
	if !r.MatchText("new Diagnosis: flu") || r.MatchText("diagnostics") {
		t.Fatal("match_any should be case-insensitive and whole-word")
	}
	if !c.RouteHeaderAllowed() {
		t.Fatal("route header allowed by default")
	}
}

func TestValidation(t *testing.T) {
	bad := []string{
		"policy: {default_route: sideways}",
		"upstreams: {a: {base_url: http://x}}\npolicy: {masked_upstream: b}",
		"policy: {entities: {person: obliterate}}",
		"policy: {entities: {feelings: mask}}",
		"policy: {rules: [{route: local}]}",
		"policy: {rules: [{match_regex: ['(('], route: local}]}",
		"unknown_key: 1",
	}
	for _, b := range bad {
		if _, err := Parse([]byte(b)); err == nil {
			t.Errorf("expected error for %q", b)
		} else if !strings.Contains(err.Error(), "config") {
			t.Errorf("unhelpful error %v", err)
		}
	}
}
