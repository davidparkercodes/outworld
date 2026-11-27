package main

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/davidparkercodes/outworld/eval/corpus"
)

func clean(s string) string { return strings.ToUpper(separators.Replace(s)) }

func digitsOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return s != ""
}

func ints(s string) []int {
	d := make([]int, 0, len(s))
	for i := 0; i < len(s); i++ {
		d = append(d, int(s[i]-'0'))
	}
	return d
}

func wsum(d []int, w ...int) int {
	sum := 0
	for i, x := range w {
		sum += d[i] * x
	}
	return sum
}

func numeric(n int, f func(d []int) bool) func(string) bool {
	return func(s string) bool {
		c := clean(s)
		return len(c) == n && digitsOnly(c) && f(ints(c))
	}
}

func luhnLen(n int) func(string) bool {
	return func(s string) bool {
		c := clean(s)
		return len(c) == n && digitsOnly(c) && corpus.Luhn(c)
	}
}

var ibanLen = map[string]int{
	"AE": 23, "AT": 20, "BE": 16, "BR": 29, "CH": 21, "CZ": 24, "DE": 22, "DK": 18, "EG": 29, "ES": 24, "FI": 18, "FR": 27,
	"GB": 22, "GR": 27, "IE": 22, "IL": 23, "IT": 27, "NL": 18, "NO": 15, "PK": 24, "PL": 28, "PT": 25, "SA": 24, "SE": 24, "TR": 26,
}

func ibanOK(s string) bool {
	c := clean(s)
	if len(c) < 15 || ibanLen[c[:2]] != len(c) || !digitsOnly(c[2:4]) || c[2:4] < "02" || c[2:4] > "98" {
		return false
	}
	mod := 0
	for _, ch := range c[4:] + c[:4] {
		switch {
		case ch >= '0' && ch <= '9':
			mod = (mod*10 + int(ch-'0')) % 97
		case ch >= 'A' && ch <= 'Z':
			mod = (mod*100 + int(ch-'A') + 10) % 97
		default:
			return false
		}
	}
	return mod == 1
}

const dniLetters = "TRWAGMYFPDXBNJZSQVHLCKE"

func dniOK(s string) bool {
	c := clean(s)
	if len(c) != 9 {
		return false
	}
	body := c[:8]
	if i := strings.IndexByte("XYZ", body[0]); i >= 0 {
		body = string(rune('0'+i)) + body[1:]
	}
	if !digitsOnly(body) {
		return false
	}
	n := 0
	for _, d := range ints(body) {
		n = n*10 + d
	}
	return dniLetters[n%23] == c[8]
}

var cfOdd = map[byte]int{'0': 1, '1': 0, '2': 5, '3': 7, '4': 9, '5': 13, '6': 15, '7': 17, '8': 19, '9': 21,
	'A': 1, 'B': 0, 'C': 5, 'D': 7, 'E': 9, 'F': 13, 'G': 15, 'H': 17, 'I': 19, 'J': 21, 'K': 2, 'L': 4, 'M': 18,
	'N': 20, 'O': 11, 'P': 3, 'Q': 6, 'R': 8, 'S': 12, 'T': 14, 'U': 16, 'V': 10, 'W': 22, 'X': 25, 'Y': 24, 'Z': 23}

func codiceFiscaleOK(s string) bool {
	c := clean(s)
	if len(c) != 16 {
		return false
	}
	sum := 0
	for i := 0; i < 15; i++ {
		ch := c[i]
		switch {
		case i%2 == 0:
			v, ok := cfOdd[ch]
			if !ok {
				return false
			}
			sum += v
		case ch >= '0' && ch <= '9':
			sum += int(ch - '0')
		case ch >= 'A' && ch <= 'Z':
			sum += int(ch - 'A')
		default:
			return false
		}
	}
	return byte('A'+sum%26) == c[15]
}

func mod11Check(sum int) int {
	c := 11 - sum%11
	if c == 11 {
		return 0
	}
	return c
}

var reHetu = regexp.MustCompile(`^(\d{6})([-+A])(\d{3})([0-9A-Y])$`)

func hetuOK(s string) bool {
	m := reHetu.FindStringSubmatch(strings.ToUpper(s))
	if m == nil {
		return false
	}
	n := 0
	for _, d := range ints(m[1] + m[3]) {
		n = n*10 + d
	}
	return "0123456789ABCDEFHJKLMNPRSTUVWXY"[n%31] == m[4][0]
}

func belgianOK(d []int) bool {
	base := 0
	for _, x := range d[:9] {
		base = base*10 + x
	}
	cc := d[9]*10 + d[10]
	return 97-base%97 == cc || 97-(2000000000+base)%97 == cc
}

func nirOK(d []int) bool {
	n := 0
	for _, x := range d[:13] {
		n = (n*10 + x) % 97
	}
	return 97-n == d[13]*10+d[14]
}

