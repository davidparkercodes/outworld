package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

func sigSources(t *testing.T, srv *Server, raw []byte) map[string]int {
	t.Helper()
	return corpusFindings(t, srv, []string{base64.StdEncoding.EncodeToString(raw)}, []int{0})
}

func wrapRandom(r *rand.Rand, mid []byte) []byte {
	out := append(uniformBytes(r, 24), mid...)
	return append(out, uniformBytes(r, 24)...)
}

func nonFrameBytes(r *rand.Rand, n int) []byte {
	b := uniformBytes(r, n)
	b[0] = 0
	return b
}

func TestPIIInsideNonProtobufBinary(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1818))
	frame := []byte{0x08, 0x96, 0x01, 0x12, 0x10}
	pii := []string{"4111111111111111", "4111 1111 1111 1111", "4111-1111-1111-1111", "123-45-6789", "jane.doe@example.com", "call 804-555-1234 today"}
	for _, p := range pii {
		for i := 0; i < 100; i++ {
			blob := wrapRandom(r, []byte(p))
			blob[0] = 0
			if len(sigSources(t, srv, blob)) == 0 {
				t.Fatalf("%q between random bytes: not blocked (%x)", p, blob)
			}
		}
	}
	cases := map[string][]byte{
		"valid frame":           append(append([]byte{}, frame...), "4111111111111111"...),
		"valid frame then 00ff": append(append(append([]byte{}, frame...), "4111111111111111"...), 0x00, 0xff),
		"00ff01 card 00ff":      append(append([]byte{0x00, 0xff, 0x01}, "4111111111111111"...), 0x00, 0xff),
	}
	for _, s := range []string{"x4111111111111111", "id4111111111111111", "4111111111111111x", "a5555555555554444", "z378282246310005"} {
		cases["short field "+s] = field(1, []byte(s))
	}
	for name, blob := range cases {
		if len(sigSources(t, srv, blob)) == 0 {
			t.Errorf("%s: not blocked", name)
		}
	}
	clean := map[string][]byte{
		"valid frame":           append(append([]byte{}, frame...), "4111111111111112"...),
		"valid frame then 00ff": append(append(append([]byte{}, frame...), "4111111111111112"...), 0x00, 0xff),
		"00ff01 id 00ff":        append(append([]byte{0x00, 0xff, 0x01}, "build 20261123 ok"...), 0x00, 0xff),
		"phone without cue":     wrapRandom(r, []byte("x 804-555-1234 y")),
		"undashed ssn":          wrapRandom(r, []byte("ref 123456789 x")),
		"short field id":        field(1, []byte("x4111111111111112")),
		"hex id":                field(1, []byte("a4111111111111111b2c3d4e5f60718293")),
	}
	for name, blob := range clean {
		blob[0] = 0
		if found := sigSources(t, srv, blob); len(found) > 0 {
			t.Errorf("PII-free %s: %v", name, found)
		}
	}
}

func TestRandomBlobsNeverBlock(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	for _, size := range []int{800, 4096, 16384} {
		r := rand.New(rand.NewSource(int64(1818 + size)))
		n := 2000 / corpusScale
		blocked := 0
		for done := 0; done < n; done += 25 {
			var blobs []string
			var pos []int
			for i := 0; i < 25; i++ {
				blobs = append(blobs, base64.StdEncoding.EncodeToString(nonFrameBytes(r, size)))
				pos = append(pos, done+i)
			}
			found, hit := corpusFindingsByBlob(t, srv, blobs, pos)
			if hit > 0 {
				t.Errorf("size %d: %v", size, found)
			}
			blocked += hit
		}
		t.Logf("size %d: %d of %d random blobs blocked", size, blocked, n)
	}
}

func TestSignatureOfACommitIsFailClosed(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	s := "the signature of the commit is q7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9"
	if found := sigSources(t, srv, field(1, []byte(s))); found["regex:context_token"] == 0 {
		t.Fatalf("signature is a key word by decision: %v", found)
	}
}

