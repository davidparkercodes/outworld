package outworld

import (
	"bytes"
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode"

	"github.com/davidparkercodes/outworld/internal/scaletest"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

type countingBackend struct {
	vault.Backend
	updates int
}

func (c *countingBackend) Update(fn func(newer []vault.Record) ([]vault.Record, error)) error {
	c.updates++
	return c.Backend.Update(fn)
}

func countingEngine(t *testing.T, path string) (*Engine, *vault.Vault, *countingBackend) {
	t.Helper()
	fb, err := vault.NewFileBackendWithSecret(path, bytes.Repeat([]byte{9}, 32), bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	cb := &countingBackend{Backend: fb}
	v, err := vault.Open(cb)
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	return e, v, cb
}

func caseWidthVariants(word string, n int) []string {
	rs := []rune(strings.ToLower(word))
	var out []string
	for mask := 0; len(out) < n && mask < 1<<(2*len(rs)); mask++ {
		var b strings.Builder
		for i, r := range rs {
			c := r
			if mask>>(2*i)&1 == 1 {
				c = r - 'a' + 'A'
			}
			if mask>>(2*i+1)&1 == 1 {
				c += 0xFEE0
			}
			b.WriteRune(c)
		}
		out = append(out, b.String())
	}
	return out
}

func TestSpellingFormsAreBoundedAndBatched(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v.db")
	e, v, cb := countingEngine(t, path)
	if _, err := e.NewSession("seed").Mask(ctx, "Ruby Kelly called about the invoice."); err != nil {
		t.Fatal(err)
	}
	variants := caseWidthVariants("kelly", 1024)
	if len(variants) != 1024 {
		t.Fatalf("got %d variants", len(variants))
	}
	in := "Ruby " + strings.Join(variants, ", Ruby ") + " called."
	before := cb.updates
	s := e.NewSession("x")
	out, err := s.Mask(ctx, in)
	if err != nil {
		t.Fatal(err)
	}
	if cb.updates-before > 1 {
		t.Errorf("%d durable updates for one request, want at most 1", cb.updates-before)
	}
	for _, vr := range variants {
		if strings.Contains(out, " "+vr+",") || strings.HasSuffix(out, " "+vr+" called.") {
			t.Fatalf("variant %q left raw", vr)
		}
	}
	perBase := map[string]int{}
	for _, r := range v.RecordsFrom(0) {
		if r.Kind == "name.form" {
			perBase[textutil.Fold(textutil.FoldForms(r.Real))]++
		}
	}
	for base, n := range perBase {
		if n > vault.MaxNameForms {
			t.Errorf("%d forms for %q, bound is %d", n, base, vault.MaxNameForms)
		}
	}
	v.Close()
	e2, v2, _ := countingEngine(t, path)
	defer v2.Close()
	back := e2.NewSession("cli").Rehydrate(out)
	exact := 0
	for _, vr := range variants {
		if strings.Contains(back, " "+vr+",") || strings.HasSuffix(back, " "+vr+" called.") {
			exact++
		}
	}
	if exact < vault.MaxNameForms-1 {
		t.Errorf("only %d variants restored exactly, want the bounded set", exact)
	}
}

func TestSpellingFormsScaleLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	ctx := context.Background()
	variants := caseWidthVariants("kelly", 1024)
	scaletest.Check(t, "mask spelling variants", scaletest.Size(1024), func(n int) func() {
		e := newEngine(t, nil)
		if _, err := e.NewSession("seed").Mask(ctx, "Ruby Kelly called about the invoice."); err != nil {
			t.Fatal(err)
		}
		text := "Ruby " + strings.Join(variants[:n], ", Ruby ") + " called."
		scope := uniqueScope("v")
		return func() {
			if _, err := e.NewSession(scope()).Mask(ctx, text); err != nil {
				t.Fatal(err)
			}
		}
	})
}

func TestVariantStandInShapes(t *testing.T) {
	ctx := context.Background()
	strayCap := func(w string) bool {
		rs := []rune(w)
		ups := 0
		for _, r := range rs {
			if r >= 'A' && r <= 'Z' {
				ups++
			}
		}
		return len(rs) > 2 && ups == 1 && rs[len(rs)-1] >= 'A' && rs[len(rs)-1] <= 'Z'
	}
	cases := []struct {
		seed, in string
		check    func(out string) string
	}{
		{"Ruby Kelly called about the invoice.", "ping rUBYkELLY today", nil},
		{"Ruby Kelly called about the invoice.", "ping rUBY kELLY today", nil},
		{"Ruby Kelly called about the invoice.", "ping ruBy kelLy today", nil},
		{"Zoe Kelly called about the invoice.", "ping Zoë Kelly today", func(out string) string {
			if n := len(strings.Fields(out)); n != 4 {
				return "two-word name became one word"
			}
			return ""
		}},
		{"José Ortega called about the invoice.", "ping José Ortega today", func(out string) string {
			if n := len(strings.Fields(out)); n != 4 {
				return "two-word name became one word"
			}
			return ""
		}},
		{"Zoe Kelly called about the invoice.", "ping Zo\u00eb Kelly today", func(out string) string {
			for _, w := range strings.Fields(out)[1:3] {
				rs := []rune(w)
				for _, r := range rs[1:] {
					if unicode.IsUpper(r) {
						return "case pattern shifted in " + w
					}
				}
			}
			return ""
		}},
		{"Zo\u00eb Kelly called about the invoice.", "ping Zoe\u0308 Kelly today", func(out string) string {
			for _, w := range strings.Fields(out)[1:3] {
				rs := []rune(w)
				for _, r := range rs[1:] {
					if unicode.IsUpper(r) {
						return "case pattern shifted in " + w
					}
				}
			}
			return ""
		}},
		{"Jos\u00e9 Ortega called about the invoice.", "ping Jose\u0301 Ortega today", func(out string) string {
			for _, w := range strings.Fields(out)[1:3] {
				rs := []rune(w)
				for _, r := range rs[1:] {
					if unicode.IsUpper(r) {
						return "case pattern shifted in " + w
					}
				}
			}
			return ""
		}},
		{"Ruby Kelly called about the invoice.", "Ruby van der Berg called about the invoice.", func(out string) string {
			if !strings.Contains(out, " van der ") || hasWholeWord(out, "Berg") {
				return "particles not kept literally or surname left raw"
			}
			return ""
		}},
	}
	for _, c := range cases {
		path := filepath.Join(t.TempDir(), "v.db")
		e, v, _ := countingEngine(t, path)
		if _, err := e.NewSession("seed").Mask(ctx, c.seed); err != nil {
			t.Fatal(err)
		}
		s := e.NewSession("two")
		out, err := s.Mask(ctx, c.in)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range strings.Fields(out) {
			if strayCap(w) {
				t.Errorf("%q -> %q: stray final capital in %q", c.in, out, w)
			}
		}
		if c.check != nil {
			if msg := c.check(out); msg != "" {
				t.Errorf("%q -> %q: %s", c.in, out, msg)
			}
		}
		if back := s.Rehydrate(out); back != c.in {
			t.Errorf("%q batch restored as %q", c.in, back)
		}
		if back := spellingStream(s, out); back != c.in {
			t.Errorf("%q stream restored as %q", c.in, back)
		}
		v.Close()
		e2, v2, _ := countingEngine(t, path)
		if back := e2.NewSession("cli").Rehydrate(out); back != c.in {
			t.Errorf("%q fresh engine restored as %q", c.in, back)
		}
		v2.Close()
	}
}

func vaultRealLeft(t *testing.T, v *vault.Vault, out string) string {
	t.Helper()
	runs := strings.FieldsFunc(textutil.Fold(out), func(r rune) bool { return r < 'a' || r > 'z' })
	for _, r := range v.RecordsFrom(0) {
		if r.Kind != "person.given" && r.Kind != "person.family" {
			continue
		}
		real := strings.Map(func(c rune) rune {
			if c >= 'a' && c <= 'z' {
				return c
			}
			return -1
		}, textutil.Fold(r.Real))
		if len(real) < 3 || detect.IsNameParticle(real) {
			continue
		}
		for _, run := range runs {
			if strings.Contains(run, real) {
				return real
			}
		}
	}
	return ""
}

func TestPunctuatedSurnamesGlueAndPairEverywhere(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct{ seed, glued string }{
		{"Ruby O'Brien called about the invoice.", "rubyobrien"},
		{"Ruby D'Angelo called about the invoice.", "rubydangelo"},
		{"Ruby Smith-Jones called about the invoice.", "rubysmithjones"},
		{"Ruby St. John called about the invoice.", "rubystjohn"},
		{"Ruby De La Cruz called about the invoice.", "rubydelacruz"},
	} {
		for _, in := range []string{c.glued + " said hi", "ping " + titleWord(c.glued) + " today"} {
			for _, mode := range []string{"same", "new", "reopen", "anchored"} {
				path := filepath.Join(t.TempDir(), "v.db")
				e, v := modeEngine(t, mode, path)
				s := e.NewSession("one")
				full := in
				if mode == "anchored" {
					full = c.seed + " " + in
				} else if _, err := s.Mask(ctx, c.seed); err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "new":
					s = e.NewSession("two")
				case "reopen":
					v.Close()
					e, v, _ = countingEngine(t, path)
					s = e.NewSession("two")
				}
				out, err := s.Mask(ctx, full)
				if err != nil {
					t.Fatal(err)
				}
				if left := vaultRealLeft(t, v, out); left != "" {
					t.Errorf("%s: %q left %q inside %q", mode, full, left, out)
				}
				if back := s.Rehydrate(out); back != full {
					t.Errorf("%s: %q restored as %q", mode, full, back)
				}
				v.Close()
			}
		}
	}
}

