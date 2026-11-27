package gateway

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/davidparkercodes/outworld/internal/scaletest"
)

func repeatShape(s string) func(*rand.Rand, int) []byte { return listShape("", s) }

func listShape(head, s string) func(*rand.Rand, int) []byte {
	return func(_ *rand.Rand, n int) []byte { return []byte((head + strings.Repeat(s, n/len(s)+2))[:n]) }
}

func adversarialShapes() map[string]func(*rand.Rand, int) []byte {
	k := map[string]func(*rand.Rand, int) []byte{
		"yaml a list":      repeatShape("a:\n- "),
		"yaml a list a":    repeatShape("a:\n- a:\n- "),
		"keys list":        listShape("keys:\n", "- item\n"),
		"key every 3":      repeatShape("token:\n- abc\n- def\n"),
		"secrets short":    listShape("secrets:\n", "- ab\n"),
		"k bracket":        repeatShape("k: ["),
		"json nest":        repeatShape(`"a":[`),
		"key equals":       repeatShape("key="),
		"x colon lines":    repeatShape("x: \n"),
		"secret 100k list": listShape("api_keys:\n", "- v\n"),
		"nested arrays": func(r *rand.Rand, n int) []byte {
			return repeatShape(`{"api_key": [["abc", "Hq7vR2mZ9xK4pL8wT3nY"]]},`)(r, n)
		},
		"dash lines":    repeatShape("- - - - - - - - \n"),
		"pem begin":     repeatShape("-----BEGIN "),
		"pem begin end": repeatShape("-----BEGIN X-----END X-----"),
		"pem end":       repeatShape("-----END "),
		"cbor small items": func(r *rand.Rand, n int) []byte {
			b := []byte{}
			for len(b) < n {
				b = append(b, 0x62, byte('a'+r.Intn(26)), byte('a'+r.Intn(26)))
			}
			return b[:n]
		},
		"base64 tokens": func(r *rand.Rand, n int) []byte {
			return repeatShape(base64.StdEncoding.EncodeToString(uniformBytes(r, 30))+" ")(r, n)
		},
	}
	frags := []string{":", ": ", ":\n", "- ", "=", "[", "{", `"`, `",`, "\n", "a", "A1"}
	for _, a := range frags {
		k["frag "+fmt.Sprintf("%q", a)] = repeatShape(a)
		for _, b := range frags {
			if a < b {
				k["frag "+fmt.Sprintf("%q%q", a, b)] = repeatShape(a + b)
			}
		}
	}
	return k
}

const (
	adversarialEnglishLimit = 3
	growthFloor             = 100 * time.Millisecond
)

func TestAdversarialBlobShapes(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	cost := func(gen func(*rand.Rand, int) []byte, size int) time.Duration {
		v := base64.StdEncoding.EncodeToString(gen(rand.New(rand.NewSource(7)), size))
		return bestCPU(2, func() { corpusFindingsByBlob(t, srv, []string{v}, []int{0}) })
	}
	small, large := 8<<10, 32<<10
	english := cost(func(r *rand.Rand, n int) []byte { return corpusEnglish(r, n) }, large)
	random := cost(uniformBytes, large)
	kinds := adversarialShapes()
	var names []string
	for n := range kinds {
		names = append(names, n)
	}
	sort.Strings(names)
	worstGrowth, worstEnglish := 0.0, 0.0
	for i, name := range names {
		if scaletest.Race && i%8 != 0 {
			continue
		}
		s, l := cost(kinds[name], small), cost(kinds[name], large)
		growth := float64(l) / float64(max(s, time.Millisecond))
		vsEnglish := float64(l) / float64(english)
		worstGrowth, worstEnglish = max(worstGrowth, growth), max(worstEnglish, vsEnglish)
		t.Logf("%-28s %v then %v (%.1fx), %.1fx english, %.1fx random", name, s.Round(time.Millisecond), l.Round(time.Millisecond), growth, vsEnglish, float64(l)/float64(random))
		if scaletest.Race {
			continue
		}
		if l > 8*max(s, growthFloor) {
			t.Errorf("%s: 4x the input took %v after %v", name, l, s)
		}
		if vsEnglish > adversarialEnglishLimit {
			t.Errorf("%s: %v is %.1fx plain English of the same size (%v)", name, l, vsEnglish, english)
		}
	}
	t.Logf("english %v, random %v, worst growth %.1fx, worst vs english %.1fx", english, random, worstGrowth, worstEnglish)
}

