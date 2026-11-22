package detect

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

type orgTok struct {
	start, end int
}

type companyMatch struct {
	start, end int
	single     bool
}

func orgTokens(text string) []orgTok {
	var out []orgTok
	start := -1
	for i, r := range text {
		word := unicode.IsLetter(r) || unicode.IsDigit(r)
		if !word && start >= 0 && (r == '\'' || r == '’') {
			nr, _ := utf8.DecodeRuneInString(text[i+utf8.RuneLen(r):])
			word = unicode.IsLetter(nr)
		}
		if word {
			if start < 0 {
				start = i
			}
			continue
		}
		if start >= 0 {
			out = append(out, orgTok{start, i})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, orgTok{start, len(text)})
	}
	return out
}

func companyGap(gap string) bool {
	if len(gap) == 0 || len(gap) > 3 {
		return false
	}
	for i := 0; i < len(gap); i++ {
		if !strings.ContainsRune(" &-.+,", rune(gap[i])) {
			return false
		}
	}
	return true
}

func companyTokEq(key, want string) bool {
	return key == want || (strings.HasSuffix(key, "s") && key[:len(key)-1] == want)
}

func companyMatches(text string) []companyMatch {
	toks := orgTokens(text)
	var out []companyMatch
	for i := 0; i < len(toks); i++ {
		t := toks[i]
		r, _ := utf8.DecodeRuneInString(text[t.start:])
		if !unicode.IsUpper(r) {
			continue
		}
		key := namedata.CompanyKey(text[t.start:t.end])
		cands := namedata.CompaniesStarting(key)
		if cands == nil && strings.HasSuffix(key, "s") {
			cands = namedata.CompaniesStarting(key[:len(key)-1])
		}
		best, bestN := -1, 0
		for _, rest := range cands {
			j, ok := i, true
			for ri, want := range rest {
				j++
				if j < len(toks) && namedata.CompanyKey(text[toks[j].start:toks[j].end]) == "and" && companyGap(text[toks[j-1].end:toks[j].start]) {
					j++
				}
				if j >= len(toks) || !companyGap(text[toks[j-1].end:toks[j].start]) {
					ok = false
					break
				}
				k := namedata.CompanyKey(text[toks[j].start:toks[j].end])
				if k != want && !(ri == len(rest)-1 && companyTokEq(k, want)) {
					ok = false
					break
				}
			}
			if ok && toks[j].end > best {
				best, bestN = toks[j].end, j-i
			}
		}
		if best > 0 {
			out = append(out, companyMatch{start: t.start, end: best, single: bestN == 0})
			i += bestN
		}
	}
	return out
}

func inCompany(comps []companyMatch, s, e int) bool {
	for _, c := range comps {
		if c.single {
			if c.start <= s && e <= c.end {
				return true
			}
			continue
		}
		if s < c.end && c.start < e {
			return true
		}
	}
	return false
}

const corpWords = `Group|GROUP|Holdings|HOLDINGS|Partners|PARTNERS|Associates|ASSOCIATES|Bank|BANK|Capital|CAPITAL|Foundation|FOUNDATION|University|UNIVERSITY|College|COLLEGE|Institute|INSTITUTE|Hospital|HOSPITAL|Law|LAW|Ventures|VENTURES|Industries|INDUSTRIES|Enterprises|ENTERPRISES|Securities|SECURITIES|Investments|INVESTMENTS`

var corpAfterRE = regexp.MustCompile(`^(?:,? (?:Inc|INC|LLC|L\.L\.C|Ltd|LTD|LLP|LP|PLC|plc|GmbH|Corp|CORP|Corporation|CORPORATION|Co|CO|Company|COMPANY)\b\.?| (?:&|and|AND) (?:Co|CO|Company|COMPANY|Sons|SONS|Daughters|Associates|ASSOCIATES|Partners|PARTNERS|Bros|Brothers|BROTHERS)\b\.?| (?:` + corpWords + `)\b)`)

var corpLastRE = regexp.MustCompile(` (?:` + corpWords + `|Inc|INC|LLC|Ltd|LTD|LLP|PLC|GmbH|Corp|CORP|Corporation|CORPORATION|Company|COMPANY)$`)

func corpSuffixEnd(text string, e int) int {
	if m := corpAfterRE.FindStringIndex(text[e:]); m != nil {
		return e + m[1]
	}
	return e
}

var orgGuarded = map[string]bool{"names:full": true, "names:first": true, "names:last": true, "names:lower": true, "names:initial": true, "names:family-first": true}

func guardOrgs(text string, spans []entity.Span, comps []companyMatch) []entity.Span {
	out := spans[:0]
	var orgs []entity.Span
	org := func(s, e int) {
		orgs = append(orgs, entity.Span{Start: s, End: e, Type: entity.Org, Label: "organization", Text: text[s:e], Source: "names:org", Priority: 25, Score: 1})
	}
	for _, sp := range spans {
		if orgGuarded[sp.Source] || sp.Source == "names:label" {
			if end := corpSuffixEnd(text, sp.End); end > sp.End {
				org(sp.Start, end)
				continue
			}
			if corpLastRE.MatchString(sp.Text) {
				org(sp.Start, sp.End)
				continue
			}
		}
		if orgGuarded[sp.Source] && inCompany(comps, sp.Start, sp.End) {
			continue
		}
		out = append(out, sp)
	}
	for _, c := range comps {
		org(c.start, corpSuffixEnd(text, c.end))
	}
	return append(out, orgs...)
}
