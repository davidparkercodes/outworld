package detect

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
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
	if gap == "" || utf8.RuneCountInString(gap) > 3 {
		return false
	}
	for _, r := range gap {
		if !strings.ContainsRune(" &-.+,", r) && !textutil.IsHyphenVariant(r) {
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
	i := sort.Search(len(comps), func(k int) bool { return comps[k].end > s })
	for ; i < len(comps) && comps[i].start < e; i++ {
		c := comps[i]
		if !c.single || (c.start <= s && e <= c.end) {
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

var corpVerbsAfter = wordSet(`reported reports announced announces downgraded downgrades upgraded upgrades acquired acquires filed files shares stock issued issues posted posts launched launches disclosed discloses raised raises cut cuts expects forecast forecasts priced`)

var corpPrepsBefore = wordSet(`at of for via through the a an`)

var corpContext = wordSet(`earnings shares share stock stocks statement analyst analysts investors investor quarterly quarter revenue profit profits downgrade upgrade acquisition merger filing ipo account accounts card fraud branch advisor advisors adviser advisers portfolio brokerage bank loan mortgage refund warranty customer customers support ticket order subscription policy claim premium app website store stores product products device recall lawsuit sec dividend guidance layoffs fund funds securities rating ratings clients spokesperson spokesman spokeswoman`)

var fromAffiliationVerbs = wordSet(`hired recruited rang came moved transferred poached calling emailing writing`)

func fromAffiliation(text string, s int) bool {
	w := wordsBefore(text, s, 2)
	if len(w) < 2 || w[0] != "from" {
		return false
	}
	if personVerbs[w[1]] || fromAffiliationVerbs[w[1]] {
		return true
	}
	j := s - len(" from ")
	if j <= 0 || j-len(w[1]) < 0 {
		return false
	}
	prev := text[j-len(w[1]) : j]
	return strings.EqualFold(prev, w[1]) && shapeOf(prev) == shapeTitle && KnownName(prev)
}

func companyEvidence(text string, s, e int) bool {
	if w := wordsBefore(text, s, 1); len(w) > 0 && corpPrepsBefore[w[0]] {
		return true
	}
	if fromAffiliation(text, s) {
		return true
	}
	rest := text[e:]
	for _, suf := range []string{"'s", "’s"} {
		if strings.HasPrefix(rest, suf) {
			return true
		}
	}
	after := wordsAfter(text, e, 3)
	if len(after) > 0 && corpVerbsAfter[after[0]] {
		return true
	}
	if len(after) > 1 && (after[1] == "it" || after[1] == "its") {
		return true
	}
	if len(after) > 2 && after[1] == "that" && (after[2] == "it" || after[2] == "its") {
		return true
	}
	a, b := sentenceBounds(text, s, e)
	for _, w := range strings.FieldsFunc(strings.ToLower(text[a:b]), func(r rune) bool { return !unicode.IsLetter(r) }) {
		if corpContext[w] {
			return true
		}
	}
	return false
}

func personEvidence(text string, s, e int) bool {
	if personVerbAfter(text, e) || personDirectedBefore(text, s) {
		return true
	}
	w := wordsBefore(text, s, 1)
	return len(w) > 0 && roleBefore[w[0]]
}

func personShaped(name string) bool {
	words := strings.Fields(name)
	if len(words) == 0 || len(words) > 3 {
		return false
	}
	for _, w := range words {
		if shapeOf(w) != shapeTitle {
			return false
		}
	}
	if len(words) == 1 {
		return KnownName(words[0]) && !IsTechTerm(words[0])
	}
	g, ok := givenOf(words[0])
	return ok && !g.Weak
}

func companyPeople(text string, comps []companyMatch) map[int]bool {
	var out map[int]bool
	corporate := map[string]bool{}
	for _, c := range comps {
		if companyEvidence(text, c.start, c.end) {
			corporate[strings.ToLower(text[c.start:c.end])] = true
		}
	}
	for i, c := range comps {
		name := text[c.start:c.end]
		if corporate[strings.ToLower(name)] || !personShaped(name) || !personEvidence(text, c.start, c.end) || corpSuffixEnd(text, c.end) > c.end {
			continue
		}
		if out == nil {
			out = map[int]bool{}
		}
		out[i] = true
	}
	return out
}

func guardOrgs(text string, spans []entity.Span, comps []companyMatch) []entity.Span {
	out := spans[:0]
	var orgs []entity.Span
	org := func(s, e int) {
		orgs = append(orgs, entity.Span{Start: s, End: e, Type: entity.Org, Label: "organization", Text: text[s:e], Source: "names:org", Priority: 25, Score: 1})
	}
	people := companyPeople(text, comps)
	var guarded []companyMatch
	for i, c := range comps {
		if !people[i] {
			guarded = append(guarded, c)
		}
	}
	for i := range comps {
		if people[i] {
			c := comps[i]
			orgs = append(orgs, entity.Span{Start: c.start, End: c.end, Type: entity.Person, Label: "person", Text: text[c.start:c.end], Source: "names:company-person", Priority: 28, Score: 1})
		}
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
		if orgGuarded[sp.Source] && inCompany(guarded, sp.Start, sp.End) {
			continue
		}
		out = append(out, sp)
	}
	for _, c := range guarded {
		org(c.start, corpSuffixEnd(text, c.end))
	}
	return append(out, orgs...)
}