func yamlAndPEMShapes() map[string]func(*rand.Rand, int) []byte {
	lines := func(head, unit string) func(*rand.Rand, int) []byte {
		return func(_ *rand.Rand, n int) []byte { return []byte((head + strings.Repeat(unit, n/len(unit)+1))[:n]) }
	}
	return map[string]func(*rand.Rand, int) []byte{
		"keys: then - item lines":    lines("keys:\n", "- abcdefghijklmnopqrstuv\n- defghijklmnopqrstuvwxy\n"),
		"key line every 3 lines":     lines("", "keys:\n- abcdefghijklmnopqrstuv\n- defghijklmnopqrstuvwxy\n- ghijklmnopqrstuvwxyzab\n"),
		"secrets: short items":       lines("secrets:\n", "- ab\n"),
		"BEGIN no END":               lines("", "-----BEGIN "),
		"BEGIN no END + label":       lines("", "--"+"---BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n"),
		"BEGIN/END alternating":      lines("", "-----BEGIN CERTIFICATE-----\nMIIB\n-----END CERTIFICATE-----\n"),
		"END only":                   lines("", "-----END PRIVATE KEY-----\n"),
		"yaml list items":            lines("items:\n", "  - abc\n"),
		"yaml secret key list short": lines("api_keys:\n", "  - ab\n"),
		"yaml nested keys":           lines("a:\n", "  b:\n    - c\n"),
		"pem begin markers":          lines("", "--"+"---BEGIN PRIVATE KEY-----\n"),
		"pem begin no end + text":    lines("", "--"+"---BEGIN RSA PRIVATE KEY-----\nabc\n"),
		"pem dashes":                 lines("", "-----"),
		"colon lines":                lines("", "key:\n"),
		"pem begin end":              repeatShape("-----BEGIN X-----END X-----"),
	}
}

const structuredEnglishLimit = 1.3

func TestLargeShapesStayLinear(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	cost := func(gen func(*rand.Rand, int) []byte, size int) time.Duration {
		v := base64.StdEncoding.EncodeToString(gen(rand.New(rand.NewSource(7)), size))
		return bestCPU(1, func() { corpusFindingsByBlob(t, srv, []string{v}, []int{0}) })
	}
	small, large := 64<<10, 256<<10
	if scaletest.Race {
		small, large = 4<<10, 16<<10
	}
	englishGen := func(r *rand.Rand, n int) []byte { return corpusEnglish(r, n) }
	before := cost(englishGen, large)
	shapes := yamlAndPEMShapes()
	var names []string
	for n := range shapes {
		names = append(names, n)
	}
	sort.Strings(names)
	type result struct{ s, l time.Duration }
	results := map[string]result{}
	for _, name := range names {
		results[name] = result{cost(shapes[name], small), cost(shapes[name], large)}
	}
	english := max(before, cost(englishGen, large))
	for _, name := range names {
		r := results[name]
		t.Logf("%-28s %v then %v (%.1fx), %.2fx english", name, r.s.Round(time.Millisecond), r.l.Round(time.Millisecond), float64(r.l)/float64(max(r.s, time.Millisecond)), float64(r.l)/float64(english))
		if scaletest.Race {
			continue
		}
		if r.l > 8*r.s+50*time.Millisecond {
			t.Errorf("%s: 4x the input took %v after %v", name, r.l, r.s)
		}
		if float64(r.l) > structuredEnglishLimit*float64(english) {
			t.Errorf("%s: %v at %d bytes is %.2fx plain English measured in the same run (%v), limit %.1fx", name, r.l, large, float64(r.l)/float64(english), english, structuredEnglishLimit)
		}
	}
	t.Logf("english %v (first %v)", english, before)
}
