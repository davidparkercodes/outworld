package textutil

import (
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	mathLetterFirst = 0x1D400
	mathLetterLast  = 0x1D6A3
	mathLetterStyle = 52
	scriptRunWindow = 64
	scriptRunDigits = 7
)

var letterlike = map[rune]rune{
	0x2102: 'C', 0x210A: 'g', 0x210B: 'H', 0x210C: 'H', 0x210D: 'H', 0x210E: 'h',
	0x2110: 'I', 0x2111: 'I', 0x2112: 'L', 0x2113: 'l', 0x2115: 'N', 0x2119: 'P',
	0x211A: 'Q', 0x211B: 'R', 0x211C: 'R', 0x211D: 'R', 0x2124: 'Z', 0x2128: 'Z',
	0x212C: 'B', 0x212D: 'C', 0x212F: 'e', 0x2130: 'E', 0x2131: 'F', 0x2133: 'M',
	0x2134: 'o',
}

func IsZeroWidth(r rune) bool {
	switch {
	case r >= 0x200B && r <= 0x200F, r >= 0x2060 && r <= 0x2064, r >= 0x202A && r <= 0x202E, r >= 0x2066 && r <= 0x2069:
		return true
	}
	return r == 0xFEFF || r == 0x00AD || r == 0x061C || r == 0x180E
}

func IsDashVariant(r rune) bool {
	return r != '-' && (r == 0x2212 || unicode.Is(unicode.Pd, r)) && !cjkDash(r)
}

func cjkDash(r rune) bool { return r == 0x301C || r == 0x3030 || r == 0x30A0 }

const (
	phoneSeparatorRunes = " \t\n\r.-~\u301c\u3030\u30a0/*\u00b7\u30fb\u2022_|:\u2212\uff5e\u30fc\uff70\u2015\u2043\u2027\u2219\u22c5\u02d7\u2796\uff65\u0387\u2e31\u2e33\u16eb\u223c\u02dc\ufe4d\ufe4e\ufe4f\u2024\u01c0\u1427\u2cba"
	phoneSeparatorASCII = " .-~/*_|:"
	foldScanLast        = 0xFFFF
)

var (
	phoneSepOnce sync.Once
	phoneSepSet  map[rune]bool
	foldedTo     map[rune][]rune
)

func buildPhoneSeparators() {
	phoneSepSet = map[rune]bool{}
	foldedTo = map[rune][]rune{}
	for _, r := range phoneSeparatorRunes {
		phoneSepSet[r] = true
	}
	for _, tab := range phoneSeparatorTables {
		forEachRune(tab, func(r rune) { phoneSepSet[r] = true })
	}
	for r := rune(0); r < 0x3100; r++ {
		if unicode.IsSpace(r) || IsZeroWidth(r) {
			phoneSepSet[r] = true
		}
	}
	phoneSepSet[0xFEFF] = true
	for r := rune(utf8.RuneSelf); r <= foldScanLast; r++ {
		f, ok := formRune(r)
		if !ok || f < 0 || f >= utf8.RuneSelf {
			continue
		}
		foldedTo[f] = append(foldedTo[f], r)
		if strings.ContainsRune(phoneSeparatorASCII, f) {
			phoneSepSet[r] = true
		}
	}
}

var phoneSeparatorTables = []*unicode.RangeTable{unicode.Zs, unicode.Zl, unicode.Zp, unicode.Pd, unicode.White_Space}

func forEachRune(tab *unicode.RangeTable, f func(rune)) {
	for _, rg := range tab.R16 {
		for r := rune(rg.Lo); r <= rune(rg.Hi); r += rune(rg.Stride) {
			f(r)
		}
	}
	for _, rg := range tab.R32 {
		for r := rune(rg.Lo); r <= rune(rg.Hi); r += rune(rg.Stride) {
			f(r)
		}
	}
}

func IsPhoneSeparator(r rune) bool {
	phoneSepOnce.Do(buildPhoneSeparators)
	return phoneSepSet[r] || unicode.Is(unicode.Pd, r)
}

func PhoneSeparators() []rune {
	phoneSepOnce.Do(buildPhoneSeparators)
	out := make([]rune, 0, len(phoneSepSet))
	for r := range phoneSepSet {
		out = append(out, r)
	}
	sort.Slice(out, func(a, b int) bool { return out[a] < out[b] })
	return out
}

func classRune(b *strings.Builder, r rune) {
	if strings.ContainsRune(`\-]^[`, r) {
		b.WriteByte('\\')
	}
	b.WriteRune(r)
}