func TestCompoundAndDanglingParticleNames(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		in   string
		raw  []string
		kept []string
	}{
		{"Ruby García de la Fuente called about the invoice.", []string{"García", "Fuente", "Ruby"}, []string{" de la "}},
		{"Javier Pérez de Cuéllar called about the invoice.", []string{"Pérez", "Cuéllar"}, []string{" de "}},
		{"José Ortega y Gasset wrote to us.", []string{"Ortega", "Gasset"}, []string{" y "}},
		{"Lucia Sánchez del Río called about the invoice.", []string{"Sánchez", "Río"}, []string{" del "}},
		{"Ruby van called about the invoice.", []string{"Ruby", " van "}, nil},
		{"Ruby De called about the invoice.", []string{"Ruby", " De "}, nil},
		{"Ruby Van called about the invoice.", []string{"Ruby", " Van "}, nil},
	} {
		e := newEngine(t, nil)
		s := e.NewSession("p")
		out, err := s.Mask(ctx, c.in)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range c.raw {
			if strings.Contains(out, w) {
				t.Errorf("%q -> %q: %s left raw", c.in, out, w)
			}
		}
		for _, w := range c.kept {
			if !strings.Contains(out, w) {
				t.Errorf("%q -> %q: particle %q not kept", c.in, out, w)
			}
		}
		if back := s.Rehydrate(out); back != c.in {
			t.Errorf("%q restored as %q", c.in, back)
		}
		if strings.Contains(c.in, "Fuente") {
			later := e.NewSession("later")
			for _, in := range []string{"Ask García de la Fuente today.", "de la Fuente said yes.", "Fuente said yes."} {
				if out, _ := later.Mask(ctx, in); strings.Contains(out, "Fuente") {
					t.Errorf("later %q -> %q", in, out)
				}
			}
		}
	}
}

func TestParticlesNeverBecomeVaultRecords(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"same", "new", "reopen", "anchored"} {
		path := filepath.Join(t.TempDir(), "v.db")
		e, v, _ := countingEngine(t, path)
		seed := "Ruby van der Berg called about the invoice."
		s := e.NewSession("one")
		in := "rubyvanderberg said hi"
		if mode == "anchored" {
			in = seed + " " + in
		} else if _, err := s.Mask(ctx, seed); err != nil {
			t.Fatal(err)
		}
		switch mode {
		case "new":
			s = e.NewSession("two")
		case "reopen":
			v.Close()
			e, v, _ = countingEngine(t, path)
			s = e.NewSession("two")
		}
		if _, err := s.Mask(ctx, in); err != nil {
			t.Fatal(err)
		}
		for _, r := range v.RecordsFrom(0) {
			if strings.HasPrefix(r.Kind, "person.") && detect.IsNameParticle(r.Real) {
				t.Errorf("%s: particle %q became a %s record", mode, r.Real, r.Kind)
			}
		}
		for _, plain := range []string{"the van der Waals force", "a van arrived"} {
			if out, _ := e.NewSession("x").Mask(ctx, plain); out != plain {
				t.Errorf("%s: %q became %q", mode, plain, out)
			}
		}
		v.Close()
	}
}

