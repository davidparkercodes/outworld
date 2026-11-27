package detect

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

func ipSpans(t *testing.T, text string) []string {
	t.Helper()
	spans, err := NewRegexDetector().Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, s := range spans {
		if s.Type == entity.IP {
			out = append(out, s.Text)
		}
	}
	return out
}

func TestNonPersonalIPsAreLeftAlone(t *testing.T) {
	cases := map[string][]string{
		"loopback":    {"127.0.0.1", "127.1.2.3", "127.255.255.254"},
		"unspecified": {"0.0.0.0", "::"},
		"broadcast":   {"255.255.255.255"},
		"netmask":     {"255.255.255.0", "255.255.0.0", "255.0.0.0", "255.255.255.128", "255.255.252.0"},
		"link-local":  {"169.254.0.1", "169.254.169.254", "fe80::1", "fe80::1ff:fe23:4567:890a", "febf::1"},
		"ipv6 lo":     {"::1"},
		"mapped":      {"::ffff:127.0.0.1", "::ffff:7f00:1", "::ffff:0.0.0.0", "::ffff:255.255.255.0", "::ffff:169.254.1.1", "::ffff:8.8.8.8", "::ffff:0808:0808"},
		"resolver":    {"8.8.8.8", "8.8.4.4", "1.1.1.1", "1.0.0.1", "9.9.9.9", "2001:4860:4860::8888", "2001:4860:4860::8844", "2606:4700:4700::1111", "2606:4700:4700::1001", "2620:fe::fe", "2620:fe::9"},
	}
	for class, ips := range cases {
		for _, ip := range ips {
			host := ip
			if strings.Contains(ip, ":") {
				host = "[" + ip + "]"
			}
			for _, text := range []string{"host " + ip + " here", "curl http://" + host + ":8080/healthz", host + ":443"} {
				if got := ipSpans(t, text); len(got) != 0 {
					t.Errorf("%s %q: want no ip span, got %v", class, text, got)
				}
			}
		}
	}
}

func TestPersonalIPsStayMasked(t *testing.T) {
	for _, ip := range []string{
		"10.1.2.3", "192.168.1.20", "172.16.5.4", "34.117.59.81", "8.8.8.9", "1.1.1.2", "128.0.0.1", "255.255.255.1", "255.0.255.0", "169.253.1.1", "126.0.0.1",
		"2a00:1450:4001:82b::200e", "2001:4860:4860::8889", "fec0::1",
		"192.0.2.10", "198.51.100.7", "203.0.113.200", "2001:db8::1:2",
	} {
		text := "server " + ip + " is up"
		if got := ipSpans(t, text); len(got) != 1 || got[0] != ip {
			t.Errorf("%q: want ip span %q, got %v", text, ip, got)
		}
	}
	if got := ipSpans(t, "curl http://192.168.1.20:8080/healthz"); len(got) != 1 || got[0] != "192.168.1.20" {
		t.Errorf("private ip inside a url: got %v", got)
	}
}

func TestNonPersonalIPCheck(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "0.0.0.0", "::1", "::", "255.255.255.0", "8.8.8.8", "fe80::abcd", "::ffff:127.0.0.1", "::ffff:7f00:1"} {
		if !nonPersonalIP(s) {
			t.Errorf("%q should be non-personal", s)
		}
	}
	for _, s := range []string{"10.0.0.1", "::ffff:10.0.0.1", "::ffff:c0a8:0114", "not an ip", "192.0.2.1", "2001:db8::5"} {
		if nonPersonalIP(s) {
			t.Errorf("%q should stay personal", s)
		}
	}
}

func TestIPv4EmbeddedIPv6IsOneWholeSpan(t *testing.T) {
	g := surrogate.NewGenerator([]byte("embedded"))
	for _, ip := range []string{"::ffff:192.168.1.20", "::FFFF:10.1.2.3", "::192.168.1.20", "64:ff9b::34.117.59.81", "0:0:0:0:0:ffff:192.168.1.20", "2001:db8:abcd::10.0.0.1", "::ffff:c0a8:0114", "::FFFF:C0A8:114"} {
		text := "mapped " + ip + " here"
		spans := detectAll(t, text)
		if len(spans) != 1 || spans[0].Type != entity.IP || spans[0].Text != ip {
			t.Errorf("%q: want one ip span %q, got %+v", text, ip, spans)
			continue
		}
		sur := g.IP(ip, 0)
		masked := text[:spans[0].Start] + sur + text[spans[0].End:]
		inner := ip[strings.LastIndexByte(ip, ':')+1:]
		if strings.Contains(masked, inner) && strings.Contains(inner, ".") || strings.Contains(masked, ip) {
			t.Errorf("%q: masked %q keeps the real ipv4 part", text, masked)
		}
		again := detectAll(t, masked)
		if len(again) != 1 || again[0].Text != sur {
			t.Errorf("%q: surrogate %q is not re-detected as one whole ip, got %+v", text, sur, again)
		}
	}
}

func TestIPv4EmbeddedIPv6RoundTripsThroughTheVault(t *testing.T) {
	v, err := vault.Open(vault.NewMemoryBackend(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	g := surrogate.NewGenerator(v.Secret())
	text := "peer ::ffff:192.168.1.20 and 10.1.2.3 and 64:ff9b::34.117.59.81 and ::ffff:c0a8:0115 up, lo ::ffff:127.0.0.1"
	spans := detectAll(t, text)
	var b strings.Builder
	last := 0
	for _, sp := range spans {
		sur, err := v.Assign("ip", sp.Text, func(a int) string { return g.IP(sp.Text, a) })
		if err != nil {
			t.Fatal(err)
		}
		b.WriteString(text[last:sp.Start])
		b.WriteString(sur)
		last = sp.End
	}
	b.WriteString(text[last:])
	masked := b.String()
	if !strings.Contains(text, "::ffff:127.0.0.1") {
		t.Fatal("exempt mapped loopback must stay")
	}
	for _, real := range []string{"192.168.1.20", "10.1.2.3", "34.117.59.81", "c0a8:0115"} {
		if strings.Contains(masked, real) {
			t.Fatalf("masked %q keeps %q", masked, real)
		}
	}
	var out strings.Builder
	last = 0
	for _, h := range v.FindSurrogates(masked) {
		out.WriteString(masked[last:h.Start])
		out.WriteString(h.Record.Real)
		last = h.End
	}
	out.WriteString(masked[last:])
	if out.String() != text {
		t.Fatalf("round trip %q -> %q -> %q", text, masked, out.String())
	}
}
