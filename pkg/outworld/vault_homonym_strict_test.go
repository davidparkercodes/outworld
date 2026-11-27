package outworld

import (
	"bytes"
	"context"
	"strings"
	"sync"
	"testing"
	"unicode"

	"github.com/davidparkercodes/outworld/pkg/vault"
)

var strictPersonTemplates = []string{
	"My daughter %s has a test on Friday.",
	"I drove %s to the site this morning.",
	"We used %s as our babysitter.",
	"Pay %s 20 dollars for lunch.",
	"Give %s 5 minutes.",
	"I learned %s was in the hospital.",
	"Can %s review the code?",
	"Did %s see the error?",
	"Send the docs to %s by Friday.",
	"Thanks to %s the build is green.",
	"%s is our release manager.",
	"%s broke the build again.",
	"%s from the build team called.",
	"%s has a test tomorrow at school.",
	"%s and %p are coming to dinner.",
	"%s or %p can pick up the kids.",
	"We use %s for babysitting on Fridays.",
	"We are using %s to cover the night shift.",
	"We switched from %s to Maria as our accountant.",
	"We will deploy %s to the client site next week.",
	"%s 3.30 pickup from school.",
	"%s 9.15 dentist",
	"%s/Maria to cover the Friday shift.",
	"Sarah-%s called about the invoice.",
	"%s test results came back from the lab.",
	"%s job offer came in today.",
	"%s client called again.",
	"%s release from the hospital is Friday.",
	"I put the gift in %s.",
	"%s app review went well.",
	"%s and Python are my two cats.",
	"%s and Maria built a treehouse.",
	"I need to restart %s on her medication.",
	"%s image in the yearbook looks great.",
	"%s log of hours is attached.",
	"%s docs are in the folder, the passport and birth certificate.",
	"%s server shift starts at 5.",
	"%s error on the timesheet needs fixing.",
	"Call %s v2 of the contract is ready.",
	"Reminder: %s code for the gate is 4411.",
	"%s schema homework is due.",
	"%s cluster of friends is coming over.",
	"%s node on the family tree needs a photo.",
	"%s version of events differs.",
	"%s config of the wedding seating is done.",
	"%s script for the school play is finished.",
	"%s package arrived at the door.",
	"%s build is stocky and strong.",
	"%s and Go are my favourite people.",
	"in %s.",
	"I believe in %s.",
	"I am learning %s by heart.",
	"Learn %s, she is our new manager.",
	"%s-Smith called about the invoice.",
	"Talk to %s_ tomorrow.",
}

var techGivenNames28 = strings.Fields("Ada Adam Apollo Athena Atlas Aurora Bert Cassandra Crystal Elmo Ernie Haskell Hermes Homer Hugo Jade Jasmine Julia Jupiter Mercury Neptune Pascal Pluto Ruby Saturn Travis Venus Watson")

func strictFileEngine(t *testing.T, path string) (*Engine, *vault.Vault) {
	t.Helper()
	v, err := openTestVault(path, bytes.Repeat([]byte{9}, 32))
	if err != nil {
		t.Fatal(err)
	}
	e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	return e, v
}

func hasWholeWord(text, w string) bool {
	for _, f := range strings.FieldsFunc(text, func(r rune) bool { return !unicode.IsLetter(r) }) {
		if f == w {
			return true
		}
	}
	return false
}