func TestNameFormsRespectWordBoundaries(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "v.db")
	e, v, _ := countingEngine(t, path)
	if _, err := e.NewSession("seed").Mask(ctx, "Zoe Kelly and Ilse Brandt called about the invoice."); err != nil {
		t.Fatal(err)
	}
	variants := []string{"zOE", "ZoE", "Zoë", "ZOË", "Ｚoe", "iLSE", "Ｉlse"}
	s := e.NewSession("x")
	forms := map[string]string{}
	for _, vr := range variants {
		in := "ping " + vr + " today"
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		forms[strings.TrimSuffix(strings.TrimPrefix(out, "ping "), " today")] = vr
	}
	rng := rand.New(rand.NewSource(7))
	chunked := func(s *Session, out string, random bool) string {
		r := s.NewStreamRehydrator(false)
		var b strings.Builder
		for i := 0; i < len(out); {
			n := 1
			if random {
				n = 1 + rng.Intn(5)
			}
			j := min(len(out), i+n)
			b.WriteString(r.Push(out[i:j]))
			i = j
		}
		b.WriteString(r.Flush())
		return b.String()
	}
	check := func(sess *Session, label string) {
		for f, real := range forms {
			if back := sess.Rehydrate("ping " + f + " today"); back != "ping "+real+" today" {
				t.Errorf("%s: form %q alone restored as %q, want %q", label, f, back, real)
			}
			for _, in := range []string{f + "velt said hi.", "the" + f + " x", "Mr " + f + "e called.", f + "ａ called.", f + "son and " + f + "a."} {
				if back := sess.Rehydrate(in); back != in {
					t.Errorf("%s batch: %q changed to %q", label, in, back)
				}
				for _, random := range []bool{false, true} {
					if back := chunked(sess, in, random); back != in {
						t.Errorf("%s stream(random=%v): %q changed to %q", label, random, in, back)
					}
				}
			}
		}
		for _, in := range []string{"Roosevelt said hi.", "Maarten and Aaron met.", "theRoose x"} {
			if back := sess.Rehydrate(in); back != in {
				t.Errorf("%s: %q changed to %q", label, in, back)
			}
		}
	}
	check(s, "same")
	v.Close()
	e2, v2, _ := countingEngine(t, path)
	defer v2.Close()
	check(e2.NewSession("cli"), "fresh engine")
}

func TestFin22NameShapes(t *testing.T) {
	ctx := context.Background()
	rows := []struct{ seed, in string }{
		{"", "Ruby St. John called about the invoice."},
		{"Ruby St. John called about the invoice.", "Ruby St. John called me."},
		{"Ruby St. John called about the invoice.", "Ask St. John today."},
		{"Ruby St. John called about the invoice.", "rubystjohn said hi"},
		{"Ruby St John called about the invoice.", "Ask St John today."},
		{"Ruby Ste. Marie called about the invoice.", "rubystemarie said hi"},
		{"Robert \"Bobby\" Kelly called about the invoice.", "bobbykelly said hi"},
		{"Robert \"Bobby\" Kelly called about the invoice.", "kellybobby said hi"},
		{"", "Ruby Van called about the invoice."},
		{"", "Ask Ruby Van today."},
		{"", "Ruby De called me."},
	}
	for _, apo := range []string{"'", "\u2019", "\u02bc", "-", ""} {
		rows = append(rows, struct{ seed, in string }{"Ruby O'Brien called about the invoice.", "rubyo" + apo + "brien said hi"})
		rows = append(rows, struct{ seed, in string }{"Ruby O'Brien called about the invoice.", "o" + apo + "brienruby said hi"})
		rows = append(rows, struct{ seed, in string }{"Ruby D'Angelo called about the invoice.", "rubyd" + apo + "angelo said hi"})
	}
	for _, c := range rows {
		for _, mode := range []string{"same", "new", "reopen", "anchored"} {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := modeEngine(t, mode, path)
			s := e.NewSession("one")
			full := c.in
			if c.seed != "" {
				if mode == "anchored" {
					full = c.seed + " " + c.in
				} else if _, err := s.Mask(ctx, c.seed); err != nil {
					t.Fatal(err)
				}
			}
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v, _ = countingEngine(t, path)
				s = e.NewSession("two")
			}
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			if c.seed == "" {
				e2, v2, _ := countingEngine(t, filepath.Join(t.TempDir(), "w.db"))
				e2.NewSession("probe").Mask(ctx, c.in)
				if left := vaultRealLeft(t, v2, out); left != "" {
					t.Errorf("%s: %q left %q inside %q", mode, full, left, out)
				}
				v2.Close()
			} else if left := vaultRealLeft(t, v, out); left != "" {
				t.Errorf("%s: %q left %q inside %q", mode, full, left, out)
			}
			if back := s.Rehydrate(out); back != full {
				t.Errorf("%s: %q restored as %q", mode, full, back)
			}
			v.Close()
		}
	}
	e := newEngine(t, nil)
	e.NewSession("seed").Mask(ctx, "Ruby Van called about the invoice. Ruby De called too.")
	for _, in := range []string{"a van arrived and de la rue too.", "We visited St. Louis and Main St. John called.", "Happy St. Patrick's Day.", "Rio de la Plata, Costa del Sol and the Ortega y Gasset Foundation.", "Banco de la Naci\u00f3n and Universidad de los Andes."} {
		if out, _ := e.NewSession("x").Mask(ctx, in); out != in {
			t.Errorf("%q became %q", in, out)
		}
	}
	if out, _ := newEngine(t, nil).NewSession("x").Mask(ctx, "Nguyen Van Minh called."); !strings.Contains(out, " Van ") {
		t.Errorf("Vietnamese middle marker lost: %q", out)
	}
}

func TestParticleHandlesEveryJoiner(t *testing.T) {
	ctx := context.Background()
	seeds := []struct {
		seed  string
		parts []string
	}{
		{"Ruby St. John called about the invoice.", []string{"ruby", "st", "john"}},
		{"Ruby Ste. Marie called about the invoice.", []string{"ruby", "ste", "marie"}},
		{"Ruby O'Brien called about the invoice.", []string{"ruby", "o", "brien"}},
		{"Ruby D'Angelo called about the invoice.", []string{"ruby", "d", "angelo"}},
		{"Ruby De La Cruz called about the invoice.", []string{"ruby", "de", "la", "cruz"}},
		{"Ruby van der Berg called about the invoice.", []string{"ruby", "van", "der", "berg"}},
		{"Ruby Van Damme called about the invoice.", []string{"ruby", "van", "damme"}},
		{"Ruby De-La Cruz called about the invoice.", []string{"ruby", "de-la", "cruz"}},
	}
	joiners := []string{".", "_", "-", ""}
	cases := []func(string) string{strings.ToLower, titleWord, strings.ToUpper}
	for _, sd := range seeds {
		mr := newModeRunner(t, sd.seed)
		given, surname := sd.parts[0], sd.parts[len(sd.parts)-1]
		particles := sd.parts[1 : len(sd.parts)-1]
		orders := [][]string{
			append(append([]string{given}, particles...), surname),
			append([]string{given, surname}, particles...),
			append(append(append([]string{}, particles...), surname), given),
			append(append([]string{surname}, particles...), given),
			append([]string{surname, given}, particles...),
			append(append(append([]string{}, particles...), given), surname),
		}
		joints := len(sd.parts) - 1
		total := 1
		for k := 0; k < joints; k++ {
			total *= len(joiners)
		}
		for oi, order := range orders {
			for combo := 0; combo < total; combo++ {
				for ci, cs := range cases {
					if combo != 0 && !raceKeep(t, combo+ci+oi, 6) {
						continue
					}
					var b strings.Builder
					c := combo
					for k, p := range order {
						if k > 0 {
							b.WriteString(joiners[c%len(joiners)])
							c /= len(joiners)
						}
						b.WriteString(cs(p))
					}
					handles := []string{b.String()}
					if combo == 0 {
						handles = append(handles, b.String()+"99", "7"+b.String(), cs(given)+cs(surname[:1])+"_"+cs(surname), cs(surname[:1])+cs(given)+"."+cs(surname))
					}
					for _, h := range handles {
						in := "ping " + h + " today"
						for _, mode := range modeNames {
							s, vv, full := mr.session(mode, in)
							out, err := s.Mask(ctx, full)
							if err != nil {
								t.Fatal(err)
							}
							if left := vaultRealLeft(t, vv, out); left != "" {
								t.Errorf("%s: %q left %q in %q", mode, full, left, out)
							}
							if back := s.Rehydrate(out); back != full {
								t.Errorf("%s: %q restored as %q", mode, full, back)
							}
							noParticleRecords(t, vv, mode+" "+full)
						}
					}
				}
			}
		}
		mr.close()
	}
}

