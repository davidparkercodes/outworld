package gateway

import (
	"encoding/base64"
	"fmt"
	"math/rand"
	"testing"
)

func TestShortFramedValuesAreText(t *testing.T) {
	tr := &fnTransport{reply: echoReply}
	gw := newTestServer(t, "", tr)
	card := "4111 1111 1111 1111"
	for name, raw := range map[string]string{
		"spaced card":         "\x0a\x13" + card,
		"bare card":           "\x0a\x10" + "4111111111111111",
		"varint then card":    "\x08\x01\x12\x13" + card,
		"card between fields": "\x08\x96\x01\x12\x13" + card + "\x18\x01\x22\x08abcdefgh",
		"phone":               "\x0a\x0e(804) 555-1234",
		"ssn":                 "\x0a\x0b123-45-6789",
		"email":               "\x0a\x14jane.doe@example.com",
	} {
		for where, b := range allOpaquePositions(base64.StdEncoding.EncodeToString([]byte(raw))) {
			resp, out := postJSON(t, gw.URL+"/v1/messages", b, nil)
			if resp.StatusCode != 422 {
				t.Fatalf("%s in %s: %d %s", name, where, resp.StatusCode, out)
			}
		}
	}
}

const alnumChars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func randomToken(r *rand.Rand, extra string) []byte {
	chars := alnumChars + extra
	b := make([]byte, 12+r.Intn(60))
	for i := range b {
		b[i] = chars[r.Intn(len(chars))]
	}
	return b
}

func TestShortValuesHaveNoFindings(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	short := func(gen func(*rand.Rand, int) []byte) func(*rand.Rand) []byte {
		return func(r *rand.Rand) []byte {
			n := 16 + r.Intn(48)
			b := gen(r, max(n, 64))
			return b[:n]
		}
	}
	gens := map[string]func(*rand.Rand) []byte{
		"uniform":  short(uniformBytes),
		"aes":      short(aesLike),
		"zlib":     short(zlibEnglish),
		"alnum":    func(r *rand.Rand) []byte { return randomToken(r, "") },
		"alnum-_.": func(r *rand.Rand) []byte { return randomToken(r, "-_.") },
	}
	const perSeed, perRequest = 2000 / corpusScale, min(50, 2000/corpusScale)
	for name, gen := range gens {
		for _, seed := range reviewerSeeds {
			name, gen, seed := name, gen, seed
			t.Run(fmt.Sprintf("%s/%d", name, seed), func(t *testing.T) {
				t.Parallel()
				r := rand.New(rand.NewSource(seed))
				total := 0
				for done := 0; done < perSeed; done += perRequest {
					var blobs []string
					var pos []int
					for i := 0; i < perRequest; i++ {
						blobs = append(blobs, base64.StdEncoding.EncodeToString(gen(r)))
						pos = append(pos, done+i)
					}
					for src, n := range corpusFindings(t, srv, blobs, pos) {
						total += n
						t.Logf("%s seed %d: %s %d", name, seed, src, n)
					}
				}
				if total != 0 {
					t.Fatalf("%s seed %d: %d findings", name, seed, total)
				}
			})
		}
	}
}