func steuerIDOK(d []int) bool {
	if d[0] == 0 {
		return false
	}
	product := 10
	for _, x := range d[:10] {
		sum := (x + product) % 10
		if sum == 0 {
			sum = 10
		}
		product = sum * 2 % 11
	}
	check := 11 - product
	if check == 10 {
		check = 0
	}
	return check == d[10]
}

func fnrOK(d []int) bool {
	k1 := mod11Check(wsum(d, 3, 7, 6, 1, 8, 9, 4, 5, 2))
	k2 := mod11Check(wsum(d, 5, 4, 3, 2, 7, 6, 5, 4, 3, 2))
	return k1 == d[9] && k2 == d[10]
}

func tcknOK(d []int) bool {
	odd, even, all := d[0]+d[2]+d[4]+d[6]+d[8], d[1]+d[3]+d[5]+d[7], 0
	for _, x := range d[:10] {
		all += x
	}
	return d[0] != 0 && ((odd*7-even)%10+10)%10 == d[9] && all%10 == d[10]
}

const curpAlphabet = "0123456789ABCDEFGHIJKLMNÑOPQRSTUVWXYZ"

func curpOK(s string) bool {
	r := []rune(strings.ToUpper(s))
	if len(r) != 18 {
		return false
	}
	alphabet := []rune(curpAlphabet)
	sum := 0
	for i := 0; i < 17; i++ {
		v := -1
		for k, a := range alphabet {
			if a == r[i] {
				v = k
			}
		}
		if v < 0 {
			return false
		}
		sum += v * (18 - i)
	}
	return rune('0'+(10-sum%10)%10) == r[17]
}

const rfcAlphabet = "0123456789ABCDEFGHIJKLMN&OPQRSTUVWXYZ Ñ"

func rfcOK(s string) bool {
	r := []rune(strings.ToUpper(s))
	if len(r) != 13 {
		return false
	}
	alphabet := []rune(rfcAlphabet)
	sum := 0
	for i := 0; i < 12; i++ {
		v := -1
		for k, a := range alphabet {
			if a == r[i] {
				v = k
			}
		}
		if v < 0 {
			return false
		}
		sum += v * (13 - i)
	}
	want := '0'
	if rem := sum % 11; rem != 0 {
		want = rune('0' + 11 - rem)
		if 11-rem == 10 {
			want = 'A'
		}
	}
	return want == r[12]
}

func mod10Weighted(d []int, w ...int) int {
	sum := 0
	for i, x := range w {
		sum += d[i] * x % 10
	}
	return (10 - sum%10) % 10
}

func cpfOK(d []int) bool {
	c1 := wsum(d, 10, 9, 8, 7, 6, 5, 4, 3, 2) * 10 % 11 % 10
	c2 := wsum(d, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2) * 10 % 11 % 10
	return c1 == d[9] && c2 == d[10]
}

func rutOK(s string) bool {
	c := clean(s)
	if len(c) < 8 || len(c) > 9 || !digitsOnly(c[:len(c)-1]) {
		return false
	}
	d := ints(c[:len(c)-1])
	sum, w := 0, 2
	for i := len(d) - 1; i >= 0; i-- {
		sum += d[i] * w
		if w++; w > 7 {
			w = 2
		}
	}
	want := "0123456789K"[mod11Check(sum)]
	return want == c[len(c)-1]
}

func cbuOK(d []int) bool {
	a := (10 - wsum(d, 7, 1, 3, 9, 7, 1, 3)%10) % 10
	b := (10 - wsum(d[8:], 3, 9, 7, 1, 3, 9, 7, 1, 3, 9, 7, 1, 3)%10) % 10
	return a == d[7] && b == d[21]
}

