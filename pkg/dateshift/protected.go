package dateshift

import (
	"regexp"
	"strings"
	"unicode/utf8"
)

const (
	versionWindow      = 64
	versionValueSpan   = 48
	versionPersonWords = 3
)

var versionCueRE = regexp.MustCompile(`(?i)ver(?:sion)?["']?[ \t]*(?:[:=]|"[ \t]*:)`)

var versionLabelRE = regexp.MustCompile(`(?:^|[^A-Za-z0-9_\-])([A-Za-z][A-Za-z0-9_\-]*)(?:"[ \t]*:[ \t]*"|["']?[ \t]*[:=][ \t]*["']?)$`)

var personalLabelRE = regexp.MustCompile(`(?i)dob|birth|born|patient|person`)

func markVersioned(text string, ms []Match) []Match {
	if !versionCueRE.MatchString(text) {
		return ms
	}
	for i := range ms {
		ms[i].Versioned = versionValue(text, ms[i].Start)
	}
	return ms
}

func versionLabel(label string) bool {
	lower := strings.ToLower(label)
	switch {
	case lower == "version" || lower == "ver":
	case strings.HasSuffix(lower, "-version") || strings.HasSuffix(lower, "_version"):
	case len(label) > len("Version") && strings.HasSuffix(label, "Version"):
	default:
		return false
	}
	return !personalLabelRE.MatchString(label)
}

func versionValue(text string, start int) bool {
	lo := max(0, start-versionWindow)
	m := versionLabelRE.FindStringSubmatchIndex(text[lo:start])
	if m == nil {
		return false
	}
	label := text[lo+m[2] : lo+m[3]]
	if !versionLabel(label) {
		return false
	}
	line := text[:lo+m[2]]
	if i := strings.LastIndexByte(line, '\n'); i >= 0 {
		line = line[i+1:]
	}
	words := strings.Fields(line)
	if len(words) > versionPersonWords {
		words = words[len(words)-versionPersonWords:]
	}
	for _, w := range words {
		if personalLabelRE.MatchString(w) {
			return false
		}
	}
	return true
}

func versionWindows(text string) [][2]int {
	var out [][2]int
	for _, h := range versionCueRE.FindAllStringIndex(text, -1) {
		a, b := max(0, h[0]-versionWindow), min(len(text), h[1]+versionValueSpan)
		for a > 0 && !utf8.RuneStart(text[a]) {
			a--
		}
		for b < len(text) && !utf8.RuneStart(text[b]) {
			b++
		}
		if n := len(out); n > 0 && a <= out[n-1][1] {
			out[n-1][1] = max(out[n-1][1], b)
			continue
		}
		out = append(out, [2]int{a, b})
	}
	return out
}