var modeNames = []string{"same", "new", "reopen", "anchored"}

type modeRunner struct {
	t        *testing.T
	seed     string
	secret   []byte
	records  []vault.Record
	seedFile string
	last     []*vault.Vault
	fileDone map[string]bool
	n        int
}

type cloneBackend struct {
	mu      sync.Mutex
	secret  []byte
	records []vault.Record
}

func (c *cloneBackend) Load() ([]byte, []vault.Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.secret, append([]vault.Record(nil), c.records...), nil
}

func (c *cloneBackend) Update(fn func([]vault.Record) ([]vault.Record, error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	add, err := fn(nil)
	if err != nil {
		return err
	}
	c.records = append(c.records, add...)
	return nil
}

func (c *cloneBackend) Close() error { return nil }

func memEngine(t *testing.T) (*Engine, *vault.Vault) {
	t.Helper()
	v, err := vault.Open(vault.NewMemoryBackend(bytes.Repeat([]byte{9}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	return e, v
}

func newModeRunner(t *testing.T, seed string) *modeRunner {
	t.Helper()
	ctx := context.Background()
	mr := &modeRunner{t: t, seed: seed, fileDone: map[string]bool{}}
	mem, memV := memEngine(t)
	if _, err := mem.NewSession("seed").Mask(ctx, seed); err != nil {
		t.Fatal(err)
	}
	mr.secret, mr.records = memV.Secret(), memV.Records()
	mr.seedFile = filepath.Join(t.TempDir(), "seed.db")
	e, v, _ := countingEngine(t, mr.seedFile)
	if _, err := e.NewSession("seed").Mask(ctx, seed); err != nil {
		t.Fatal(err)
	}
	v.Close()
	return mr
}

func (mr *modeRunner) fileCopy() (*Engine, *vault.Vault) {
	mr.t.Helper()
	raw, err := os.ReadFile(mr.seedFile)
	if err != nil {
		mr.t.Fatal(err)
	}
	path := filepath.Join(mr.t.TempDir(), fmt.Sprintf("row%d.db", mr.n))
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		mr.t.Fatal(err)
	}
	return strictFileEngine(mr.t, path)
}

func (mr *modeRunner) clone() (*Engine, *vault.Vault) {
	mr.t.Helper()
	v, err := vault.Open(&cloneBackend{secret: mr.secret, records: append([]vault.Record(nil), mr.records...)})
	if err != nil {
		mr.t.Fatal(err)
	}
	e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
	if err != nil {
		mr.t.Fatal(err)
	}
	return e, v
}

func (mr *modeRunner) session(mode, in string) (*Session, *vault.Vault, string) {
	mr.t.Helper()
	mr.close()
	mr.n++
	name := fmt.Sprintf("row%d", mr.n)
	file := mode == "reopen" || !mr.fileDone[mode]
	mr.fileDone[mode] = true
	var e *Engine
	var v *vault.Vault
	switch {
	case mode == "anchored" && file:
		e, v = strictFileEngine(mr.t, filepath.Join(mr.t.TempDir(), name+".db"))
	case mode == "anchored":
		e, v = memEngine(mr.t)
	case file:
		e, v = mr.fileCopy()
	default:
		e, v = mr.clone()
	}
	mr.last = append(mr.last, v)
	s := e.NewSession(name)
	switch mode {
	case "same":
		if _, err := s.Mask(context.Background(), mr.seed); err != nil {
			mr.t.Fatal(err)
		}
	case "anchored":
		return s, v, mr.seed + " " + in
	}
	return s, v, in
}

func (mr *modeRunner) close() {
	for _, v := range mr.last {
		v.Close()
	}
	mr.last = nil
}

func noParticleRecords(t *testing.T, v *vault.Vault, where string) {
	t.Helper()
	for _, r := range v.RecordsFrom(0) {
		if strings.HasPrefix(r.Kind, "person.") && r.Kind != "person.joined" && detect.IsNameParticle(r.Real) {
			t.Errorf("%s: recorded particle %q", where, r.Real)
		}
	}
}

func TestQuotedNicknameKeepsSurnameRole(t *testing.T) {
	ctx := context.Background()
	for _, seed := range []string{"Robert \"Bobby\" Kelly called about the invoice.", "Robert (Bobby) Kelly called about the invoice."} {
		e := newEngine(t, nil)
		if _, err := e.NewSession("seed").Mask(ctx, seed); err != nil {
			t.Fatal(err)
		}
		if _, ok := e.Vault().Get("person.family", "Kelly"); !ok {
			t.Errorf("%q: Kelly not recorded as a surname", seed)
		}
		for _, in := range []string{"bobbykelly said hi", "Kelly said hi"} {
			for _, anchored := range []bool{false, true} {
				s := e.NewSession("x")
				full := in
				if anchored {
					s = newEngine(t, nil).NewSession("a")
					full = seed + " " + in
				}
				out, _ := s.Mask(ctx, full)
				if strings.Contains(strings.ToLower(out), "kelly") {
					t.Errorf("%q: %q -> %q", seed, full, out)
				}
				if back := s.Rehydrate(out); back != full {
					t.Errorf("%q: %q restored as %q", seed, full, back)
				}
			}
		}
	}
}

func TestFunctionWordAndParticleSurnameRows(t *testing.T) {
	ctx := context.Background()
	run := func(seed, in string, check func(mode, out string)) {
		for _, mode := range []string{"same", "new", "reopen", "anchored"} {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := modeEngine(t, mode, path)
			s := e.NewSession("one")
			full := in
			if mode == "anchored" {
				full = seed + " " + in
			} else if _, err := s.Mask(ctx, seed); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v, _ = countingEngine(t, path)
				s = e.NewSession("two")
			}
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			if mode == "anchored" {
				out = strings.TrimPrefix(out[strings.Index(out, ".")+1:], " ")
			}
			check(mode, out)
			if back := s.Rehydrate(out); mode != "anchored" && back != full {
				t.Errorf("%s: %q restored as %q", mode, full, back)
			}
			v.Close()
		}
	}
	verbs := []struct{ seed, in, word string }{
		{"Will Smith called about the invoice.", "I think Smith will call.", "will"},
		{"Will Smith called about the invoice.", "Smith will call later.", "will"},
		{"Will Smith called about the invoice.", "smith will call.", "will"},
		{"May Kelly called about the invoice.", "Kelly may call.", "may"},
		{"May Kelly called about the invoice.", "I think Kelly may come.", "may"},
	}
	for _, c := range verbs {
		run(c.seed, c.in, func(mode, out string) {
			if !hasWholeWord(out, c.word) {
				t.Errorf("%s: %q lost the verb: %q", mode, c.in, out)
			}
			if f := textutil.Fold(out); strings.Contains(f, "smith") || strings.Contains(f, "kelly") {
				t.Errorf("%s: %q left the surname: %q", mode, c.in, out)
			}
		})
	}
	for _, in := range []string{"Smith, Will called.", "smith, will called.", "Smith Will called.", "smithwill said hi", "smith.will said hi", "smith_will said hi", "will smith called.", "Will Smith called."} {
		run("Will Smith called about the invoice.", in, func(mode, out string) {
			if f := textutil.Fold(out); strings.Contains(f, "smith") || strings.Contains(f, "will") {
				t.Errorf("%s: %q -> %q", mode, in, out)
			}
		})
	}
	raw := []struct{ seed, in string }{
		{"Ruby Van called about the invoice.", "Van Systems shipped it."},
		{"Ruby Van called about the invoice.", "Van Rentals Inc delivered."},
		{"Ruby De called about the invoice.", "De Laurentiis Studios released it."},
		{"Ruby De called about the invoice.", "De Niro starred."},
		{"Ruby De called about the invoice.", "De Gaulle spoke."},
		{"Ruby Da called about the invoice.", "Da Vinci painted it."},
		{"Ruby La called about the invoice.", "La Paz is high."},
		{"Ruby Le called about the invoice.", "Le Monde printed it."},
	}
	for _, c := range raw {
		run(c.seed, c.in, func(mode, out string) {
			if mode != "anchored" && out != c.in {
				t.Errorf("%s: %q became %q", mode, c.in, out)
			}
		})
	}
	for _, in := range []string{"Van called me yesterday.", "Ask Van.", "Thanks, Van."} {
		run("Ruby Van called about the invoice.", in, func(mode, out string) {
			if hasWholeWord(out, "Van") {
				t.Errorf("%s: %q -> %q", mode, in, out)
			}
		})
	}
}

func TestFin23NameRows(t *testing.T) {
	ctx := context.Background()
	modes := []string{"same", "new", "reopen", "anchored"}
	run := func(seed, in string, check func(mode, out string)) {
		for _, mode := range modes {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := modeEngine(t, mode, path)
			s := e.NewSession("one")
			full := in
			if mode == "anchored" {
				full = seed + "\n\n" + in
			} else if _, err := s.Mask(ctx, seed); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v, _ = countingEngine(t, path)
				s = e.NewSession("two")
			}
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			check(mode, out[strings.LastIndex(out, "\n")+1:])
			if back := s.Rehydrate(out); back != full {
				t.Errorf("%s: %q restored as %q", mode, full, back)
			}
			v.Close()
		}
	}
	for _, sep := range []string{"`", "‘", "´", "ʹ", "＇", "′", "^", "~", "·"} {
		in := "rubyo" + sep + "brien said hi"
		run("Ruby O'Brien called about the invoice.", in, func(mode, out string) {
			if f := textutil.Fold(out); strings.Contains(f, "ruby") || strings.Contains(f, "brien") {
				t.Errorf("%s: %q -> %q", mode, in, out)
			}
		})
	}
	people := "Morgan Lee called about the invoice. Taylor Moss called too. Robert \"Bobby\" Kelly called as well."
	words := []string{"morgan", "lee", "taylor", "moss", "bobby", "kelly"}
	pairs := [][2]string{{"taylor", "morgan"}, {"lee", "moss"}, {"taylor", "lee"}, {"bobby", "morgan"}, {"bobby", "moss"}, {"kelly", "taylor"}}
	for _, p := range pairs {
		for _, in := range []string{p[0] + " " + p[1] + " called.", p[1] + " " + p[0] + " called."} {
			run(people, in, func(mode, out string) {
				for _, tok := range strings.Fields(strings.TrimSuffix(out, " called.")) {
					for _, w := range words {
						if strings.EqualFold(tok, w) {
							t.Errorf("%s: %q left %q in %q", mode, in, tok, out)
						}
					}
				}
			})
		}
	}
	for _, in := range []string{"Van drivers were late.", "The Van is parked.", "Caravan Van rentals.", "Van fleet is parked outside."} {
		run("Ruby Van called about the invoice.", in, func(mode, out string) {
			if mode != "anchored" && out != in {
				t.Errorf("%s: %q became %q", mode, in, out)
			}
		})
	}
	for _, in := range []string{"De facto, it works.", "De Niro starred.", "De la Soul rapped."} {
		run("Ruby De called about the invoice.", in, func(mode, out string) {
			if mode != "anchored" && out != in {
				t.Errorf("%s: %q became %q", mode, in, out)
			}
		})
	}
	for _, in := range []string{"Ask Ruby Van today.", "Ruby Van called again.", "Mr. Van called."} {
		run("Ruby Van called about the invoice.", in, func(mode, out string) {
			if hasWholeWord(out, "Van") {
				t.Errorf("%s: %q -> %q", mode, in, out)
			}
		})
	}
	run("Ruby St. John called about the invoice.", "Visit St. John's church.", func(mode, out string) {
		if !strings.Contains(out, "'s church") || hasWholeWord(out, "John") {
			t.Errorf("%s: possessive or name wrong in %q", mode, out)
		}
	})
}

func TestCamelCaseHandlesRestoreExactly(t *testing.T) {
	ctx := context.Background()
	orders := [][]string{
		{"RubyKelly said hi", "%s said hi", "Ping %s now", "%s"},
		{"%s said hi", "RubyKelly said hi", "%s said hi"},
		{"%[2]s %[3]s said hi", "%s said hi", "RubyKelly said hi"},
	}
	for oi, order := range orders {
		path := filepath.Join(t.TempDir(), "v.db")
		e, v, _ := countingEngine(t, path)
		seed, err := e.NewSession("seed").Mask(ctx, "Ruby Kelly called about the invoice.")
		if err != nil {
			t.Fatal(err)
		}
		words := strings.Fields(seed)
		g, f := words[0], words[1]
		typed := g + f
		var outs, ins []string
		for _, tmpl := range order {
			in := fmt.Sprintf(tmpl, typed, g, f)
			s := e.NewSession("two")
			out, err := s.Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if back := s.Rehydrate(out); back != in {
				t.Errorf("order %d: %q -> %q batch restored as %q", oi, in, out, back)
			}
			if back := spellingStream(s, out); back != in {
				t.Errorf("order %d: %q -> %q stream restored as %q", oi, in, out, back)
			}
			if strings.Contains(in, "RubyKelly") && strings.Contains(out, "RubyKelly") {
				t.Errorf("order %d: %q left the real handle raw: %q", oi, in, out)
			}
			ins, outs = append(ins, in), append(outs, out)
		}
		v.Close()
		e2, v2, _ := countingEngine(t, path)
		for i, out := range outs {
			if back := e2.NewSession("cli").Rehydrate(out); back != ins[i] {
				t.Errorf("order %d: %q fresh engine restored as %q", oi, ins[i], back)
			}
		}
		v2.Close()
	}
}

func TestFirstSightParticlesAndCamelParts(t *testing.T) {
	ctx := context.Background()
	for _, seed := range []string{"", "Ruby Van called about the invoice.", "Van called me yesterday.", "Ruby van called about the invoice."} {
		for _, in := range []string{"Vincent van Gogh painted it.", "Ruby van der Berg called.", "Ludwig van Beethoven wrote it."} {
			e := newEngine(t, nil)
			if seed != "" {
				if _, err := e.NewSession("seed").Mask(ctx, seed); err != nil {
					t.Fatal(err)
				}
			}
			s := e.NewSession("x")
			out, err := s.Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, " van ") || len(strings.Fields(out)) != len(strings.Fields(in)) {
				t.Errorf("seed %q: %q -> %q: particle not kept", seed, in, out)
			}
			if back := s.Rehydrate(out); back != in {
				t.Errorf("seed %q: %q restored as %q", seed, in, back)
			}
		}
	}
	for _, first := range []string{"ping rubykelly today", "ping RubyKelly today", "ping rubyKelly today"} {
		path := filepath.Join(t.TempDir(), "v.db")
		e, v, _ := countingEngine(t, path)
		seed, err := e.NewSession("seed").Mask(ctx, "Ruby Kelly called about the invoice.")
		if err != nil {
			t.Fatal(err)
		}
		w := strings.Fields(seed)
		g, f := w[0], w[1]
		camel := g + f
		if first == "ping RubyKelly today" {
			camel = g + strings.ToLower(f)
		}
		want := map[string]string{
			"ping RubyKelly today": "ping " + camel + " today",
			"ping rubyKelly today": "ping " + strings.ToLower(g) + f + " today",
			"ping rubykelly today": "ping " + strings.ToLower(g+f) + " today",
		}
		var outs, ins []string
		for _, in := range []string{first, "ping RubyKelly today", "ping rubyKelly today", "ping rubykelly today"} {
			s := e.NewSession("two")
			out, err := s.Mask(ctx, in)
			if err != nil {
				t.Fatal(err)
			}
			if out != want[in] {
				t.Errorf("after %q: %q -> %q, want %q", first, in, out, want[in])
			}
			if back := spellingStream(s, out); back != in {
				t.Errorf("after %q: %q stream restored as %q", first, in, back)
			}
			outs, ins = append(outs, out), append(ins, in)
		}
		v.Close()
		e2, v2, _ := countingEngine(t, path)
		for i, out := range outs {
			if back := e2.NewSession("cli").Rehydrate(out); back != ins[i] {
				t.Errorf("after %q: %q fresh engine restored as %q", first, ins[i], back)
			}
		}
		v2.Close()
	}
}

