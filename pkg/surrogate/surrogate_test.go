package surrogate

import (
	"regexp"
	"strings"
	"testing"
)

func TestDeterministicPerSecret(t *testing.T) {
	a, b, c := NewGenerator([]byte("one")), NewGenerator([]byte("one")), NewGenerator([]byte("two"))
	if a.GivenName("John", 0) != b.GivenName("John", 0) {
		t.Fatal("same secret must give same surrogate")
	}
	diff := 0
	for _, n := range []string{"John", "Mary", "Wei", "Priya", "Omar", "Lena"} {
		if a.GivenName(n, 0) != c.GivenName(n, 0) {
			diff++
		}
	}
	if diff == 0 {
		t.Fatal("different secrets should usually differ")
	}
}

func TestShapes(t *testing.T) {
	g := NewGenerator([]byte("shape"))
	phone := regexp.MustCompile(`^\([2-8]\d\d\) 555-01\d\d$`)
	for i := 0; i < 50; i++ {
		if p := g.Phone("(804) 222-1234", i); !phone.MatchString(p) {
			t.Fatalf("phone %q", p)
		}
	}
	if p := g.Phone("+1 804.222.1234", 0); !regexp.MustCompile(`^\+1 [2-8]\d\d\.555\.01\d\d$`).MatchString(p) {
		t.Fatalf("phone11 %q", p)
	}
	if p := g.Phone("+44 20 7946 0958", 0); !strings.HasPrefix(p, "+44 ") || len(p) != len("+44 20 7946 0958") {
		t.Fatalf("intl %q", p)
	}
	e := g.Email("x@corp.test", 0)
	if !regexp.MustCompile(`^[a-z]+\.[a-z]+@example\.(com|org|net)$`).MatchString(e) {
		t.Fatalf("email %q", e)
	}
	addr := g.Address("123 Main Street, Apt 4B, Richmond, VA 23220", 0)
	if !regexp.MustCompile(`^\d{3} [A-Z][a-z]+ Street, Apt \d[A-Z], [A-Z][a-z]+, VA \d{5}$`).MatchString(addr) {
		t.Fatalf("address %q", addr)
	}
	if a := g.Address("P.O. Box 1234", 0); !regexp.MustCompile(`^P\.O\. Box \d{4}$`).MatchString(a) {
		t.Fatalf("po box %q", a)
	}
	if o := g.Org("Initech LLC", 0); !strings.HasSuffix(o, " LLC") {
		t.Fatalf("org %q", o)
	}
	if ip := g.IP("10.1.2.3", 0); !regexp.MustCompile(`^(192\.0\.2|198\.51\.100|203\.0\.113)\.\d+$`).MatchString(ip) {
		t.Fatalf("ip %q", ip)
	}
	if ip := g.IP("fe80::1", 0); !strings.HasPrefix(ip, "2001:db8::") {
		t.Fatalf("ip6 %q", ip)
	}
	if s := g.Shape("gov_id", "AB-123", 0); !regexp.MustCompile(`^[A-Z]{2}-\d{3}$`).MatchString(s) {
		t.Fatalf("shape %q", s)
	}
}

func TestFallbackNamesAreUnbounded(t *testing.T) {
	g := NewGenerator([]byte("many"))
	seen := map[string]bool{}
	for i := 0; i < 5000; i++ {
		seen[g.GivenName("x", i)] = true
	}
	if len(seen) < 1000 {
		t.Fatalf("only %d distinct names across attempts", len(seen))
	}
}

func TestBundledListsHaveNoDuplicatesOrCommonWords(t *testing.T) {
	for _, list := range [][]string{givenNames, familyNames} {
		seen := map[string]bool{}
		for _, n := range list {
			l := strings.ToLower(n)
			if seen[l] {
				t.Fatalf("duplicate %q", n)
			}
			seen[l] = true
		}
	}
	for _, n := range givenNames {
		if fictionalFamily[strings.ToLower(n)] {
			t.Fatalf("%q is in both lists", n)
		}
	}
}
