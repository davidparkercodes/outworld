package detect

import (
	"context"
	"encoding/base64"
	"math/rand"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func cleanBlob(v string) bool {
	raw, ok := binaryBlob(v)
	return ok && blobClean(raw, NewInflateBudgetFor(testLimits))
}

func seededBytes(seed int64, n int) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(seed)).Read(b)
	return b
}

var testLimits = InflateLimits{Blob: 4 << 20, Request: 8 << 20}

func secretSpans(t *testing.T, text string) []entity.Span {
	t.Helper()
	spans, err := NewRegexDetectorWith(testLimits).Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	var out []entity.Span
	for _, s := range Resolve(spans) {
		if s.Type == entity.Secret {
			out = append(out, s)
		}
	}
	return out
}

func TestCleanBinaryBase64NotSecret(t *testing.T) {
	for _, kb := range []int{1, 3, 6, 12, 48} {
		for seed := int64(0); seed < 30; seed++ {
			v := base64.StdEncoding.EncodeToString(seededBytes(int64(kb)*1000+seed, kb<<10))
			for _, text := range []string{"see " + v, v, `{"data": "` + v + `"}`, "see " + strings.TrimRight(v, "="), "see " + strings.NewReplacer("+", "-", "/", "_").Replace(v)} {
				if got := secretSpans(t, text); len(got) > 0 {
					t.Errorf("%dKB seed %d: clean blob flagged by %s (%d chars)", kb, seed, got[0].Source, got[0].End-got[0].Start)
				}
			}
		}
	}
}

func shortSecretTokens(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, n := range []int{18, 48, 96, 300, 768, 1000, 1023} {
		for seed := int64(0); seed < 200; seed++ {
			v := base64.StdEncoding.EncodeToString(seededBytes(int64(n)*7919+seed, n))
			text := "see " + v
			if highEntropyValid(text, 4, len(text)) && !slugPath(v) {
				out = append(out, v)
				break
			}
		}
	}
	if len(out) != 7 {
		t.Fatalf("found %d short tokens the secret rule accepts", len(out))
	}
	return out
}

func TestShortBase64KeepsSecretRule(t *testing.T) {
	for _, v := range shortSecretTokens(t) {
		got := secretSpans(t, "see "+v)
		if len(got) != 1 || got[0].Text != v || got[0].Source != "regex:high_entropy" {
			t.Errorf("%d char token: want whole-token high_entropy, got %+v", len(v), got)
		}
	}
}

func TestBlobWithFindingsStaysSecret(t *testing.T) {
	for _, pii := range []string{" ssn 123-45-6789 ", " mail jane.doe@example.com ", " card 4111 1111 1111 1111 ", " call (804) 555-1234 now ", " token gh" + "p_Ab3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5 "} {
		for _, kb := range []int{1, 3, 12, 48} {
			raw := seededBytes(int64(kb), kb<<10)
			copy(raw[len(raw)/2:], pii)
			v := base64.StdEncoding.EncodeToString(raw)
			if cleanBlob(v) {
				t.Errorf("%q inside %dKB blob counted as clean", pii, kb)
			}
			whole := false
			for _, sp := range secretSpans(t, "see "+v) {
				whole = whole || sp.Text == v
			}
			if !whole {
				t.Errorf("%q inside %dKB blob: token not stripped whole", pii, kb)
			}
		}
	}
}

func TestSecretKeyedLongBase64StillStrips(t *testing.T) {
	v := base64.StdEncoding.EncodeToString(seededBytes(5, 3000))
	for _, text := range []string{`{"api_key": "` + v + `"}`, "client_secret=" + v, "Authorization: Bearer " + v} {
		if got := secretSpans(t, text); len(got) == 0 || got[0].Text != v {
			t.Errorf("%.40q: keyed long base64 not caught: %+v", text, got)
		}
	}
}

func TestVendorKeyGluedToBase64RunEdges(t *testing.T) {
	v := base64.StdEncoding.EncodeToString(seededBytes(5, 3000))
	for _, k := range []string{"AK" + "IAQWERTYUIOPASDFGH", "gh" + "p_Ab3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5", "AI" + "zaSyAb3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3", "sk" + "-ant-api03-Ab3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5", "np" + "m_Ab3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5"} {
		for name, text := range map[string]string{"front": "see " + k + strings.TrimRight(v, "="), "back": "see " + strings.TrimRight(v, "=") + k} {
			found := false
			for _, s := range secretSpans(t, text) {
				found = found || strings.Contains(s.Text, k)
			}
			if !found {
				t.Errorf("%s glued to the %s of a long base64 run not caught", k[:4], name)
			}
		}
	}
	if got := scanBlobVendorKeys("see gh"+"p_Ab3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5"+v[:600], ""); len(got) != 0 {
		t.Errorf("edge scan ran on a short run: %+v", got)
	}
	if got := scanBlobVendorKeys("see Qx"+v[:300]+"AK"+"IAQWERTYUIOPASDFGH"+v[300:600], ""); len(got) != 1 {
		t.Errorf("budgeted scan missed a key inside a short run: %+v", got)
	}
	mid := "see " + v[:1500] + "gh" + "p_Ab3dE5fG7hI9jK1lM3nO5pQ7rS9tU1vW3xY5" + v[1500:]
	if got := scanBlobVendorKeys(mid, ""); len(got) != 0 {
		t.Errorf("over-budget github shape scanned inside the run: %+v", got)
	}
}

func TestGluedVendorEdgeChanceRate(t *testing.T) {
	hits := 0
	for seed := int64(0); seed < 100; seed++ {
		v := base64.StdEncoding.EncodeToString(seededBytes(seed+500, 600<<10))
		hits += len(scanBlobVendorKeys("see "+v, ""))
	}
	if hits != 0 {
		t.Errorf("edge vendor scan fired on %d of 100 random 600 KB blobs", hits)
	}
}

func TestTextBase64KeepsTextExemption(t *testing.T) {
	text := strings.Repeat("The release notes cover the build plan for the whole team this quarter. ", 30)
	v := base64.StdEncoding.EncodeToString([]byte(text))
	if _, ok := binaryBlob(v); ok {
		t.Errorf("printable text treated as a binary blob")
	}
	if got := secretSpans(t, "see "+v); len(got) != 0 {
		t.Errorf("text base64 flagged: %+v", got)
	}
}

func TestInflateDefaultCaps(t *testing.T) {
	if InflateBlobMax != 1<<20 || InflateRequestMax != 2<<20 {
		t.Errorf("default caps %d %d", InflateBlobMax, InflateRequestMax)
	}
	if b := NewInflateBudget(); b.Blob != 1<<20 || b.Left != 2<<20 {
		t.Errorf("default budget %+v", b)
	}
}