func TestTrailingParticleSurnamesAndHandleRows(t *testing.T) {
	ctx := context.Background()
	rows := []struct {
		seed  string
		ins   []string
		words []string
	}{
		{"Ruby John Van called about the invoice.", []string{"Ruby John Van called me.", "Ask Ruby John Van today."}, []string{"ruby", "john", "van"}},
		{"Maria Luisa De called about the invoice.", []string{"Maria Luisa De called me.", "Ask Maria Luisa De today."}, []string{"maria", "luisa", "de"}},
		{"Anna Maria Van called about the invoice.", []string{"Anna Maria Van called me."}, []string{"anna", "maria", "van"}},
		{"Ruby De-La Cruz called about the invoice.", []string{"rubydelacruz said hi", "Ask De-La Cruz today."}, []string{"ruby", "cruz", "rubydelacruz"}},
		{"Da-bit Lee called about the invoice.", []string{"dabit lee called.", "da-bit lee called.", "dabit, lee called.", "lee dabit called."}, []string{"dabit", "bit", "lee"}},
		{"François Lefèvre called about the invoice.", []string{"mail f.lefevre today", "mail francois.l today", "lefevre called.", "francois called."}, []string{"francois", "lefevre"}},
	}
	for _, r := range rows {
		for _, in := range r.ins {
			for _, mode := range []string{"same", "new", "reopen", "anchored"} {
				path := filepath.Join(t.TempDir(), "v.db")
				e, v := modeEngine(t, mode, path)
				s := e.NewSession("one")
				full := in
				if mode == "anchored" {
					full = r.seed + " " + in
				} else if _, err := s.Mask(ctx, r.seed); err != nil {
					t.Fatal(err)
				}
				switch mode {
				case "new":
					s = e.NewSession("two")
				case "reopen":
					v.Close()
					e, v, _ = countingEngine(t, path)
					s = e.NewSession("two")
				}
				out, err := s.Mask(ctx, full)
				if err != nil {
					t.Fatal(err)
				}
				for _, tok := range strings.FieldsFunc(textutil.Fold(out), func(c rune) bool { return c < 'a' || c > 'z' }) {
					for _, w := range r.words {
						if tok == w {
							t.Errorf("%s: %q left %q in %q", mode, full, w, out)
						}
					}
				}
				if back := s.Rehydrate(out); back != full {
					t.Errorf("%s: %q restored as %q", mode, full, back)
				}
				v.Close()
			}
		}
	}
}

