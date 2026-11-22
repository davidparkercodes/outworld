package namedata

import (
	_ "embed"
	"strings"
	"sync"
	"unicode"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

//go:embed given.txt
var givenRaw string

//go:embed surnames.txt
var surnamesRaw string

//go:embed companies.txt
var companiesRaw string

//go:embed handlewords.txt
var handleWordsRaw string

type Given struct {
	Gender string
	Weak   bool
}

var (
	once     sync.Once
	given    map[string]Given
	surnames map[string]bool

	orgOnce     sync.Once
	companies   map[string][][]string
	handleWords map[string]bool
)

func load() {
	given = make(map[string]Given, 16000)
	for _, line := range strings.Split(givenRaw, "\n") {
		name, flags, ok := strings.Cut(line, " ")
		if !ok || len(flags) < 2 {
			continue
		}
		g := flags[:1]
		if g == "n" {
			g = ""
		}
		given[name] = Given{Gender: g, Weak: flags[1] == 'w'}
	}
	surnames = make(map[string]bool, 45000)
	for _, line := range strings.Split(surnamesRaw, "\n") {
		name, flag, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		surnames[name] = flag == "w"
	}
}

func LookupGiven(name string) (Given, bool) {
	once.Do(load)
	g, ok := given[textutil.Fold(name)]
	return g, ok
}

func LookupSurname(name string) (weak bool, ok bool) {
	once.Do(load)
	weak, ok = surnames[textutil.Fold(name)]
	return weak, ok
}

func CompanyKey(word string) string {
	word = textutil.Fold(word)
	if !strings.ContainsAny(word, "'’") {
		return word
	}
	return strings.NewReplacer("'", "", "’", "").Replace(word)
}

func companyTokens(name string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(name, func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '\'' || r == '’')
	}) {
		if k := CompanyKey(w); k != "" && k != "and" {
			out = append(out, k)
		}
	}
	return out
}

func loadOrgs() {
	companies = make(map[string][][]string, 512)
	for _, line := range strings.Split(companiesRaw, "\n") {
		toks := companyTokens(line)
		if len(toks) == 0 {
			continue
		}
		companies[toks[0]] = append(companies[toks[0]], toks[1:])
	}
	handleWords = make(map[string]bool, 2048)
	for _, line := range strings.Split(handleWordsRaw, "\n") {
		if line != "" {
			handleWords[line] = true
		}
	}
}

func CompaniesStarting(key string) [][]string {
	orgOnce.Do(loadOrgs)
	return companies[key]
}

func IsHandleWord(word string) bool {
	orgOnce.Do(loadOrgs)
	return handleWords[textutil.Fold(word)]
}