func TestKeyWordAcrossLineBreak(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	for _, s := range []string{"my api key\nq7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9", "password:\n  q7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9",
		"the token =\nq7Zk2mN9xR4tW8vB1cD5fG3hJ6kL0pS2uY4aE7iO9"} {
		if len(sigSources(t, srv, field(1, []byte(s)))) == 0 {
			t.Errorf("%q: not blocked", s)
		}
	}
}

func TestPIIInBinaryFieldOfAValidFrame(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1819))
	for _, p := range []string{"jane.doe@example.com", "123-45-6789", "4111111111111111", "phone 804-555-1234"} {
		for i := 0; i < 50; i++ {
			mid := append(append(uniformBytes(r, 40), p...), uniformBytes(r, 40)...)
			raw := append(append(field(1, []byte("hello world session")), field(2, mid)...), field(3, uniformBytes(r, 60))...)
			if !parseFrame(raw).framed {
				continue
			}
			if len(sigSources(t, srv, raw)) == 0 {
				t.Fatalf("%q in a binary field of a valid frame: not blocked (runs %q)", p, printableRuns(mid))
			}
		}
	}
}

func cborText(s string) []byte { return append([]byte{0x60 + byte(len(s))}, s...) }

func cborMap(r *rand.Rand, pairs ...string) []byte {
	b := []byte{0xa0 + byte(len(pairs)/2+2), 0x61, 'f', 0xfb}
	b = append(b, uniformBytes(r, 8)...)
	for _, p := range pairs {
		b = append(b, cborText(p)...)
	}
	b = append(b, 0x61, 'b', 0x58, 64)
	return append(b, uniformBytes(r, 64)...)
}

func TestCBORMapsWithPII(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1820))
	for _, pairs := range [][]string{{"email", "jane.doe@example.com", "name", "Jane"}, {"contact", "jane@example.org", "zip", "23220"},
		{"phone", "804-555-1234"}, {"phone", "8045551234"}, {"phone", "(804) 555-1234"}, {"mobile", "+1 804 555 1234"},
		{"phoneNumber", "804-555-1234"}, {"phoneNumber", "804.555.1234"}, {"phoneNumber", "8045551234"}, {"tel", "8045551234"}} {
		raw := cborMap(r, pairs...)
		if textual(raw) || parseFrame(raw).framed {
			t.Fatalf("cbor sample must be binary: %x", raw)
		}
		if len(sigSources(t, srv, raw)) == 0 {
			t.Errorf("cbor %v: not blocked", pairs)
		}
	}
	for _, pairs := range [][]string{{"id", "8045551234"}, {"name", "Jane", "zip", "23220"}, {"host", "example.com"}} {
		if found := sigSources(t, srv, cborMap(r, pairs...)); len(found) > 0 {
			t.Errorf("cbor %v: %v", pairs, found)
		}
	}
}

func TestUnmappedOffsetsFailClosed(t *testing.T) {
	text := "abc"
	v := outworld.Violation{Source: "regex:email", Start: 2, End: 40}
	for _, src := range []string{"regex:email", "regex:high_entropy", "regex:context_token", "names:first"} {
		v.Source = src
		if rs := []rune(text); !binaryPIIKept(rs, v) || !decodedKept(rs, v) || !noisyFrameKept(rs, v) || !tokenKept(rs, v) {
			t.Fatalf("a %s finding whose offsets do not map must be kept", src)
		}
	}
}

func cborNeighbours(r *rand.Rand, key, value string) []byte {
	printable := func() byte { return byte(0x21 + r.Intn(0x5e)) }
	b := append(uniformBytes(r, 40+r.Intn(40)), 0xa3, printable(), printable())
	b = append(b, cborText(key)...)
	b = append(b, cborText(value)...)
	b = append(b, printable(), printable(), printable())
	return append(b, uniformBytes(r, 40+r.Intn(40))...)
}