var fileModeSeen sync.Map

func modeEngine(t *testing.T, mode, path string) (*Engine, *vault.Vault) {
	t.Helper()
	if _, seen := fileModeSeen.LoadOrStore(t.Name()+"|"+mode, true); mode == "reopen" || !seen {
		return strictFileEngine(t, path)
	}
	return memEngine(t)
}

func TestTrailingStreetAbbreviationStaysAsTyped(t *testing.T) {
	ctx := context.Background()
	rows := []struct {
		seed, in, kept string
		masked         []string
	}{
		{"Ruby John St called about the invoice.", "Ruby John St called about the invoice.", "St", []string{"Ruby", "John"}},
		{"Maria Luisa St called about the invoice.", "Maria Luisa St called me.", "St", []string{"Maria", "Luisa"}},
		{"Ruby John Ste. called about the invoice.", "Ruby John Ste. called me.", "Ste", []string{"Ruby", "John"}},
		{"Ruby St called about the invoice.", "Ruby St called.", "St", []string{"Ruby"}},
		{"Ruby St. called about the invoice.", "Ruby St. called.", "St", []string{"Ruby"}},
		{"Ruby St. John called about the invoice.", "Ruby St. John called.", "St", []string{"Ruby", "John"}},
	}
	for _, r := range rows {
		for _, mode := range modeNames {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := modeEngine(t, mode, path)
			s := e.NewSession("one")
			full := r.in
			if mode == "anchored" {
				full = r.seed + " " + r.in
			} else if _, err := s.Mask(ctx, r.seed); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v = strictFileEngine(t, path)
				s = e.NewSession("two")
			}
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Count(out, " "+r.kept) != strings.Count(full, " "+r.kept) {
				t.Errorf("%s: %q lost %q: %q", mode, full, r.kept, out)
			}
			for _, w := range r.masked {
				if hasWholeWord(out, w) {
					t.Errorf("%s: %q left %q in %q", mode, full, w, out)
				}
			}
			if back := s.Rehydrate(out); back != full {
				t.Errorf("%s: %q restored as %q", mode, full, back)
			}
			v.Close()
		}
	}
}

