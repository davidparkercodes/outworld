package gateway

import (
	"strings"
	"testing"
)

func TestConfigYAML(t *testing.T) {
	c := Config{
		Listen: "127.0.0.1:9000", VaultPath: "/tmp/x/outworld.vault", UpstreamURL: "http://127.0.0.1:9001/v1",
		Dictionary: "/tmp/people.csv", DateShift: true,
		Entities: map[string]string{"person": "mask", "date": "shift", "org": "keep"},
	}
	want := `listen: 127.0.0.1:9000
vault:
  path: "/tmp/x/outworld.vault"
  key_env: OUTWORLD_VAULT_KEY
upstreams:
  cloud: {base_url: "http://127.0.0.1:9001/v1", api_key_env: CLOUD_API_KEY}
policy:
  default_route: masked
  masked_upstream: cloud
  date_shift: true
  dictionary: "/tmp/people.csv"
  entities: {date: shift, org: keep, person: mask}
memo: {enabled: false}
`
	if got := c.YAML(); got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
	c.Dictionary = ""
	if strings.Contains(c.YAML(), "dictionary") {
		t.Fatal("dictionary key should be omitted when empty")
	}
	c.API, c.Memo = APIAnthropic, true
	got := c.YAML()
	for _, want := range []string{
		`cloud: {base_url: "http://127.0.0.1:9001/v1", api_key_env: CLOUD_API_KEY, api: anthropic}`,
		"  anthropic: {masked_upstream: cloud}\n", "memo: {enabled: true}\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("anthropic config missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "\n  masked_upstream: cloud") {
		t.Fatalf("anthropic config must not set the top-level masked_upstream:\n%s", got)
	}
}
