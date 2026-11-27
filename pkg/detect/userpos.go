package detect

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
)

var userCueRE = regexp.MustCompile(`(?i:\b(?:user|username|login|account|handle|userid))["'\x60“”‘’]?(?:[ \t]*[:=][ \t]*|[ \t]+)["'\x60“”‘’]?@?([A-Za-z][A-Za-z0-9._\-]{1,30})`)

var userMentionRE = regexp.MustCompile(`(?:^|[\s(\[{,;:>"'\x60*])@([A-Za-z][A-Za-z0-9._\-]{1,30})`)

func UsernamePositions(text string) [][2]int {
	var out [][2]int
	add := func(s, e int) {
		e = trimHandle(text, s, e)
		if e-s < 3 {
			return
		}
		if e < len(text) {
			if r, _ := utf8.DecodeRuneInString(text[e:]); textutil.IsWordRune(r) {
				return
			}
		}
		body := textutil.Fold(text[s:e])
		if handleStop[body] || handleWordish(body) || textutil.IsCommonWord(body) || namedata.IsHandleWord(body) {
			return
		}
		out = append(out, [2]int{s, e})
	}
	low := strings.ToLower(text)
	if strings.Contains(low, "user") || strings.Contains(low, "login") || strings.Contains(low, "account") || strings.Contains(low, "handle") {
		for _, m := range userCueRE.FindAllStringSubmatchIndex(text, -1) {
			add(m[2], m[3])
		}
	}
	if strings.Contains(text, "sers") || strings.Contains(text, "home") {
		for _, m := range homeHandleRE.FindAllStringSubmatchIndex(text, -1) {
			add(m[2], m[3])
		}
	}
	if strings.Contains(text, "@") {
		for _, m := range userMentionRE.FindAllStringSubmatchIndex(text, -1) {
			add(m[2], m[3])
		}
	}
	return out
}