func TestHeldParticleRealsMaskAtSpanEdges(t *testing.T) {
	ctx := context.Background()
	stDict := "value,type,aliases\nRuby St,person,\n"
	rows := []struct {
		seed, dict, in string
		gone           []string
		kept           []string
	}{
		{"Van Ruby called about the invoice.", "", "van Ruby called me.", []string{"van", "Ruby"}, nil},
		{"Van Ruby called about the invoice.", "", "Ask van Ruby today.", []string{"van", "Ruby"}, nil},
		{"Van Ruby called about the invoice.", "", "a van arrived.", nil, []string{"van"}},
		{"Kelly De called about the invoice.", "", "De-Kelly called me.", []string{"De", "Kelly"}, nil},
		{"Kelly De called about the invoice.", "", "de-Kelly called me.", []string{"de", "Kelly"}, nil},
		{"Kelly De called about the invoice.", "", "de Kelly called me.", []string{"de", "Kelly"}, nil},
		{"Kelly De called about the invoice.", "", "de la rue is a street.", nil, []string{"de", "la", "rue"}},
		{"Ruby St called about the invoice.", stDict, "Ruby John St called.", []string{"Ruby", "St"}, nil},
		{"Ruby John called about the invoice.", "", "Ruby John St called.", []string{"Ruby", "John"}, []string{"St"}},
		{"Ruby Kelly called about the invoice. Anna Moss called too. ruby_k and kelly.r and r.kelly wrote too.", "", "ruby_k99 said hi", []string{"ruby"}, nil},
		{"Ruby Kelly called about the invoice. Anna Moss called too. ruby_k and kelly.r and r.kelly wrote too.", "", "rubyk_anna said hi", []string{"ruby", "anna"}, nil},
		{"Ruby Kelly called about the invoice. Anna Moss called too. ruby_k and kelly.r and r.kelly wrote too.", "", "kruby_anna said hi", []string{"ruby", "anna"}, nil},
		{"Ruby Kelly called about the invoice. Anna Moss called too. ruby_k and kelly.r and r.kelly wrote too.", "", "rkelly.anna said hi", []string{"kelly", "anna"}, nil},
		{"Ruby Kelly called about the invoice. Anna Moss called too. ruby_k and kelly.r and r.kelly wrote too.", "", "x_ruby_k_anna said hi", []string{"ruby", "anna"}, nil},
		{"Ruby Kelly called about the invoice. Anna Moss called too. ruby_k and kelly.r and r.kelly wrote too.", "", "kelly.r_anna said hi", []string{"kelly", "anna"}, nil},
	}
	letters := func(out string) []string {
		return strings.FieldsFunc(out, func(c rune) bool { return !unicode.IsLetter(c) })
	}
	for _, r := range rows {
		for _, mode := range modeNames {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := modeEngine(t, mode, path)
			if r.dict != "" {
				d, err := detect.ParseDictionaryCSV(strings.NewReader(r.dict))
				if err != nil {
					t.Fatal(err)
				}
				if e, err = New(Options{Vault: v, Policy: DefaultPolicy(), Dictionary: d}); err != nil {
					t.Fatal(err)
				}
			}
			s := e.NewSession("one")
			full := r.in
			if mode == "anchored" {
				full = r.seed + " " + r.in
			} else if _, err := s.Mask(ctx, r.seed); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v = strictFileEngine(t, path)
				if r.dict != "" {
					d, _ := detect.ParseDictionaryCSV(strings.NewReader(r.dict))
					e, _ = New(Options{Vault: v, Policy: DefaultPolicy(), Dictionary: d})
				}
				s = e.NewSession("two")
			}
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			tail := out
			if mode == "anchored" {
				tail = out[len(out)-len(r.in)-8:]
				tail = tail[strings.Index(tail, " ")+1:]
			}
			for _, w := range letters(tail) {
				for _, g := range r.gone {
					if w == g {
						t.Errorf("%s: %q left %q in %q", mode, full, g, out)
					}
				}
			}
			for _, k := range r.kept {
				if !hasWholeWord(tail, k) {
					t.Errorf("%s: %q lost %q in %q", mode, full, k, out)
				}
			}
			if back := s.Rehydrate(out); back != full {
				t.Errorf("%s: %q restored as %q", mode, full, back)
			}
			v.Close()
		}
	}
}