func TestCBORPairsWithRandomNeighbours(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	for seed := int64(0); seed < 12; seed++ {
		r := rand.New(rand.NewSource(seed))
		for _, key := range []string{"tel", "phone", "mobile", "phoneNumber", "cell"} {
			for _, value := range []string{"8045551234", "804-555-1234", "+18045551234"} {
				raw := cborNeighbours(r, key, value)
				if len(sigSources(t, srv, raw)) == 0 {
					t.Errorf("seed %d %s=%s: not blocked (%x)", seed, key, value, raw)
				}
			}
			if found := sigSources(t, srv, cborNeighbours(r, key, "hello world")); len(found) > 0 {
				t.Errorf("seed %d %s with no number: %v", seed, key, found)
			}
		}
		for _, key := range []string{"id", "count", "callId", "phoneId"} {
			if found := sigSources(t, srv, cborNeighbours(r, key, "8045551234")); len(found) > 0 {
				t.Errorf("seed %d %s=8045551234: %v", seed, key, found)
			}
		}
		raw := cborNeighbours(r, "contact", "jane.doe@example.com")
		if len(sigSources(t, srv, raw)) == 0 {
			t.Errorf("seed %d email: not blocked (%x)", seed, raw)
		}
	}
}

func bestCPU(rounds int, run func()) time.Duration {
	best := time.Duration(1 << 62)
	for i := 0; i < rounds; i++ {
		runtime.GC()
		start := processCPU()
		run()
		best = min(best, processCPU()-start)
	}
	return best
}

func TestCBORScanStaysLinear(t *testing.T) {
	small := bytes.Repeat([]byte{0x79}, 1<<18)
	large := bytes.Repeat([]byte{0x79}, 1<<20)
	ts := bestCPU(7, func() { cborPairs(small) })
	tl := bestCPU(7, func() { cborPairs(large) })
	if scaletest.Race {
		t.Logf("4x the input: %v then %v", ts, tl)
		return
	}
	if tl > 8*ts+2*time.Millisecond {
		t.Fatalf("4x the input took %v after %v", tl, ts)
	}
}

func headerRun(n int, hi, lo byte) []byte {
	b := make([]byte, 0, n)
	for len(b) < n {
		b = append(b, 0x79, hi, lo)
	}
	return b[:n]
}

func TestCBORItemCheckDoesNotReadTheItem(t *testing.T) {
	const size = 3*4096 + 70000
	short, long := headerRun(size, 0x21, 0x21), headerRun(size, 0x40, 0x40)
	if _, _, ok := cborItemAt(short, printablePrefix(short), 0); !ok {
		t.Fatal("the input must hold valid long text-string items")
	}
	ts := bestCPU(9, func() { cborPairs(short) })
	tl := bestCPU(9, func() { cborPairs(long) })
	if scaletest.Race {
		t.Logf("items of 8481 and 16448 bytes: %v and %v", ts, tl)
		return
	}
	if tl > ts*3/2+2*time.Millisecond {
		t.Fatalf("doubling only the declared item length took %v after %v: the item check reads the item", tl, ts)
	}
}
func cborString(s string) []byte {
	if len(s) < 24 {
		return cborText(s)
	}
	return append([]byte{0x78, byte(len(s))}, s...)
}

func cborRecord(r *rand.Rand, key, value string, pre, post []byte) []byte {
	if pre == nil {
		pre = uniformBytes(r, 2+r.Intn(10))
	}
	if post == nil {
		post = uniformBytes(r, 2+r.Intn(6))
	}
	b := append(append([]byte{}, pre...), 0xa2)
	b = append(b, cborString("id")...)
	b = append(b, 0x1a)
	b = append(b, uniformBytes(r, 4)...)
	b = append(b, cborString(key)...)
	b = append(b, cborString(value)...)
	return append(b, post...)
}

var (
	cborPhoneKeys = []string{"tel", "phone", "mobile", "phoneNumber", "cell", "telephone", "phone_number", "mobileNumber", "cellPhone", "contactPhone",
		"emergencyContactPhoneNumber"}
	cborPhoneValues = []string{"8045551234", "804-555-1234", "+18045551234", "(804) 555-1234", "804.555.1234"}
	cborGlue        = [][2]string{{"A", "0"}, {"z", "9"}, {"7", "x"}, {"E", "e"}, {"ab", "12"}, {"\x00", "5"}, {"5", "\x00"}, {"Q", ""}}
)

