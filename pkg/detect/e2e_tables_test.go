package detect_test

import (
	"bytes"
	"context"
	"sort"
	"strings"
	"testing"
	"unicode"

	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/outworld"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

var endToEndDiffers = map[string]string{
	"Jean-Claude Van Damme said": "a name particle with no vault record is kept as written, so Van stays in place by design",
	"Dear Tran Thi Mai,":         "the family-first surrogate keeps the middle marker Thi in place by design",
	"Marx\u2013Engels letters":   "the fold pass rewrites the en dash to an ASCII hyphen and the ASCII form masks the pair",
	"the Taft\u2013Hartley act":  "the fold pass rewrites the en dash to an ASCII hyphen and the ASCII form masks the pair",
}

func freshSession(t *testing.T) *outworld.Session {
	t.Helper()
	v, err := vault.Open(vault.NewMemoryBackend(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	e, err := outworld.New(outworld.Options{Vault: v, Policy: outworld.DefaultPolicy()})
	if err != nil {
		t.Fatal(err)
	}
	return e.NewSession("tables")
}

func nameWords(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return !unicode.IsLetter(r) && r != '\'' })
}

func replacedWords(in, out string) []string {
	kept := map[string]bool{}
	for _, w := range nameWords(out) {
		kept[w] = true
	}
	seen := map[string]bool{}
	var gone []string
	for _, w := range nameWords(in) {
		if !kept[w] && !seen[w] {
			seen[w] = true
			gone = append(gone, w)
		}
	}
	sort.Strings(gone)
	return gone
}

func expectedWords(spans []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, sp := range spans {
		for _, w := range nameWords(sp) {
			if !seen[w] {
				seen[w] = true
				out = append(out, w)
			}
		}
	}
	sort.Strings(out)
	return out
}

func TestDetectTablesHoldEndToEnd(t *testing.T) {
	for _, row := range detect.EndToEndRows() {
		s := freshSession(t)
		out, err := s.Mask(context.Background(), row.Text)
		if err != nil {
			t.Fatalf("%s %q: %v", row.Table, row.Text, err)
		}
		got, want := strings.Join(replacedWords(row.Text, out), " "), strings.Join(expectedWords(row.Want), " ")
		if why, ok := endToEndDiffers[row.Text]; ok {
			if got == want {
				t.Errorf("%s %q: listed as differing (%s) but now matches; remove it from the list", row.Table, row.Text, why)
			}
			continue
		}
		if got != want {
			t.Errorf("%s %q: end to end replaced [%s], detect level gives [%s] (output %q)", row.Table, row.Text, got, want, out)
		}
		if back := s.Rehydrate(out); back != row.Text {
			t.Errorf("%s %q: restored %q", row.Table, row.Text, back)
		}
	}
}