const lineBreakRunes = "\n\r\v\f\u0085\u2028\u2029"

func IsPhoneIgnorable(r rune) bool {
	switch {
	case unicode.Is(unicode.Cf, r), r >= 0xFE00 && r <= 0xFE0F, r >= 0xE0000 && r <= 0xE007F, r >= 0xE0100 && r <= 0xE01EF, r >= 0x180B && r <= 0x180F:
		return true
	}
	switch r {
	case 0x034F, 0x115F, 0x1160, 0x3164, 0xFFA0, 0x17B4, 0x17B5:
		return true
	}
	return false
}

func IsPhoneGapRune(r rune) bool {
	if IsPhoneIgnorable(r) || GluingDigit(r) {
		return false
	}
	return IsPhoneSeparator(r) || unicode.In(r, unicode.Lm, unicode.Sk) || !unicode.IsLetter(r) && !unicode.IsDigit(r)
}

func IsLineBreak(r rune) bool { return strings.ContainsRune(lineBreakRunes, r) }

func phoneClass(lineBreaks bool) string {
	var b strings.Builder
	b.WriteString(`[\p{Pd}`)
	for _, r := range PhoneSeparators() {
		if lineBreaks || !IsLineBreak(r) {
			classRune(&b, r)
		}
	}
	b.WriteByte(']')
	return b.String()
}

func PhoneSeparatorClass() string { return phoneClass(true) }

func PhoneGapPattern() string {
	var lb strings.Builder
	for _, r := range lineBreakRunes {
		classRune(&lb, r)
	}
	return `(?:` + phoneClass(false) + `?[ \t]*(?:\r\n|[` + lb.String() + `])[ \t]*|` + phoneClass(true) + `)`
}

func FoldClass(c rune) string {
	phoneSepOnce.Do(buildPhoneSeparators)
	var b strings.Builder
	b.WriteByte('[')
	classRune(&b, c)
	for _, r := range foldedTo[c] {
		classRune(&b, r)
	}
	b.WriteByte(']')
	return b.String()
}

const PlainPhoneSeparators = " .-"

func RuneClass(runes string, whitespace bool) string {
	var b strings.Builder
	b.WriteByte('[')
	if whitespace {
		b.WriteString(`\s`)
	}
	for _, r := range runes {
		if whitespace && r == ' ' {
			continue
		}
		classRune(&b, r)
	}
	b.WriteByte(']')
	return b.String()
}

func IsCJKLetter(r rune) bool {
	return r >= 0x2E80 && unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}

func IsHyphenVariant(r rune) bool {
	return r == 0x2010 || r == 0x2011 || r == 0xFE63 || r == 0xFF0D || r == 0x2212
}

func formRune(r rune) (rune, bool) {
	switch {
	case r >= 0xFF01 && r <= 0xFF5E:
		return r - 0xFEE0, true
	case r == 0x00A0 || r == 0x1680 || r >= 0x2000 && r <= 0x200A || r == 0x202F || r == 0x205F || r == 0x3000:
		return ' ', true
	case IsZeroWidth(r):
		return -1, true
	case IsDashVariant(r):
		return '-', true
	case letterlike[r] != 0:
		return letterlike[r], true
	case r >= mathLetterFirst && r <= mathLetterLast:
		k := (r - mathLetterFirst) % mathLetterStyle
		if k < 26 {
			return 'A' + k, true
		}
		return 'a' + k - 26, true
	}
	if d, ok := scriptDigit(r); ok {
		return d, true
	}
	if d, ok := decimalDigit(r); ok {
		return d, true
	}
	return r, false
}

func decimalDigit(r rune) (rune, bool) {
	if r < utf8.RuneSelf || !unicode.IsDigit(r) {
		return 0, false
	}
	for _, rg := range unicode.Nd.R16 {
		if r >= rune(rg.Lo) && r <= rune(rg.Hi) && (r-rune(rg.Lo))%rune(rg.Stride) == 0 {
			return '0' + ((r-rune(rg.Lo))/rune(rg.Stride))%10, true
		}
	}
	for _, rg := range unicode.Nd.R32 {
		if r >= rune(rg.Lo) && r <= rune(rg.Hi) && (r-rune(rg.Lo))%rune(rg.Stride) == 0 {
			return '0' + ((r-rune(rg.Lo))/rune(rg.Stride))%10, true
		}
	}
	return 0, false
}

