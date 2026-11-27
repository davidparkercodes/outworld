package detect

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/internal/scaletest"
)

func longWordDetectors() []Detector {
	return []Detector{NewRegexDetector(), DateDetector{}, NameDetector{}}
}

func longBase64Word(n int) string {
	const alphabet = "aZ3kQ9mXb7LpR2sTv8WnYc4HdJ6eFgK1uVi5oP0qBwExCyDzGrAhMjNlOfSt"
	var b strings.Builder
	for i := 0; b.Len() < n; i++ {
		b.WriteByte(alphabet[(i*31+i/7)%len(alphabet)])
	}
	return b.String()
}

func longGluedTitles(n int) string {
	return strings.Repeat("MariaGarciaJohnSmithAdaLovelace", n/31+1)[:n]
}

func TestUnbrokenWordsScaleLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	size := scaletest.Size(64 << 10)
	for _, c := range []struct {
		name string
		gen  func(int) string
	}{
		{"base64_word", longBase64Word},
		{"glued_titles", longGluedTitles},
		{"base64_in_sentence", func(n int) string { return "Hi Maria, the key is " + longBase64Word(n) + " thanks." }},
	} {
		dets := longWordDetectors()
		checkGrowth(t, c.name, c.gen, size, func(text string) {
			for _, r := range RunAll(context.Background(), text, dets) {
				if r.Err != nil {
					t.Fatal(r.Err)
				}
			}
		})
	}
}
