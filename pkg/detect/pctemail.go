package detect

import (
	"strings"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

const (
	pctValueMax    = 256
	pctDecodeTimes = 2
)

func isHexByte(c byte) bool {
	return c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F'
}

func hexVal(c byte) byte {
	switch {
	case c >= 'a':
		return c - 'a' + 10
	case c >= 'A':
		return c - 'A' + 10
	}
	return c - '0'
}

func pctDecodeOnce(s string) (string, []int, []int, bool) {
	var b strings.Builder
	starts := make([]int, 0, len(s))
	ends := make([]int, 0, len(s))
	changed := false
	for i := 0; i < len(s); {
		if s[i] == '%' && i+2 < len(s) && isHexByte(s[i+1]) && isHexByte(s[i+2]) {
			b.WriteByte(hexVal(s[i+1])<<4 | hexVal(s[i+2]))
			starts, ends = append(starts, i), append(ends, i+3)
			i += 3
			changed = true
			continue
		}
		b.WriteByte(s[i])
		starts, ends = append(starts, i), append(ends, i+1)
		i++
	}
	return b.String(), starts, ends, changed
}

func pctDecode(s string) (string, []int, []int) {
	out, starts, ends, _ := pctDecodeOnce(s)
	for n := 1; n < pctDecodeTimes && strings.Contains(out, "%"); n++ {
		next, s2, e2, changed := pctDecodeOnce(out)
		if !changed {
			break
		}
		for k := range s2 {
			s2[k], e2[k] = starts[s2[k]], ends[e2[k]-1]
		}
		out, starts, ends = next, s2, e2
	}
	return out, starts, ends
}

func pctValueStop(c byte) bool {
	return c == '?' || c == '&' || c == '=' || c == '/' || c == '#' || c == ';' || c == '"' || c == '\'' || c == '<' || c == '>' || isSpaceByte(c)
}

func scanEncodedEmails(text, _ string) []entity.Span {
	var out []entity.Span
	done := -1
	for i := 0; i < len(text); i++ {
		if text[i] != '%' || i < done || !(strings.HasPrefix(text[i:], "%40") || strings.HasPrefix(text[i:], "%2540")) {
			continue
		}
		a := i
		for a > 0 && i-a < pctValueMax && !pctValueStop(text[a-1]) {
			a--
		}
		b := i
		for b < len(text) && b-i < pctValueMax && !pctValueStop(text[b]) {
			b++
		}
		done = b
		dec, starts, ends := pctDecode(text[a:b])
		for _, m := range emailRE.FindAllStringIndex(dec, -1) {
			s, e := a+starts[m[0]], a+ends[m[1]-1]
			if !strings.Contains(text[s:e], "%") {
				continue
			}
			out = append(out, entity.Span{Start: s, End: e})
		}
	}
	return out
}

func EncodedEmail(s string) (string, func(string) string, bool) {
	if strings.Contains(s, "@") || !strings.Contains(s, "%") {
		return "", nil, false
	}
	dec, starts, ends := pctDecode(s)
	if !strings.Contains(dec, "@") {
		return "", nil, false
	}
	enc := map[byte]string{}
	for k := range dec {
		if orig := s[starts[k]:ends[k]]; orig != dec[k:k+1] {
			enc[dec[k]] = orig
		}
	}
	encode := func(v string) string {
		var b strings.Builder
		for i := 0; i < len(v); i++ {
			if e, ok := enc[v[i]]; ok {
				b.WriteString(e)
			} else {
				b.WriteByte(v[i])
			}
		}
		return b.String()
	}
	return dec, encode, true
}