func digitLike(r rune) bool {
	if r >= '0' && r <= '9' {
		return true
	}
	if _, ok := scriptDigit(r); ok {
		return true
	}
	_, ok := decimalDigit(r)
	return ok
}

func runSeparator(r rune) bool {
	return r == '-' || r == '.' || r == ' ' || r == '/' || r == '(' || r == ')' || IsDashVariant(r) || r == 0x00A0
}

func scriptRunIsNumber(s string, i, size int) bool {
	digits := 1
	for j := i; j > 0 && i-j < scriptRunWindow; {
		r, n := utf8.DecodeLastRuneInString(s[:j])
		if digitLike(r) {
			digits++
		} else if !runSeparator(r) {
			break
		}
		j -= n
	}
	for j := i + size; j < len(s) && j-i < scriptRunWindow; {
		r, n := utf8.DecodeRuneInString(s[j:])
		if digitLike(r) {
			digits++
		} else if !runSeparator(r) {
			break
		}
		j += n
	}
	return digits >= scriptRunDigits
}

func scriptDigit(r rune) (rune, bool) {
	switch {
	case r == 0x00B9:
		return '1', true
	case r == 0x00B2 || r == 0x00B3:
		return '0' + r - 0x00B0, true
	case r == 0x2070 || r >= 0x2074 && r <= 0x2079:
		return '0' + r - 0x2070, true
	case r >= 0x2080 && r <= 0x2089:
		return '0' + r - 0x2080, true
	}
	return 0, false
}

func asciiAlnumAt(s string, i int) bool {
	if i < 0 || i >= len(s) {
		return false
	}
	c := s[i]
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func foldAt(s string, i int, r rune, size int) (rune, bool) {
	f, ok := formRune(r)
	if !ok {
		return r, false
	}
	if _, script := scriptDigit(r); script && (asciiAlnumAt(s, i-1) || asciiAlnumAt(s, i+size)) && !scriptRunIsNumber(s, i, size) {
		return r, false
	}
	return f, true
}

func NeedsFormFold(s string) bool {
	for i := 0; i < len(s); {
		if s[i] < utf8.RuneSelf {
			i++
			continue
		}
		r, size := utf8.DecodeRuneInString(s[i:])
		if _, ok := foldAt(s, i, r, size); ok {
			return true
		}
		i += size
	}
	return false
}

func FoldFormsMap(s string) (string, []int, []int) {
	var b strings.Builder
	b.Grow(len(s))
	starts := make([]int, 0, len(s))
	ends := make([]int, 0, len(s))
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		f, changed := foldAt(s, i, r, size)
		switch {
		case !changed:
			for k := 0; k < size; k++ {
				b.WriteByte(s[i+k])
				starts = append(starts, i)
				ends = append(ends, i+size)
			}
		case f >= 0:
			n, _ := b.WriteRune(f)
			for k := 0; k < n; k++ {
				starts = append(starts, i)
				ends = append(ends, i+size)
			}
		}
		i += size
	}
	return b.String(), starts, ends
}

func FoldForms(s string) string {
	if !NeedsFormFold(s) {
		return s
	}
	f, _, _ := FoldFormsMap(s)
	return f
}

func FoldRuneAt(s string, i int) (rune, int, bool) {
	r, size := utf8.DecodeRuneInString(s[i:])
	if r < utf8.RuneSelf {
		return r, size, false
	}
	f, ok := foldAt(s, i, r, size)
	return f, size, ok
}

func DigitLike(r rune) bool { return digitLike(r) }

func DigitValue(r rune) (int, bool) {
	if r >= '0' && r <= '9' {
		return int(r - '0'), true
	}
	if d, ok := scriptDigit(r); ok {
		return int(d - '0'), true
	}
	if d, ok := decimalDigit(r); ok {
		return int(d - '0'), true
	}
	return 0, false
}

var superscriptDigits = [10]rune{0x2070, 0x00B9, 0x00B2, 0x00B3, 0x2074, 0x2075, 0x2076, 0x2077, 0x2078, 0x2079}

func StyleDigit(like rune, d int) rune {
	v, ok := DigitValue(like)
	switch {
	case !ok || like < utf8.RuneSelf:
		return '0' + rune(d)
	case like >= 0x2080 && like <= 0x2089:
		return 0x2080 + rune(d)
	case like == superscriptDigits[v]:
		return superscriptDigits[d]
	}
	return like - rune(v) + rune(d)
}

func GluingDigit(r rune) bool { _, script := scriptDigit(r); return digitLike(r) && !script }
