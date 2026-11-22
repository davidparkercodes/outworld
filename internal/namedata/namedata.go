package namedata

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

//go:embed given.txt
var givenRaw string

//go:embed surnames.txt
var surnamesRaw string

type Given struct {
	Gender string
	Weak   bool
}

var (
	once     sync.Once
	given    map[string]Given
	surnames map[string]bool
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