func TestCBORRecordSweep(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	seeds := 100 / corpusScale
	missed, total := 0, 0
	missedBy := map[string]int{}
	check := func(name string, raw []byte) {
		total++
		if len(sigSources(t, srv, raw)) == 0 {
			missed++
			missedBy[strings.SplitN(name, "=", 2)[0]+" "+strings.SplitN(name, "=", 2)[1]]++
			if missed <= 10 {
				t.Errorf("%s: not blocked (%x)", name, raw)
			}
		}
	}
	for seed := 0; seed < seeds; seed++ {
		r := rand.New(rand.NewSource(int64(70000 + seed)))
		for _, k := range cborPhoneKeys {
			for _, v := range cborPhoneValues {
				check(k+"="+v, cborRecord(r, k, v, nil, nil))
			}
		}
	}
	for i, g := range cborGlue {
		r := rand.New(rand.NewSource(int64(71000 + i)))
		for _, k := range cborPhoneKeys {
			for _, v := range cborPhoneValues {
				check(fmt.Sprintf("glue %q|%q %s=%s", g[0], g[1], k, v), cborRecord(r, k, v, []byte("\x01\x02"+g[0]), []byte(g[1]+"\x03")))
			}
		}
	}
	if missed > 0 {
		t.Errorf("%d of %d cued CBOR phones not blocked: %v", missed, total, missedBy)
	}
	falsePos := 0
	for seed := 0; seed < 4*seeds; seed++ {
		r := rand.New(rand.NewSource(int64(72000 + seed)))
		for _, k := range []string{"count", "size", "ts", "offset", "seq", "callId", "phoneId", "phone_count"} {
			v := fmt.Sprintf("%d", 1000000000+r.Int63n(9000000000))
			found := sigSources(t, srv, cborRecord(r, k, v, nil, nil))
			for src := range found {
				if strings.HasPrefix(src, "regex:phone_") {
					falsePos++
					t.Errorf("negative %s=%s: %v", k, v, found)
					break
				}
			}
		}
	}
	t.Logf("cued phones blocked %d of %d, negatives blocked %d", total-missed, total, falsePos)
}

func TestCardsGluedToHeaderLikeBytes(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	card := "4111" + "1111" + "1111" + "1111"
	fill := bytes.Repeat([]byte{0x1c}, 24)
	rim := []byte{0xbd, 0xd1, 0x00, 0xff, 0x13}
	wrap := func(mid string) []byte { return append(append(append([]byte{}, rim...), mid...), rim...) }
	blobs := map[string][]byte{
		"D*24 card fill": append(append(bytes.Repeat([]byte("D"), 24), card...), fill...),
		"D*16 card fill": append(append(bytes.Repeat([]byte("D"), 16), card...), fill...),
		"D*8 card fill":  append(append(bytes.Repeat([]byte("D"), 8), card...), fill...),
		"rim D*8 card":   wrap("DDDDDDDD" + card),
		"rim D*16 card":  wrap("DDDDDDDDDDDDDDDD" + card),
		"rim hex card":   wrap("deadbeef" + card),
		"rim card hex":   wrap(card + "deadbeef"),
		"rim hex16 card": wrap("deadbeefdeadbeef" + card),
		"rim sha card":   wrap("sha:deadbeef" + card),
	}
	for name, raw := range blobs {
		if len(sigSources(t, srv, raw)) == 0 {
			t.Errorf("%s: not blocked (%x)", name, raw)
		}
	}
}

func sigTypes(t *testing.T, srv *Server, raw []byte) map[string]bool {
	t.Helper()
	sess := srv.engine.NewSession("")
	narrow, err := srv.maskTwice(context.Background(), sess, opaqueTurn([]string{base64.StdEncoding.EncodeToString(raw)}, []int{0}), config.APIAnthropic)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	if lerr := leakOf(sess, narrow); lerr != nil {
		for _, v := range lerr.Violations {
			out[string(v.Type)] = true
		}
	}
	return out
}

func withoutCBOR(srv *Server, run func()) {
	srv.cborScan = func([]byte) ([]string, []string, bool) { return nil, nil, true }
	defer func() { srv.cborScan = nil }()
	run()
}