func strictMaskModes(t *testing.T, seed, in string, check func(mode, out, back string)) {
	t.Helper()
	mr := cachedModeRunner(t, seed)
	for _, mode := range []string{"same", "new", "reopen"} {
		s, _, _ := mr.session(mode, in)
		out, err := s.Mask(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		check(mode, out, s.Rehydrate(out))
	}
}

type runnerKey struct {
	t    *testing.T
	seed string
}

var (
	runnersMu sync.Mutex
	runners   = map[runnerKey]*modeRunner{}
)

func cachedModeRunner(t *testing.T, seed string) *modeRunner {
	t.Helper()
	runnersMu.Lock()
	defer runnersMu.Unlock()
	k := runnerKey{t, seed}
	if mr, ok := runners[k]; ok {
		return mr
	}
	mr := newModeRunner(t, seed)
	runners[k] = mr
	t.Cleanup(func() {
		runnersMu.Lock()
		delete(runners, k)
		runnersMu.Unlock()
		mr.close()
	})
	return mr
}

func TestVaultKnownHomonymMasksEverywhereButStrictExceptions(t *testing.T) {
	for _, n := range techGivenNames28 {
		other := "Ada"
		if n == "Ada" {
			other = "Julia"
		}
		mr := newModeRunner(t, n+" Smith called about the invoice.")
		for ti, tpl := range strictPersonTemplates {
			if !raceKeep(t, ti+len(n), 4) {
				continue
			}
			in := strings.ReplaceAll(strings.ReplaceAll(tpl, "%s", n), "%p", other)
			for _, mode := range []string{"same", "new", "reopen"} {
				s, _, _ := mr.session(mode, in)
				out, err := s.Mask(context.Background(), in)
				if err != nil {
					t.Fatal(err)
				}
				if hasWholeWord(out, n) {
					t.Errorf("%s %s: %q left raw as %q", mode, n, in, out)
				}
				if back := s.Rehydrate(out); back != in && !strings.Contains(out, "[redacted") {
					t.Errorf("%s %s: %q restored as %q", mode, n, in, back)
				}
			}
		}
		mr.close()
	}
}

func TestVaultKnownHomonymStrictKeeps(t *testing.T) {
	ctx := context.Background()
	keeps := []string{
		"Run `Ruby --version` first.",
		"```\nRuby.run(1)\n```",
		"Edit my_ruby_script.rb now.",
		"See ruby-lang for the docs.",
		"Call `Config.Ruby` now.",
		"`Module.Ruby` is loaded.",
		"Use `Ruby::Version` here.",
		"Open ruby.yml please.",
		"Look in `/usr/lib/Ruby/core` for it.",
		"See `https://example.com/Ruby/docs` today.",
		"install the ruby gem",
	}
	for _, in := range keeps {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, "Ruby Smith called about the invoice.")
		if out, _ := e.NewSession("later").Mask(ctx, in); out != in {
			t.Errorf("Ruby in vault: %q became %q", in, out)
		}
	}
	for in, n := range map[string]string{"`System.Crystal` fails.": "Crystal", "Use `Julia::Core` here.": "Julia"} {
		e := newEngine(t, nil)
		e.NewSession("seed").Mask(ctx, n+" Smith called about the invoice.")
		if out, _ := e.NewSession("later").Mask(ctx, in); out != in {
			t.Errorf("%s in vault: %q became %q", n, in, out)
		}
	}
}

func TestDottedAndHyphenNamesMask(t *testing.T) {
	cases := []struct{ seed, in, name string }{
		{"Julia Smith called about the invoice.", "Anne.Julia called.", "Julia"},
		{"Julia Smith called about the invoice.", "Anne.Julia wants a refund.", "Julia"},
		{"Ruby Smith called about the invoice.", "Mary.Ruby wants a refund.", "Ruby"},
		{"Ruby Smith called about the invoice.", "Ruby.Smith wants a refund.", "Ruby"},
		{"Ruby Smith called about the invoice.", "J.Ruby wants a refund.", "Ruby"},
		{"Ruby Smith called about the invoice.", "Dr.Ruby met us.", "Ruby"},
		{"Ruby Smith called about the invoice.", "Mrs.Ruby met us.", "Ruby"},
		{"Ruby Smith called about the invoice.", "Install Ruby-Gems first.", "Ruby"},
		{"Ruby Smith called about the invoice.", "Kotlin-Ruby bridge works.", "Ruby"},
		{"Ruby Smith called about the invoice.", "AB-Ruby met us.", "Ruby"},
		{"Ruby Smith called about the invoice.", "Ruby/Maria to cover the Friday shift.", "Ruby"},
	}
	for _, c := range cases {
		strictMaskModes(t, c.seed, c.in, func(mode, out, back string) {
			if hasWholeWord(out, c.name) {
				t.Errorf("%s: %q left raw as %q", mode, c.in, out)
			}
			if back != c.in {
				t.Errorf("%s: %q restored as %q", mode, c.in, back)
			}
		})
	}
	ctx := context.Background()
	for _, in := range []string{"J.Ruby Kelly called. J.Ruby wants a refund.", "Anne.Julia Kelly called. Anne.Julia wants a refund."} {
		s := newEngine(t, nil).NewSession("p")
		out, err := s.Mask(ctx, in)
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range []string{"Ruby", "Julia", "JULIA", "Kelly"} {
			if strings.Contains(in, w) && hasWholeWord(out, w) {
				t.Errorf("%q: %s left raw in %q", in, w, out)
			}
		}
		if back := s.Rehydrate(out); back != in {
			t.Errorf("%q: restored %q", in, back)
		}
	}
}
