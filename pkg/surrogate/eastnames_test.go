package surrogate

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

func TestEastPoolsAreCleanAndDisjoint(t *testing.T) {
	seen := map[string]bool{}
	for _, list := range eastPools {
		if len(list) < 12 {
			t.Fatalf("pool too small: %v", list)
		}
		for _, n := range list {
			l := strings.ToLower(n)
			if seen[l] {
				t.Errorf("duplicate %q", n)
			}
			seen[l] = true
			if len(n) < 4 {
				t.Errorf("%q is too short to restore safely in lowercase", n)
			}
			if fictionalGiven[l] || fictionalFamily[l] {
				t.Errorf("%q is also in the general lists", n)
			}
			if IsEastMarker(n) {
				t.Errorf("%q is a middle marker", n)
			}
		}
	}
}

var eastPools = [][]string{vietnameseFamilies, vietnameseGiven, chineseFamilies, chineseSyllables, chineseGiven, koreanFamilies, koreanSyllables}

func fileTokens(t *testing.T, path string) map[string]bool {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, w := range regexp.MustCompile(`\p{L}+`).FindAllString(strings.ToLower(string(data)), -1) {
		out[w] = true
	}
	return out
}

func TestEastPoolsAvoidPlacesTermsAndCommonSurnames(t *testing.T) {
	banned := map[string]string{}
	for _, w := range strings.Fields(`
hong ming qing minh viet thanh han tang song yuan kong zhou liao liang chen zhao wei jin sui shang xia chu yan
jiang huang tian chao xiang feng qiang jing xiao xian shan chang dong baoshan xinyi
binh dinh duong giang quang khanh chau trang phuc vinh ninh thai long truong hoang kieu trung huong phuong khoa hanh phan
kwon kwan seon`) {
		banned[w] = "a word in a well-known place, dynasty or term"
	}
	for _, w := range strings.Fields(`nguyen tran pham wang zhang deng yong kim lee park choi jung kang`) {
		banned[w] = "a very common real surname or a dictionary entry"
	}
	phrases := fileTokens(t, "../../internal/namedata/phrases.txt")
	geo := fileTokens(t, "../detect/geo.go")
	for _, list := range eastPools {
		for _, n := range list {
			l := strings.ToLower(n)
			if why, bad := banned[l]; bad {
				t.Errorf("%q is %s", n, why)
			}
			if phrases[l] {
				t.Errorf("%q is a token in phrases.txt", n)
			}
			if geo[l] {
				t.Errorf("%q is a token in the geo tables", n)
			}
		}
	}
}

func TestEastOrigin(t *testing.T) {
	cases := []struct {
		family string
		given  []string
		want   Origin
	}{
		{"Nguyen", []string{"Van", "An"}, OriginVietnamese},
		{"Tran", []string{"Mai"}, OriginVietnamese},
		{"Lee", []string{"Thi", "Hoa"}, OriginVietnamese},
		{"Kim", []string{"Min-jun"}, OriginKorean},
		{"Park", []string{"Ji", "Sung"}, OriginKorean},
		{"Ryu", []string{"Ji-hoon"}, OriginKorean},
		{"Zhang", []string{"Wei"}, OriginChinese},
		{"Li", []string{"Xiao-ming"}, OriginChinese},
		{"Sun", []string{"Yang"}, OriginChinese},
	}
	for _, c := range cases {
		if got := EastOrigin(c.family, c.given); got != c.want {
			t.Errorf("%s %v: got %d, want %d", c.family, c.given, got, c.want)
		}
	}
}

func TestEastNamesFollowOriginAndShape(t *testing.T) {
	g := NewGenerator([]byte("0123456789abcdef0123456789abcdef"))
	in := func(list []string, n string) bool {
		for _, x := range list {
			if x == n {
				return true
			}
		}
		return false
	}
	for attempt := 0; attempt < 30; attempt++ {
		if n := g.EastFamilyName(OriginVietnamese, "Nguyen", attempt); !in(vietnameseFamilies, n) {
			t.Errorf("vietnamese family %q", n)
		}
		if n := g.EastFamilyName(OriginKorean, "Kim", attempt); !in(koreanFamilies, n) {
			t.Errorf("korean family %q", n)
		}
		if n := g.EastFamilyName(OriginChinese, "Li", attempt); !in(chineseFamilies, n) {
			t.Errorf("chinese family %q", n)
		}
		if n := g.EastGivenName(OriginVietnamese, "An", 1, attempt); !in(vietnameseGiven, n) {
			t.Errorf("vietnamese given %q", n)
		}
		if n := g.EastGivenName(OriginChinese, "Wei", 1, attempt); !in(chineseSyllables, n) {
			t.Errorf("chinese syllable %q", n)
		}
		if n := g.EastGivenName(OriginChinese, "Xiaoming", 2, attempt); !in(chineseGiven, n) {
			t.Errorf("chinese given %q", n)
		}
		if n := g.EastGivenName(OriginKorean, "jun", 1, attempt); !in(koreanSyllables, n) {
			t.Errorf("korean syllable %q", n)
		}
		n := g.EastGivenName(OriginKorean, "Minjun", 2, attempt)
		if n != strings.ToUpper(n[:1])+strings.ToLower(n[1:]) || len(n) < 8 {
			t.Errorf("korean joined given %q", n)
		}
	}
	if a, b := g.EastFamilyName(OriginKorean, "Kim", 0), g.EastFamilyName(OriginKorean, "kim", 0); a != b {
		t.Errorf("not deterministic: %q %q", a, b)
	}
	if n := g.EastFamilyName(OriginKorean, "Kim", 10000); IsBundledEastName(n) || n == "" {
		t.Errorf("no fallback after the pool: %q", n)
	}
	if n := g.EastGivenName(OriginKorean, "Min", 1, 10000); IsBundledEastName(n) || n == "" {
		t.Errorf("no fallback after the pool: %q", n)
	}
}

func TestInitialNeverRepeatsTheRealLetter(t *testing.T) {
	g := NewGenerator([]byte("0123456789abcdef0123456789abcdef"))
	for _, real := range []string{"kchen", "k", "Karen", "mwilliams", "a", "z"} {
		got := g.Initial(real, real[:1])
		if len(got) != 1 || got[0] < 'a' || got[0] > 'z' || got[0] == strings.ToLower(real)[0] {
			t.Errorf("%q: got %q", real, got)
		}
		if got != g.Initial(real, real[:1]) {
			t.Errorf("%q: not deterministic", real)
		}
	}
}