var raceSampleSeed = func() int {
	if v := os.Getenv("OUTWORLD_RACE_SAMPLE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return int(time.Now().UnixNano() % 9973)
}()

var raceSampleLogged sync.Map

func raceKeep(t *testing.T, i, every int) bool {
	if !scaletest.Race {
		return true
	}
	if _, seen := raceSampleLogged.LoadOrStore(t.Name(), true); !seen {
		t.Logf("race sample seed %d (rerun with OUTWORLD_RACE_SAMPLE=%d)", raceSampleSeed, raceSampleSeed)
	}
	return (i+raceSampleSeed)%every == 0
}

func openTestVault(path string, key []byte) (*vault.Vault, error) {
	b, err := vault.NewFileBackendWithSecret(path, key, key)
	if err != nil {
		return nil, err
	}
	return vault.Open(b)
}

func TestStandInsNeverContainTheRealRootUnderAnySecret(t *testing.T) {
	ctx := context.Background()
	seeds := []string{
		"Ruby D'Angelo called about the invoice.",
		"Ruby O'Brien called about the invoice.",
		"Ruby St. John called about the invoice.",
		"Ruby Ste. Marie called about the invoice.",
		"Ruby De La Cruz called about the invoice.",
		"Ruby De-La Cruz called about the invoice.",
		"Ruby van der Berg called about the invoice.",
		"Ruby Van Damme called about the invoice.",
		"Ruby Smith-Jones called about the invoice.",
		"Christina Berger called about the invoice.",
		"Angelina Morgan called about the invoice.",
	}
	particles := []string{"della", "delle", "del", "dos", "das", "van", "von", "der", "den", "ter", "ten", "ste", "st", "de", "la", "le", "da", "di", "du", "o", "d", "l"}
	roots := func(real string) []string {
		var out []string
		parts := strings.FieldsFunc(real, func(r rune) bool { return !unicode.IsLetter(r) })
		for _, base := range append([]string{real}, parts...) {
			r := nameRootKey(base)
			for len(r) >= 5 {
				out = append(out, r)
				cut := 0
				for _, p := range particles {
					if strings.HasPrefix(r, p) && len(r)-len(p) >= 5 {
						cut = len(p)
						break
					}
				}
				if cut == 0 {
					break
				}
				r = r[cut:]
			}
		}
		return out
	}
	secrets := 120
	if scaletest.Race {
		secrets = 12
	}
	reals := []struct{ kind, real string }{
		{"person.family", "D'Angelo"}, {"person.family", "O'Brien"}, {"person.family", "John"}, {"person.family", "Marie"},
		{"person.family", "Cruz"}, {"person.family", "Berg"}, {"person.family", "Damme"}, {"person.family", "Smith-Jones"},
		{"person.family", "Berger"}, {"person.family", "Morgan"}, {"person.given", "Christina"}, {"person.given", "Angelina"},
	}
	for i := 0; i < secrets; i++ {
		v, err := vault.Open(&cloneBackend{secret: bytes.Repeat([]byte{byte(i), byte(i * 7), byte(i * 13), byte(i * 31)}, 8)})
		if err != nil {
			t.Fatal(err)
		}
		e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range reals {
			for attempt := 0; attempt < 64; attempt++ {
				var c string
				if r.kind == "person.given" {
					c = notContaining(e.gen.GivenNameFor(r.real, "", attempt), r.real)
				} else {
					c = notContaining(e.gen.FamilyName(r.real, attempt), r.real)
				}
				for _, root := range roots(r.real) {
					if c != "" && strings.Contains(nameRootKey(c), root) {
						t.Errorf("secret %d attempt %d: %q candidate %q contains %q", i, attempt, r.real, c, root)
					}
				}
			}
		}
	}
	for i := 0; i < secrets; i++ {
		secret := bytes.Repeat([]byte{byte(i), byte(i * 7), byte(i * 13), byte(i * 31)}, 8)
		for _, seed := range seeds {
			v, err := vault.Open(&cloneBackend{secret: secret})
			if err != nil {
				t.Fatal(err)
			}
			e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := e.NewSession("s").Mask(ctx, seed); err != nil {
				t.Fatal(err)
			}
			for _, r := range v.Records() {
				if r.Kind != "person.given" && r.Kind != "person.family" {
					continue
				}
				sk := nameRootKey(r.Surrogate)
				for _, root := range roots(r.Real) {
					if strings.Contains(sk, root) {
						t.Errorf("secret %d: %q stand-in %q contains %q", i, r.Real, r.Surrogate, root)
					}
				}
			}
		}
	}
}

func TestMutantKillingNameRows(t *testing.T) {
	ctx := context.Background()
	handles := "Ruby Kelly called about the invoice. ruby_k and kelly.r and r.kelly wrote too."
	rows := []struct {
		seed, in string
		gone     []string
		kept     []string
	}{
		{handles, "Ruby_K_ called.", []string{"Ruby"}, []string{"K"}},
		{handles, "ruby_k99 said hi", []string{"ruby"}, nil},
		{"Ruby van der Berg called about the invoice.", "Maria-Luisa-De called me.", []string{"Maria", "Luisa", "De"}, nil},
		{"Ruby van der Berg called about the invoice.", "We met Ruby van der Berg yesterday.", []string{"Ruby", "Berg"}, []string{"van", "der"}},
		{"Ruby van Berg called about the invoice.", "ask van Berg today.", []string{"Berg"}, []string{"van"}},
		{"Ana De La Cruz called about the invoice.", "De La Cruz said hi.", []string{"Cruz"}, []string{"De", "La"}},
		{"Ruby Kelly called about the invoice.", "Mr. van Berg called me.", []string{"Berg"}, []string{"van"}},
		{"Ruby Kelly called about the invoice.", "Mr. Van Der Berg called me.", []string{"Berg"}, []string{"Van", "Der"}},
		{"Ruby Kelly called about the invoice.", "Dr. De La Cruz called me.", []string{"Cruz"}, []string{"De", "La"}},
	}
	for _, r := range rows {
		for _, mode := range modeNames {
			path := filepath.Join(t.TempDir(), "v.db")
			e, v := modeEngine(t, mode, path)
			s := e.NewSession("one")
			full := r.in
			if mode == "anchored" {
				full = r.seed + "\n\n" + r.in
			} else if _, err := s.Mask(ctx, r.seed); err != nil {
				t.Fatal(err)
			}
			switch mode {
			case "new":
				s = e.NewSession("two")
			case "reopen":
				v.Close()
				e, v = strictFileEngine(t, path)
				s = e.NewSession("two")
			}
			out, err := s.Mask(ctx, full)
			if err != nil {
				t.Fatal(err)
			}
			tail := out[strings.LastIndex(out, "\n")+1:]
			for _, g := range r.gone {
				if hasWholeWord(tail, g) || hasWholeWord(tail, strings.ToLower(g)) {
					t.Errorf("%s: %q left %q in %q", mode, full, g, out)
				}
			}
			for _, k := range r.kept {
				if !hasWholeWord(tail, k) {
					t.Errorf("%s: %q lost %q in %q", mode, full, k, out)
				}
			}
			if back := s.Rehydrate(out); back != full {
				t.Errorf("%s: %q restored as %q", mode, full, back)
			}
			v.Close()
		}
	}
}