func TestCBORPathOnlyAddsFindings(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	r := rand.New(rand.NewSource(1920))
	pii := []string{"4111111111111111", "4111 1111 1111 1111", "123-45-6789", "jane.doe@example.com", "call 804-555-1234", "phone 8045551234",
		"DDDDDDDD4111111111111111", "deadbeef5555555555554444", "ephonel804-555-1234", "x123-45-6789y"}
	n := 300 / corpusScale
	for i := 0; i < n; i++ {
		raw := append(uniformBytes(r, 80+r.Intn(80)), pii[r.Intn(len(pii))]...)
		raw[0] = 0
		if r.Intn(2) == 0 {
			raw = append(raw, cborText(cborPhoneKeys[r.Intn(len(cborPhoneKeys)-1)])...)
			raw = append(raw, cborText(cborPhoneValues[r.Intn(len(cborPhoneValues))])...)
		}
		raw = append(raw, uniformBytes(r, 80+r.Intn(80))...)
		with := sigTypes(t, srv, raw)
		if len(with) == 0 {
			t.Errorf("embedded PII not blocked: %x", raw)
		}
		var without map[string]bool
		withoutCBOR(srv, func() { without = sigTypes(t, srv, raw) })
		for typ := range without {
			if !with[typ] {
				t.Errorf("the CBOR path removed a %s finding from %x: with %v, without %v", typ, raw, with, without)
			}
		}
	}
}

func cborPads(r *rand.Rand) map[string][]byte {
	two := func(n int, head []byte) []byte {
		b := append([]byte{}, head...)
		for i := 0; i < n; i++ {
			b = append(b, 0x62, byte('a'+r.Intn(26)), byte('a'+r.Intn(26)))
		}
		return b
	}
	floats := []byte{0x99, 0x07, 0xd0}
	for i := 0; i < 2000; i++ {
		floats = append(append(floats, 0xfb), uniformBytes(r, 8)...)
	}
	text := strings.Repeat("the build notes cover the release plan for the team. ", 120)[:6000]
	return map[string][]byte{
		"two-letter x4200": two(4200, []byte{0x99, 0x10, 0x68}),
		"two-letter x1000": two(1000, []byte{0x99, 0x03, 0xe8}),
		"english 6000":     append(cborText("description"), append([]byte{0x79, byte(len(text) >> 8), byte(len(text))}, text...)...),
		"floats x2000":     floats,
	}
}

func TestCBORPairsPastManyItems(t *testing.T) {
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	pairs := [][2]string{{"email", "jane.doe@example.com"}, {"phone", "804-555-1234"}, {"phone", "8045551234"}, {"mobile", "+1 804 555 1234"},
		{"tel", "804.555.1234"}, {"ssn", "123-45-6789"}, {"card", "4111111111111111"}, {"phoneNumber", "8045551234"}, {"emergencyContactPhoneNumber", "8045551234"}}
	for i, kv := range pairs {
		r := rand.New(rand.NewSource(int64(1950 + i)))
		for name, pad := range cborPads(r) {
			for _, padFirst := range []bool{true, false} {
				pair := append(cborString(kv[0]), cborString(kv[1])...)
				inner := append(append([]byte{}, pair...), pad...)
				if padFirst {
					inner = append(append([]byte{}, pad...), pair...)
				}
				raw := append(append([]byte{0xa4, 0x00}, uniformBytes(r, 40)...), inner...)
				raw = append(raw, uniformBytes(r, 40)...)
				if len(sigSources(t, srv, raw)) == 0 {
					t.Errorf("%s=%s with pad %s (pad first %v): not blocked", kv[0], kv[1], name, padFirst)
				}
			}
		}
	}
	r := rand.New(rand.NewSource(1960))
	for name, pad := range cborPads(r) {
		raw := append(append([]byte{0xa4, 0x00}, uniformBytes(r, 40)...), pad...)
		raw = append(append(raw, cborString("id")...), cborString("8045551234")...)
		if found := sigSources(t, srv, raw); len(found) > 0 {
			t.Errorf("pad %s with an uncued number: %v", name, found)
		}
	}
}
