package outworld

import (
	"encoding/base64"
	"fmt"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"math/rand"
	"strings"
	"testing"
)

func fidelityBlob(seed int64, n int) []byte {
	b := make([]byte, n)
	rand.New(rand.NewSource(seed)).Read(b)
	return b
}

func TestMaskKeepsCleanBinaryBase64(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("b64")
	seeds := int64(30)
	if scaletest.Race {
		seeds = 4
	}
	for _, kb := range []int{1, 3, 6, 12, 48} {
		for seed := int64(0); seed < seeds; seed++ {
			v := base64.StdEncoding.EncodeToString(fidelityBlob(int64(kb)*1000+seed, kb<<10))
			for _, in := range []string{"see " + v, `{"data": "` + v + `"}`} {
				out, err := s.Mask(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				if out != in {
					t.Errorf("%dKB seed %d: clean base64 changed: %.60q", kb, seed, out)
				}
			}
		}
	}
}

func TestMaskStillStripsBase64WithFindings(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("b64")
	var cases []string
	for i, pii := range []string{" ssn 123-45-6789 ", " mail jane.doe@example.com ", " card 4111 1111 1111 1111 "} {
		raw := fidelityBlob(int64(i), 3<<10)
		copy(raw[1500:], pii)
		cases = append(cases, base64.StdEncoding.EncodeToString(raw))
	}
	cases = append(cases, "AK"+"IAQWERTYUIOPASDFGH"+base64.RawStdEncoding.EncodeToString(fidelityBlob(10, 3000)))
	for i, v := range cases {
		out, err := s.Mask(ctx, "see "+v)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, v) {
			t.Errorf("case %d: base64 with a finding passed unchanged", i)
		}
	}
	keyed := fmt.Sprintf(`{"api_key": "%s"}`, cases[0])
	if out, _ := s.Mask(ctx, keyed); strings.Contains(out, cases[0]) {
		t.Errorf("secret-named key with long base64 passed unchanged")
	}
}