var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 2, 3, 4, 0, 6, 7, 8, 9, 5}, {2, 3, 4, 0, 1, 7, 8, 9, 5, 6}, {3, 4, 0, 1, 2, 8, 9, 5, 6, 7}, {4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
	{5, 9, 8, 7, 6, 0, 4, 3, 2, 1}, {6, 5, 9, 8, 7, 1, 0, 4, 3, 2}, {7, 6, 5, 9, 8, 2, 1, 0, 4, 3}, {8, 7, 6, 5, 9, 3, 2, 1, 0, 4}, {9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 5, 7, 6, 2, 8, 3, 0, 9, 4}, {5, 8, 0, 3, 7, 9, 6, 1, 4, 2}, {8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0}, {4, 2, 8, 6, 5, 7, 3, 9, 0, 1}, {2, 7, 9, 3, 8, 0, 6, 4, 1, 5}, {7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

func verhoeffOK(d []int) bool {
	c := 0
	for i := 0; i < len(d); i++ {
		c = verhoeffD[c][verhoeffP[i%8][d[len(d)-1-i]]]
	}
	return c == 0 && d[0] > 1
}

func chinaIDOK(s string) bool {
	c := clean(s)
	if len(c) != 18 || !digitsOnly(c[:17]) {
		return false
	}
	sum := wsum(ints(c[:17]), 7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2)
	return "10X98765432"[sum%11] == c[17]
}

func myNumberOK(d []int) bool {
	sum := 0
	for n := 1; n <= 11; n++ {
		q := n + 1
		if n > 6 {
			q = n - 5
		}
		sum += d[11-n] * q
	}
	check := 0
	if rem := sum % 11; rem > 1 {
		check = 11 - rem
	}
	return check == d[11]
}

func nricOK(s string) bool {
	c := clean(s)
	if len(c) != 9 || !digitsOnly(c[1:8]) {
		return false
	}
	sum := wsum(ints(c[1:8]), 2, 7, 6, 5, 4, 3, 2)
	table := "JZIHGFEDCBA"
	switch c[0] {
	case 'S':
	case 'T':
		sum += 4
	case 'F':
		table = "XWUTRQPNMLK"
	case 'G':
		sum += 4
		table = "XWUTRQPNMLK"
	default:
		return false
	}
	return table[sum%11] == c[8]
}

func hkidOK(s string) bool {
	c := strings.NewReplacer("(", "", ")", "").Replace(clean(s))
	if len(c) == 8 {
		c = " " + c
	}
	if len(c) != 9 || !digitsOnly(c[2:8]) {
		return false
	}
	val := func(b byte) int {
		if b == ' ' {
			return 36
		}
		return int(b-'A') + 10
	}
	sum := val(c[0])*9 + val(c[1])*8 + wsum(ints(c[2:8]), 7, 6, 5, 4, 3, 2)
	return "0123456789A"[(11-sum%11)%11] == c[8]
}

const twLetters = "ABCDEFGHJKLMNPQRSTUVXYWZIO"

func taiwanOK(s string) bool {
	c := clean(s)
	if len(c) != 10 || !digitsOnly(c[1:]) {
		return false
	}
	i := strings.IndexByte(twLetters, c[0])
	if i < 0 {
		return false
	}
	n := i + 10
	return (n/10+n%10*9+wsum(ints(c[1:]), 8, 7, 6, 5, 4, 3, 2, 1, 1))%10 == 0
}

func ppsOK(s string) bool {
	c := clean(s)
	if len(c) < 8 || len(c) > 9 || !digitsOnly(c[:7]) {
		return false
	}
	sum := wsum(ints(c[:7]), 8, 7, 6, 5, 4, 3, 2)
	if len(c) == 9 && c[8] != 'W' {
		sum += int(c[8]-'A'+1) * 9
	}
	return "WABCDEFGHIJKLMNOPQRSTUV"[sum%23] == c[7]
}

func irdOK(s string) bool {
	c := clean(s)
	if len(c) == 8 {
		c = "0" + c
	}
	if len(c) != 9 || !digitsOnly(c) {
		return false
	}
	d := ints(c)
	rem := wsum(d, 3, 2, 7, 6, 5, 4, 3, 2) % 11
	return (rem == 0 && d[8] == 0) || (rem != 0 && 11-rem == d[8])
}

func innOK(d []int) bool {
	a := wsum(d, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8) % 11 % 10
	b := wsum(d, 3, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8) % 11 % 10
	return a == d[10] && b == d[11]
}

func snilsOK(d []int) bool {
	sum := wsum(d, 9, 8, 7, 6, 5, 4, 3, 2, 1)
	if sum > 101 {
		sum %= 101
	}
	if sum == 100 || sum == 101 {
		sum = 0
	}
	return sum == d[9]*10+d[10]
}

var validators = map[string]func(string) bool{
	"es-dni":   dniOK,
	"es-nie":   dniOK,
	"it-cf":    codiceFiscaleOK,
	"nl-bsn":   numeric(9, func(d []int) bool { return (wsum(d, 9, 8, 7, 6, 5, 4, 3, 2)-d[8])%11 == 0 }),
	"pt-nif":   numeric(9, func(d []int) bool { return mod11Check(wsum(d, 9, 8, 7, 6, 5, 4, 3, 2))%10 == d[8] }),
	"pl-pesel": numeric(11, func(d []int) bool { return (10-wsum(d, 1, 3, 7, 9, 1, 3, 7, 9, 1, 3)%10)%10 == d[10] }),
	"se-pnr":   luhnLen(10),
	"fi-hetu":  hetuOK,
	"ch-ahv": numeric(13, func(d []int) bool {
		return d[0] == 7 && d[1] == 5 && d[2] == 6 && (10-wsum(d, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3, 1, 3)%10)%10 == d[12]
	}),
	"be-nn":       numeric(11, belgianOK),
	"fr-nir":      numeric(15, nirOK),
	"de-steuerid": numeric(11, steuerIDOK),
	"no-fnr":      numeric(11, fnrOK),
	"dk-cpr":      numeric(10, func(d []int) bool { return wsum(d, 4, 3, 2, 7, 6, 5, 4, 3, 2, 1)%11 == 0 }),
	"cz-rc": numeric(10, func(d []int) bool {
		n := 0
		for _, x := range d {
			n = (n*10 + x) % 11
		}
		return n == 0
	}),
	"tr-tckn":    numeric(11, tcknOK),
	"gb-nhs":     numeric(10, func(d []int) bool { return mod11Check(wsum(d, 10, 9, 8, 7, 6, 5, 4, 3, 2)) == d[9] }),
	"ie-pps":     ppsOK,
	"gr-amka":    luhnLen(11),
	"ru-inn":     numeric(12, innOK),
	"ru-snils":   numeric(11, snilsOK),
	"us-routing": numeric(9, func(d []int) bool { return wsum(d, 3, 7, 1, 3, 7, 1, 3, 7, 1)%10 == 0 }),
	"ca-sin":     luhnLen(9),
	"mx-curp":    curpOK,
	"mx-rfc":     rfcOK,
	"mx-clabe": numeric(18, func(d []int) bool {
		return mod10Weighted(d, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7, 1, 3, 7) == d[17]
	}),
	"br-cpf":      numeric(11, cpfOK),
	"cl-rut":      rutOK,
	"ar-cuil":     numeric(11, func(d []int) bool { return mod11Check(wsum(d, 5, 4, 3, 2, 7, 6, 5, 4, 3, 2)) == d[10] }),
	"ar-cbu":      numeric(22, cbuOK),
	"in-aadhaar":  numeric(12, verhoeffOK),
	"cn-rid":      chinaIDOK,
	"jp-mynumber": numeric(12, myNumberOK),
	"kr-rrn":      numeric(13, func(d []int) bool { return (11-wsum(d, 2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5)%11)%10 == d[12] }),
	"sg-nric":     nricOK,
	"hk-hkid":     hkidOK,
	"tw-id":       taiwanOK,
	"th-id":       numeric(13, func(d []int) bool { return (11-wsum(d, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2)%11)%10 == d[12] }),
	"il-id":       luhnLen(9),
	"ae-eid":      luhnLen(15),
	"za-id":       luhnLen(13),
	"au-tfn":      numeric(9, func(d []int) bool { return wsum(d, 1, 4, 3, 7, 5, 8, 6, 9, 10)%11 == 0 }),
	"au-medicare": numeric(10, func(d []int) bool { return d[0] >= 2 && d[0] <= 6 && wsum(d, 1, 3, 7, 9, 1, 3, 7, 9)%10 == d[8] }),
	"nz-ird":      irdOK,
	"is-kennitala": numeric(10, func(d []int) bool {
		return mod11Check(wsum(d, 3, 2, 7, 6, 5, 4, 3, 2)) == d[8] && (d[9] == 9 || d[9] == 0)
	}),
	"gr-afm": numeric(9, func(d []int) bool { return wsum(d, 256, 128, 64, 32, 16, 8, 4, 2)%11%10 == d[8] }),
}

func validatorFor(variant string) func(string) bool {
	switch {
	case strings.HasSuffix(variant, "-iban"):
		return ibanOK
	case strings.HasPrefix(variant, "card-"):
		return func(s string) bool { n := len(clean(s)); return n >= 13 && n <= 19 && corpus.Luhn(s) }
	}
	return validators[variant]
}

const mintAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

func mint(spec string) error {
	variant, pattern, ok := strings.Cut(spec, ":")
	if !ok {
		return fmt.Errorf("mint needs variant:pattern")
	}
	valid := validatorFor(variant)
	if valid == nil {
		return fmt.Errorf("no checksum is known for variant %q", variant)
	}
	holes := strings.Count(pattern, "?")
	if holes == 0 || holes > 3 {
		return fmt.Errorf("pattern needs one to three ? characters")
	}
	total := 1
	for i := 0; i < holes; i++ {
		total *= len(mintAlphabet)
	}
	for n := 0; n < total; n++ {
		k := n
		fill := make([]byte, holes)
		for i := holes - 1; i >= 0; i-- {
			fill[i] = mintAlphabet[k%len(mintAlphabet)]
			k /= len(mintAlphabet)
		}
		var sb strings.Builder
		h := 0
		for i := 0; i < len(pattern); i++ {
			if pattern[i] == '?' {
				sb.WriteByte(fill[h])
				h++
			} else {
				sb.WriteByte(pattern[i])
			}
		}
		if valid(sb.String()) {
			fmt.Println(sb.String())
			return nil
		}
	}
	return fmt.Errorf("no filling of %q passes the %s checksum", pattern, variant)
}
