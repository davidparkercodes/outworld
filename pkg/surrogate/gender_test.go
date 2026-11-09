package surrogate

import (
	"strings"
	"testing"
)

func TestGenderedListsAreDisjoint(t *testing.T) {
	seen := map[string]string{}
	for g, list := range map[string][]string{Female: femaleGiven, Male: maleGiven, "n": neutralGiven} {
		if len(list) < 40 {
			t.Errorf("list %q too small: %d", g, len(list))
		}
		for _, n := range list {
			if prev, ok := seen[n]; ok {
				t.Errorf("%s in both %q and %q", n, prev, g)
			}
			seen[n] = g
		}
	}
}

func TestGivenNameFollowsGender(t *testing.T) {
	g := NewGenerator([]byte("0123456789abcdef0123456789abcdef"))
	for _, c := range []struct{ real, want string }{{"Priya", Female}, {"Marcus", Male}, {"Zorblax", Neutral}} {
		for i := 0; i < 5; i++ {
			if got := GenderOf(g.GivenName(c.real, i)); got != c.want {
				t.Errorf("%s attempt %d: gender %q want %q", c.real, i, got, c.want)
			}
		}
	}
}

func TestNickname(t *testing.T) {
	for _, c := range []struct{ alias, real, sur, prefix string }{
		{"Marc", "Marcus", "Ezra", "Ez"},
		{"Toby", "Tobiah", "Leander", "Le"},
		{"Rike", "Henrike", "Theodora", "Dora"},
		{"Bubba", "Robert", "Silas", "Si"},
	} {
		seen := map[string]bool{}
		for i := 0; i < 60; i++ {
			n := Nickname(c.alias, c.real, c.sur, i)
			if n == "" || strings.EqualFold(n, c.alias) {
				t.Fatalf("%v attempt %d: bad nickname %q", c, i, n)
			}
			seen[n] = true
		}
		first := Nickname(c.alias, c.real, c.sur, 0)
		if !strings.HasPrefix(first, c.prefix) || first == c.sur {
			t.Errorf("%v: first nickname %q", c, first)
		}
		if len(seen) < 20 {
			t.Errorf("%v: only %d distinct nicknames", c, len(seen))
		}
	}
}
