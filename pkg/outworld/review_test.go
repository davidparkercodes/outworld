package outworld

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

func keyedEngine(t *testing.T, key []byte, withDict bool, extra ...detect.Detector) *Engine {
	t.Helper()
	v, err := vault.Open(vault.NewMemoryBackend(key))
	if err != nil {
		t.Fatal(err)
	}
	o := Options{Vault: v, Policy: DefaultPolicy(), Extra: extra}
	if withDict {
		d, err := detect.ParseDictionaryCSV(strings.NewReader(dict))
		if err != nil {
			t.Fatal(err)
		}
		o.Dictionary = d
	}
	e, err := New(o)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func primedMask(t *testing.T, s *Session, texts ...string) []string {
	t.Helper()
	s.SetPriming(true)
	for _, in := range texts {
		if _, err := s.Mask(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	s.SetPriming(false)
	out := make([]string, len(texts))
	for i, in := range texts {
		var err error
		if out[i], err = s.Mask(ctx, in); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func surrogatesOf(e *Engine, real string) map[string]string {
	out := map[string]string{}
	for _, r := range e.vault.Records() {
		if strings.EqualFold(r.Real, real) {
			out[r.Kind] = r.Surrogate
		}
	}
	return out
}

func TestResidueRecordDoesNotShadowLaterRealName(t *testing.T) {
	surnameFirst := reDetector{name: "test:surname-first", typ: entity.Person, re: regexp.MustCompile(`\bKowalewicz, [A-Z][a-z]+`), skip: "Thiha"}
	thiha := reDetector{name: "test:first", typ: entity.Location, re: regexp.MustCompile(`\bThiha\b`)}
	full := reDetector{name: "test:full", typ: entity.Person, re: regexp.MustCompile(`\bAnna Kowalewicz\b`)}
	later := "the ticket was filed under kowalewicz in the tracker"

	control := residueEngine(t, thiha, surnameFirst, full)
	maskRoundTrip(t, control, "Anna Kowalewicz filed it.")
	if out := maskRoundTrip(t, control, later); strings.Contains(out, "kowalewicz") {
		t.Fatalf("control: known surname not masked: %q", out)
	}

	e := residueEngine(t, thiha, surnameFirst, full)
	first := maskRoundTrip(t, e, "Present: Kowalewicz, Thiha")
	if !residueRecord(e, "Kowalewicz") {
		t.Fatalf("expected a residue record: %+v", e.vault.Records())
	}
	if out := maskRoundTrip(t, e, "filed under kowalewicz last week"); out != "filed under kowalewicz last week" {
		t.Fatalf("a residue-only word must not be hunted: %q", out)
	}
	second := maskRoundTrip(t, e, "Anna Kowalewicz filed it.")
	recs := surrogatesOf(e, "Kowalewicz")
	if recs["person.family"] == "" || recs["person.family"] != recs[residueKind] {
		t.Fatalf("one surname, two surrogates: %+v", recs)
	}
	if !strings.Contains(first, recs[residueKind]) || !strings.Contains(second, recs[residueKind]) {
		t.Fatalf("surrogate %q missing from %q / %q", recs[residueKind], first, second)
	}
	out := maskRoundTrip(t, e, later)
	if strings.Contains(strings.ToLower(out), "kowalewicz") {
		t.Fatalf("real surname sent in clear after a residue record: %q", out)
	}
	if !strings.Contains(out, strings.ToLower(recs["person.family"])) {
		t.Fatalf("expected the family surrogate in %q", out)
	}
}

func TestRealNameRecordIsReusedByResidue(t *testing.T) {
	surnameFirst := reDetector{name: "test:surname-first", typ: entity.Person, re: regexp.MustCompile(`\bKowalewicz, [A-Z][a-z]+`), skip: "Thiha"}
	thiha := reDetector{name: "test:first", typ: entity.Location, re: regexp.MustCompile(`\bThiha\b`)}
	full := reDetector{name: "test:full", typ: entity.Person, re: regexp.MustCompile(`\bAnna Kowalewicz\b`)}
	e := residueEngine(t, thiha, surnameFirst, full)
	maskRoundTrip(t, e, "Anna Kowalewicz filed it.")
	maskRoundTrip(t, e, "Present: Kowalewicz, Thiha")
	recs := surrogatesOf(e, "Kowalewicz")
	if len(recs) != 1 || recs["person.family"] == "" {
		t.Fatalf("want one family record, got %+v", recs)
	}
}

func TestOneRealNamePartOneSurrogate(t *testing.T) {
	given := reDetector{name: "test:given", typ: entity.Person, re: regexp.MustCompile(`\bCasimira(?: [A-Z][a-z]+)?`)}
	family := reDetector{name: "test:honorific", typ: entity.Person, re: regexp.MustCompile(`\bHelmut Casimira\b`)}
	e := residueEngine(t, given, family)
	a := maskRoundTrip(t, e, "Casimira Oduya called.")
	b := maskRoundTrip(t, e, "Then Helmut Casimira called.")
	sur := strings.Fields(a)[0]
	if !strings.Contains(b, " "+sur+" called.") {
		t.Fatalf("the same word got two surrogates: %q then %q", a, b)
	}
}

var addrTakesFirstName = reDetector{name: "test:addr", typ: entity.Address, re: regexp.MustCompile(`Rua Augusta 100, 1100-048 Lisboa Maria`)}

func TestLoserRemainderIsMaskedNotSent(t *testing.T) {
	name := reDetector{name: "test:name", typ: entity.Person, re: regexp.MustCompile(`\bMaria Kowalski Lopez\b`)}
	e := residueEngine(t, addrTakesFirstName, name)
	in := "Ship it to Rua Augusta 100, 1100-048 Lisboa Maria Kowalski Lopez said hello"
	s := e.NewSession("")
	s.SetTrace(true)
	out, err := s.Mask(ctx, in)
	if err != nil || s.Err() != nil {
		t.Fatalf("%v %v", err, s.Err())
	}
	for _, r := range []string{"Kowalski", "Lopez", "Maria", "Augusta"} {
		if strings.Contains(out, r) {
			t.Fatalf("%q sent in clear: %q", r, out)
		}
	}
	if !strings.HasPrefix(out, "Ship it to ") || !strings.HasSuffix(out, " said hello") {
		t.Fatalf("text outside the spans changed: %q", out)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate: %q", back)
	}
	residues := 0
	for _, r := range s.Replacements() {
		if r.Source == residueSource {
			residues++
		}
	}
	if residues != 1 {
		t.Fatalf("want one residue replacement, got %+v", s.Replacements())
	}
	again := e.NewSession("")
	if out2, _ := again.Mask(ctx, in); out2 != out || again.MemoHits() != 1 {
		t.Fatalf("memo must return the bytes already sent: %q vs %q", out2, out)
	}
	fresh := e.NewSession("other-scope")
	if out3, _ := fresh.Mask(ctx, in); strings.Contains(out3, "Kowalski") || fresh.Rehydrate(out3) != in {
		t.Fatalf("second scope: %q", out3)
	}
}

func TestLoserRemainderOfStripTypeIsRedacted(t *testing.T) {
	token := reDetector{name: "test:token", typ: entity.Secret, re: regexp.MustCompile(`Maria-tok-[a-z0-9]+`)}
	e := residueEngine(t, addrTakesFirstName, token)
	s := e.NewSession("")
	out, _ := s.Mask(ctx, "Ship it to Rua Augusta 100, 1100-048 Lisboa Maria-tok-9f3a7c21 today")
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected block: %v (%q)", err, out)
	}
	if strings.Contains(out, "9f3a7c21") || !strings.Contains(out, "[redacted") || !strings.HasSuffix(out, " today") {
		t.Fatalf("secret remainder left in %q", out)
	}
}

func TestLoserRemainderOfOtherTypeIsMaskedAsItsType(t *testing.T) {
	phone := reDetector{name: "test:phone", typ: entity.Phone, re: regexp.MustCompile(`Maria x\d{4}`)}
	e := residueEngine(t, addrTakesFirstName, phone)
	s := e.NewSession("")
	in := "Ship it to Rua Augusta 100, 1100-048 Lisboa Maria x4471 today"
	out, _ := s.Mask(ctx, in)
	if err := s.Err(); err != nil {
		t.Fatalf("unexpected block: %v (%q)", err, out)
	}
	if strings.Contains(out, "4471") || !strings.HasSuffix(out, " today") {
		t.Fatalf("phone remainder sent in clear: %q", out)
	}
	if back := s.Rehydrate(out); back != in {
		t.Fatalf("rehydrate: %q", back)
	}
}

func TestLoserRemainderWithoutContentIsLeftAlone(t *testing.T) {
	phone := reDetector{name: "test:phone", typ: entity.Phone, re: regexp.MustCompile(`Maria, x4\b`)}
	e := residueEngine(t, addrTakesFirstName, phone)
	s := e.NewSession("")
	s.SetTrace(true)
	in := "Ship it to Rua Augusta 100, 1100-048 Lisboa Maria, x4 today"
	out, _ := s.Mask(ctx, in)
	if err := s.Err(); err != nil || !strings.HasSuffix(out, ", x4 today") || len(s.Replacements()) != 1 {
		t.Fatalf("%v %q %+v", err, out, s.Replacements())
	}
}

func TestCoveredLosersAreNotMaskedTwice(t *testing.T) {
	full := reDetector{name: "test:full", typ: entity.Person, re: regexp.MustCompile(`\bOttoline Brandvold\b`)}
	last := reDetector{name: "test:last", typ: entity.Person, re: regexp.MustCompile(`\bBrandvold\b`)}
	first := reDetector{name: "test:firstname", typ: entity.Person, re: regexp.MustCompile(`\bOttoline\b`)}
	tail := reDetector{name: "test:tail", typ: entity.Secret, re: regexp.MustCompile(`Brandvold [;,.]`)}
	e := residueEngine(t, full, last, first, tail)
	s := e.NewSession("")
	s.SetTrace(true)
	in := "Ask Ottoline Brandvold ; she knows."
	out, err := s.Mask(ctx, in)
	if err != nil || s.Err() != nil {
		t.Fatalf("%v %v", err, s.Err())
	}
	if len(s.Replacements()) != 1 || s.Stats().Masked != 1 {
		t.Fatalf("want one replacement, got %+v (%q)", s.Replacements(), out)
	}
	if !strings.HasPrefix(out, "Ask ") || !strings.HasSuffix(out, " ; she knows.") || s.Rehydrate(out) != in {
		t.Fatalf("out %q", out)
	}
}

func TestRememberedTextStaysByteStableWithOverlaps(t *testing.T) {
	name := reDetector{name: "test:name", typ: entity.Person, re: regexp.MustCompile(`\bMaria Kowalski Lopez\b`)}
	e := residueEngine(t, addrTakesFirstName, name)
	in := "Ship it to Rua Augusta 100, 1100-048 Lisboa Maria Kowalski Lopez said hello"
	sent := "Ship it to SOMETHING ALREADY SENT said hello"
	e.NewSession("").Remember(in, sent)
	if out, _ := e.NewSession("").Mask(ctx, in); out != sent {
		t.Fatalf("memo bytes changed: %q", out)
	}
}

func TestInitialismSurrogateAvoidsCapsInTheRequest(t *testing.T) {
	in := "Priya Raman reviewed it. She works on the OB ward.\nPR: looks good to me"
	for k := 1; k <= 120; k++ {
		e := keyedEngine(t, bytes.Repeat([]byte{byte(k)}, 32), true)
		s := e.NewSession("")
		out := primedMask(t, s, in)[0]
		if err := s.Err(); err != nil {
			t.Fatalf("key %d: %v", k, err)
		}
		if !strings.Contains(out, "works on the OB ward.") {
			t.Fatalf("key %d: %q", k, out)
		}
		sur := out[strings.IndexByte(out, '\n')+1 : strings.LastIndex(out, ":")]
		if sur == "PR" || sur == "OB" || len(sur) != 2 {
			t.Fatalf("key %d: initialism surrogate %q in %q", k, sur, out)
		}
		if back := s.Rehydrate(out); back != in {
			t.Fatalf("key %d: rehydrate %q", k, back)
		}
		low := strings.ToLower(sur)
		reply := "The OB ward is busy. " + sur + " said so; " + low + " and x" + sur + " and " + sur + "y stay."
		want := "The OB ward is busy. PR said so; " + low + " and x" + sur + " and " + sur + "y stay."
		if got := s.Rehydrate(reply); got != want {
			t.Fatalf("key %d: reply %q", k, got)
		}
		r := s.NewStreamRehydrator(false)
		var b strings.Builder
		for i := 0; i < len(reply); i += 3 {
			b.WriteString(r.Push(reply[i:min(i+3, len(reply))]))
		}
		b.WriteString(r.Flush())
		if b.String() != want {
			t.Fatalf("key %d: stream %q", k, b.String())
		}
	}
}

func TestRealTextEqualToInitialismSurrogateIsRemasked(t *testing.T) {
	for k := 1; k <= 60; k++ {
		e := keyedEngine(t, bytes.Repeat([]byte{byte(k)}, 32), false)
		first := "Attendees: Milica Jovanović, Oksana Hrytsenko, Dana Whitfield\nMJ: I will send the notes.\nOH: thanks.\nDW: noted."
		s := e.NewSession("")
		out := primedMask(t, s, first)[0]
		if s.Err() != nil || s.Rehydrate(out) != first {
			t.Fatalf("key %d: %v %q", k, s.Err(), out)
		}
		var surs []string
		for _, r := range e.vault.Records() {
			if isInitialism(r.Kind, r.Surrogate) {
				surs = append(surs, r.Surrogate)
			}
		}
		if len(surs) != 3 {
			t.Fatalf("key %d: want 3 initialism records, got %v", k, surs)
		}
		later := "The " + surs[0] + " unit and the " + surs[1] + " desk share a budget with " + surs[2] + "."
		s2 := e.NewSession("")
		out2 := primedMask(t, s2, later)[0]
		if err := s2.Err(); err != nil {
			t.Fatalf("key %d: %v", k, err)
		}
		for _, sur := range surs {
			if regexp.MustCompile(`\b` + sur + `\b`).MatchString(out2) {
				t.Fatalf("key %d: real %q equal to an initialism surrogate sent as-is: %q", k, sur, out2)
			}
		}
		if back := s2.Rehydrate(out2); back != later {
			t.Fatalf("key %d: rehydrate %q from %q", k, back, out2)
		}
		if back := s2.Rehydrate("Reply about " + surs[0]); back != "Reply about MJ" {
			t.Fatalf("key %d: stored initialism no longer restores: %q", k, back)
		}
	}
}

var familyFirstInputs = []string{
	"Zhao Hao wrote to Li Wei about the invoice.",
	"Nguyen Van An joined the call with Tran Thi Mai.",
	"Kim Min-jun and Park Ji-woo reviewed the draft. Min-jun signed off.",
	"Chen Wei-Ling wrote. Wei-Ling replied.",
	"Zhang Wei met Wang Fang and Liu Yang, then Zhou Xin called Wu Lei.",
	"Pham Thi Lan and Le Van Duc are out; Lan returns Monday.",
	"Choi Soo-ah emailed Jung Ha-eun and Kang Dong-won.",
	"Huang Lei said Sun Yang agreed with Xu Jing and Ma Lin.",
}

func TestPlacesAndTermsSurviveRehydration(t *testing.T) {
	reply := "We flew from Hong Kong to Ho Chi Minh City and on to Viet Nam. The Ming dynasty, the Tang dynasty, the Song dynasty, the Qing court, the Yuan and Han Chinese history came up, as did Thanh Hoa, Kowloon and the Zhou and Liao periods. hong kong, ming vase, tang poetry, han river."
	for k := 1; k <= scaletest.Size(200*16)/16; k++ {
		e := keyedEngine(t, bytes.Repeat([]byte{byte(k)}, 32), false)
		s := e.NewSession("")
		outs := primedMask(t, s, familyFirstInputs...)
		if err := s.Err(); err != nil {
			t.Fatalf("key %d: %v", k, err)
		}
		for i, out := range outs {
			if back := s.Rehydrate(out); back != familyFirstInputs[i] {
				t.Fatalf("key %d: round trip %q -> %q -> %q", k, familyFirstInputs[i], out, back)
			}
		}
		if got := s.Rehydrate(reply); got != reply {
			t.Fatalf("key %d: reply changed to %q (masked: %q)", k, got, outs)
		}
	}
}

func TestHyphenatedGivenNameHasOneRendering(t *testing.T) {
	pair := regexp.MustCompile(`\S+-\S+`)
	for k := 1; k <= 80; k++ {
		key := bytes.Repeat([]byte{byte(k)}, 32)
		for _, primed := range []bool{false, true} {
			e := keyedEngine(t, key, false)
			s := e.NewSession("")
			in := "Chen Wei-Ling wrote. Wei-Ling replied."
			var out string
			if primed {
				out = primedMask(t, s, in)[0]
			} else {
				out, _ = s.Mask(ctx, in)
			}
			m := pair.FindAllString(strings.NewReplacer(".", " ").Replace(out), -1)
			if s.Err() != nil || len(m) != 2 || m[0] != m[1] || strings.Contains(out, "Wei") || strings.Contains(out, "Ling") {
				t.Fatalf("key %d primed=%v: %q", k, primed, out)
			}
			if s.Rehydrate(out) != in {
				t.Fatalf("key %d: rehydrate %q", k, s.Rehydrate(out))
			}
		}
		e := keyedEngine(t, key, false)
		lone := maskRoundTrip(t, e, "Thanks, Wei-Ling replied to Chen Wei-Ling yesterday.")
		m := pair.FindAllString(lone, -1)
		if len(m) != 2 || m[0] != m[1] {
			t.Fatalf("key %d: lone form first: %q", k, lone)
		}
		later := maskRoundTrip(t, e, "Chen Wei-Ling wrote.")
		if !strings.Contains(later, " "+m[0]+" wrote.") {
			t.Fatalf("key %d: %q then %q", k, lone, later)
		}
	}
}

var headVaultKey = bytes.Repeat([]byte{7}, 32)

func TestVaultWrittenByEarlierReleaseKeepsItsBytes(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "head-vault.bin"))
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		In  string `json:"in"`
		Out string `json:"out"`
	}
	data, err := os.ReadFile(filepath.Join("testdata", "head-vault.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) < 15 {
		t.Fatalf("fixture too small: %d", len(cases))
	}
	open := func(path string) *Engine {
		v, err := vault.OpenFile(path, headVaultKey)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { v.Close() })
		e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	check := func(e *Engine, label string) {
		for _, c := range cases {
			s := e.NewSession("")
			out := primedMask(t, s, c.In)[0]
			if err := s.Err(); err != nil {
				t.Errorf("%s: %q: %v", label, c.In, err)
			}
			if out != c.Out {
				t.Errorf("%s: bytes changed for %q\n was %q\n now %q", label, c.In, c.Out, out)
			}
			if back := s.Rehydrate(out); back != c.In {
				t.Errorf("%s: rehydrate %q -> %q", label, out, back)
			}
		}
	}
	path := filepath.Join(t.TempDir(), "head-vault.bin")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	check(open(path), "first open")
	e := open(path)
	maskRoundTrip(t, e, "Ask Ottoline Brandvold about TODO(o.brandvold) and Ngo Thi Hoa.")
	check(e, "after new records")
	check(open(path), "reopened")

	fresh := keyedEngine(t, headVaultKey, false)
	maskRoundTrip(t, fresh, "Please ask Karen Chen about the retry loop.")
	out := maskRoundTrip(t, fresh, "TODO(k.chen): tidy the retry loop before release.")
	if strings.Contains(out, "(k.") {
		t.Fatalf("a vault without earlier-release records must not keep the real initial: %q", out)
	}
}

var fuzzInputs = append([]string{
	"Marisol Quenby (m.quenby@halvorsen-bio.example, 415-555-0134) starts on 2026-04-17.",
	"TODO(k.chen): tidy the retry loop before release.",
	"Please ask Karen Chen about the retry loop, then ping kchen.",
	"git checkout jsmith/fix-login and ping @mwilliams when done.",
	"FIXME(d_whitfield): see /home/ohrytsenko/notes and ask J. Smith.",
	"branch k.chen/retry was rebased onto dwhitfield/main by a.okafor",
	"Reviewed-by: Dana Whitfield <dana.whitfield@corp.example>",
	"Attendees: Milica Jovanović, Maria Jensen, Oksana Hrytsenko\nMJ: I will send the notes.\nOH: thanks.",
	"Present: J. R. Adjei (chair), Milica Jovanović, Lane Pruitt\n1. LP reported the council will repaint. JRA to write to the council.\n2. MJ to circulate the quotes.\nLP: happy to help.",
	"Priya Raman reviewed it. She works on the OB ward.\nPR: looks good to me",
	"Dr. Okafor met Mary Smith-Jones and Willem van der Heijden at 14 Birch Lane.",
	"Contact Oksana Hrytsenko at +44 20 7946 0958 or oksana_h@mail.example.",
	"Send it to 221 Alder Street, Apt 4B, Richmond, VA 23220 for Tobias Lindqvist-Moreau.",
	"Ship it to Rua Augusta 100, 1100-048 Lisboa. Maria Kowalski Lopez said hello.",
	"Hi John,\nJohnny's appointment is on March 15, 2024. JOHN SMITH and john smith are the same.",
	"between WILLEM VAN DER HEIJDEN and the tenant, with the consent of Mr van der Heijden.",
	"Will Turner booked The Turner Room and the Turner Award dinner. WT to confirm.",
	"Host 10.0.0.5 answered Sam Abbott; SA: confirmed. The MDH and HL teams agreed with YM.",
	"We flew from Hong Kong to Ho Chi Minh City; the Ming dynasty and Tang dynasty came up with Han Chinese history.",
	"Ursula Stone and Ian Tate met. US to send aid. IT to fix laptops.",
	"Thanks, Wei-Ling replied to Chen Wei-Ling yesterday.",
	"cc Gina Bell, Carl Underwood and Ines Tavares (GB, CU, IT) on the PR for the OB and ER wards.",
}, familyFirstInputs...)

func TestRoundTripAcrossRandomKeys(t *testing.T) {
	sessions := scaletest.Size(400*16) / 16
	if n, err := strconv.Atoi(os.Getenv("OUTWORLD_ROUNDTRIP_SESSIONS")); err == nil && n > 0 {
		sessions = n
	}
	rng := rand.New(rand.NewSource(20261122))
	failures := 0
	for i := 0; i < sessions && failures < 20; i++ {
		key := make([]byte, 32)
		rng.Read(key)
		e := keyedEngine(t, key, i%2 == 0)
		for req := 0; req < 2; req++ {
			n := 1 + rng.Intn(5)
			texts := make([]string, n)
			for j := range texts {
				texts[j] = fuzzInputs[rng.Intn(len(fuzzInputs))]
			}
			s := e.NewSession(fmt.Sprintf("scope-%d", rng.Intn(2)))
			outs := primedMask(t, s, texts...)
			if err := s.Err(); err != nil {
				failures++
				t.Errorf("session %d key %x: blocked: %v\n texts %q\n violations %+v", i, key[:4], err, texts, s.Violations())
				continue
			}
			for j, out := range outs {
				if back := s.Rehydrate(out); back != texts[j] {
					failures++
					t.Errorf("session %d key %x: round trip\n in   %q\n out  %q\n back %q", i, key[:4], texts[j], out, back)
				}
			}
			joined := strings.Join(outs, "\n\n")
			if back := s.Rehydrate(joined); back != strings.Join(texts, "\n\n") {
				failures++
				t.Errorf("session %d key %x: joined round trip\n out  %q\n back %q", i, key[:4], joined, back)
			}
		}
	}
	t.Logf("sessions=%d requests=%d failures=%d", sessions, sessions*2, failures)
}
