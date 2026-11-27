package detect

import (
	"crypto/sha256"
	"math/big"
	"strconv"
	"strings"
)

func compactAlnum(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9', c >= 'A' && c <= 'Z':
			b.WriteByte(c)
		case c >= 'a' && c <= 'z':
			b.WriteByte(c - 32)
		}
	}
	return b.String()
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if !isDigitByte(s[i]) {
			return false
		}
	}
	return true
}

func dg(s string, i int) int { return int(s[i] - '0') }

func num2(s string, i int) int { return dg(s, i)*10 + dg(s, i+1) }

func weighted(d string, w ...int) int {
	sum := 0
	for i, x := range w {
		sum += dg(d, i) * x
	}
	return sum
}

func monthDayOK(mm, dd int) bool { return mm >= 1 && mm <= 12 && dd >= 1 && dd <= 31 }

func sameDigits(d string) bool { return strings.Trim(d, d[:1]) == "" }

func luhnAny(d string) bool {
	sum := 0
	double := false
	for i := len(d) - 1; i >= 0; i-- {
		n := dg(d, i)
		if double {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		double = !double
	}
	return sum%10 == 0
}

func mod11Check(sum int) int {
	r := 11 - sum%11
	if r == 11 {
		return 0
	}
	return r
}

var verhoeffD = [10][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 2, 3, 4, 0, 6, 7, 8, 9, 5}, {2, 3, 4, 0, 1, 7, 8, 9, 5, 6}, {3, 4, 0, 1, 2, 8, 9, 5, 6, 7}, {4, 0, 1, 2, 3, 9, 5, 6, 7, 8},
	{5, 9, 8, 7, 6, 0, 4, 3, 2, 1}, {6, 5, 9, 8, 7, 1, 0, 4, 3, 2}, {7, 6, 5, 9, 8, 2, 1, 0, 4, 3}, {8, 7, 6, 5, 9, 3, 2, 1, 0, 4}, {9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
}

var verhoeffP = [8][10]int{
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, {1, 5, 7, 6, 2, 8, 3, 0, 9, 4}, {5, 8, 0, 3, 7, 9, 6, 1, 4, 2}, {8, 9, 1, 6, 0, 4, 3, 5, 2, 7},
	{9, 4, 5, 3, 1, 2, 6, 8, 7, 0}, {4, 2, 8, 6, 5, 7, 3, 9, 0, 1}, {2, 7, 9, 3, 8, 0, 6, 4, 1, 5}, {7, 0, 4, 6, 9, 1, 3, 2, 5, 8},
}

func verhoeff(d string) bool {
	c := 0
	for i := 0; i < len(d); i++ {
		c = verhoeffD[c][verhoeffP[i%8][dg(d, len(d)-1-i)]]
	}
	return c == 0
}

func frNIR(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 15 || !strings.ContainsRune("12378", rune(c[0])) {
		return false
	}
	body := c[:13]
	switch c[5:7] {
	case "2A":
		body = c[:5] + "19" + c[7:13]
	case "2B":
		body = c[:5] + "18" + c[7:13]
	}
	if !allDigits(body) || !allDigits(c[13:]) || num2(c, 3) == 0 {
		return false
	}
	n, _ := strconv.ParseInt(body, 10, 64)
	return int(97-n%97) == num2(c, 13)
}

func deSteuerID(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 11 || !allDigits(c) || c[0] == '0' {
		return false
	}
	var freq [10]int
	for i := 0; i < 10; i++ {
		freq[dg(c, i)]++
	}
	repeated := 0
	for _, f := range freq {
		if f > 3 {
			return false
		}
		if f > 1 {
			repeated++
		}
	}
	if repeated != 1 {
		return false
	}
	product := 10
	for i := 0; i < 10; i++ {
		sum := (dg(c, i) + product) % 10
		if sum == 0 {
			sum = 10
		}
		product = sum * 2 % 11
	}
	check := 11 - product
	if check == 10 {
		check = 0
	}
	return check == dg(c, 10)
}

func deAusweis(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 10 || !isDigitByte(c[9]) || !hasDigit(c[:9]) {
		return false
	}
	w := []int{7, 3, 1}
	sum := 0
	for i := 0; i < 9; i++ {
		v := int(c[i] - '0')
		if c[i] >= 'A' {
			v = int(c[i]-'A') + 10
		}
		sum += v * w[i%3]
	}
	return sum%10 == dg(c, 9)
}

func nlBSN(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 9 || !allDigits(c) || sameDigits(c) {
		return false
	}
	return (weighted(c, 9, 8, 7, 6, 5, 4, 3, 2)-dg(c, 8))%11 == 0
}

func beNN(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 11 || !allDigits(c) || num2(c, 2)%20 > 12 || num2(c, 4) > 31 {
		return false
	}
	base, _ := strconv.ParseInt(c[:9], 10, 64)
	chk := int64(num2(c, 9))
	return 97-base%97 == chk || 97-(2000000000+base)%97 == chk
}

func ptNIF(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 9 || !allDigits(c) || !strings.ContainsRune("1235689", rune(c[0])) {
		return false
	}
	r := 11 - weighted(c, 9, 8, 7, 6, 5, 4, 3, 2)%11
	if r >= 10 {
		r = 0
	}
	return r == dg(c, 8)
}

func esCIF(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 9 || !strings.ContainsRune("ABCDEFGHJNPQRSUVW", rune(c[0])) || !allDigits(c[1:8]) {
		return false
	}
	sum := 0
	for i := 0; i < 7; i++ {
		d := dg(c, i+1)
		if i%2 == 0 {
			d *= 2
			d = d/10 + d%10
		}
		sum += d
	}
	ctrl := (10 - sum%10) % 10
	return c[8] == byte('0'+ctrl) || c[8] == "JABCDEFGHI"[ctrl]
}

func plPESEL(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 11 || !allDigits(c) || !monthDayOK(num2(c, 2)%20, num2(c, 4)) {
		return false
	}
	return (10-weighted(c, 1, 3, 7, 9, 1, 3, 7, 9, 1, 3)%10)%10 == dg(c, 10)
}

func sePersonnummer(raw string) bool {
	c := compactAlnum(raw)
	if len(c) == 12 {
		c = c[2:]
	}
	if len(c) != 10 || !allDigits(c) {
		return false
	}
	dd := num2(c, 4)
	if dd > 60 {
		dd -= 60
	}
	return monthDayOK(num2(c, 2), dd) && luhnAny(c)
}

func dkCPR(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 10 || !allDigits(c) || !monthDayOK(num2(c, 2), num2(c, 0)) {
		return false
	}
	return weighted(c, 4, 3, 2, 7, 6, 5, 4, 3, 2, 1)%11 == 0
}

func nordicID(raw string) bool { return sePersonnummer(raw) || dkCPR(raw) }

func noFnr(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 11 || !allDigits(c) {
		return false
	}
	dd, mm := num2(c, 0), num2(c, 2)
	if dd > 40 {
		dd -= 40
	}
	if mm > 40 {
		mm -= 40
	}
	if !monthDayOK(mm, dd) {
		return false
	}
	k1 := mod11Check(weighted(c, 3, 7, 6, 1, 8, 9, 4, 5, 2))
	k2 := mod11Check(weighted(c, 5, 4, 3, 2, 7, 6, 5, 4, 3, 2))
	return k1 == dg(c, 9) && k2 == dg(c, 10)
}

func fiHETU(raw string) bool {
	if len(raw) != 11 || !allDigits(raw[:6]) || !allDigits(raw[7:10]) || !monthDayOK(num2(raw, 2), num2(raw, 0)) {
		return false
	}
	n, _ := strconv.Atoi(raw[:6] + raw[7:10])
	return "0123456789ABCDEFHJKLMNPRSTUVWXY"[n%31] == raw[10]
}

func iePPS(raw string) bool {
	c := compactAlnum(raw)
	if len(c) < 8 || len(c) > 9 || !allDigits(c[:7]) {
		return false
	}
	sum := weighted(c, 8, 7, 6, 5, 4, 3, 2)
	if len(c) == 9 && c[8] != 'W' {
		sum += 9 * int(c[8]-'A'+1)
	}
	return "WABCDEFGHIJKLMNOPQRSTUV"[sum%23] == c[7]
}

func ean13(c string) bool {
	if len(c) != 13 || !allDigits(c) {
		return false
	}
	sum := 0
	for i := 0; i < 12; i++ {
		if i%2 == 0 {
			sum += dg(c, i)
		} else {
			sum += 3 * dg(c, i)
		}
	}
	return (10-sum%10)%10 == dg(c, 12)
}

func chAHV(raw string) bool {
	c := compactAlnum(raw)
	return strings.HasPrefix(c, "756") && ean13(c)
}

func atSVNR(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 10 || !allDigits(c) || c[0] == '0' || !monthDayOK(num2(c, 6), num2(c, 4)) {
		return false
	}
	return weighted(c, 3, 7, 9, 0, 5, 8, 4, 2, 1, 6)%11 == dg(c, 3)
}

func brCPF(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 11 || !allDigits(c) || sameDigits(c) {
		return false
	}
	if weighted(c, 10, 9, 8, 7, 6, 5, 4, 3, 2)*10%11%10 != dg(c, 9) {
		return false
	}
	return weighted(c, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2)*10%11%10 == dg(c, 10)
}

func brCNPJ(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 14 || !allDigits(c) || sameDigits(c) {
		return false
	}
	digit := func(sum int) int {
		if r := sum % 11; r >= 2 {
			return 11 - r
		}
		return 0
	}
	if digit(weighted(c, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)) != dg(c, 12) {
		return false
	}
	return digit(weighted(c, 6, 5, 4, 3, 2, 9, 8, 7, 6, 5, 4, 3, 2)) == dg(c, 13)
}

func mxCURP(raw string) bool {
	return len(raw) == 18 && allDigits(raw[4:10]) && monthDayOK(num2(raw, 6), num2(raw, 8))
}

func mxRFC(raw string) bool {
	c := compactAlnum(raw)
	i := len(c) - 9
	return i >= 3 && allDigits(c[i:i+6]) && monthDayOK(num2(c, i+2), num2(c, i+4))
}

func itCF(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 16 || !allDigits(c[9:11]) {
		return false
	}
	dd := num2(c, 9)
	if dd > 40 {
		dd -= 40
	}
	return dd >= 1 && dd <= 31
}

func arCUIL(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 11 || !allDigits(c) {
		return false
	}
	switch c[:2] {
	case "20", "23", "24", "27", "30", "33", "34":
	default:
		return false
	}
	return mod11Check(weighted(c, 5, 4, 3, 2, 7, 6, 5, 4, 3, 2)) == dg(c, 10)
}

func clRUT(raw string) bool {
	c := compactAlnum(raw)
	if len(c) < 8 || len(c) > 9 || !allDigits(c[:len(c)-1]) {
		return false
	}
	sum, m := 0, 2
	for i := len(c) - 2; i >= 0; i-- {
		sum += dg(c, i) * m
		if m++; m > 7 {
			m = 2
		}
	}
	want := byte('K')
	if r := mod11Check(sum); r < 10 {
		want = byte('0' + r)
	}
	return c[len(c)-1] == want
}

func luhn9(raw string) bool {
	c := compactAlnum(raw)
	return len(c) == 9 && allDigits(c) && !sameDigits(c) && luhnAny(c)
}

func auTFN(raw string) bool {
	c := compactAlnum(raw)
	return len(c) == 9 && allDigits(c) && !sameDigits(c) && weighted(c, 1, 4, 3, 7, 5, 8, 6, 9, 10)%11 == 0
}

func auMedicare(raw string) bool {
	c := compactAlnum(raw)
	if len(c) < 10 || len(c) > 11 || !allDigits(c) || c[0] < '2' || c[0] > '6' {
		return false
	}
	return weighted(c, 1, 3, 7, 9, 1, 3, 7, 9)%10 == dg(c, 8)
}

func nzIRD(raw string) bool {
	c := compactAlnum(raw)
	if len(c) == 8 {
		c = "0" + c
	}
	if len(c) != 9 || !allDigits(c) {
		return false
	}
	if n, _ := strconv.Atoi(c); n < 10000000 || n > 150000000 {
		return false
	}
	check := func(sum int) int {
		if r := sum % 11; r != 0 {
			return 11 - r
		}
		return 0
	}
	k := check(weighted(c, 3, 2, 7, 6, 5, 4, 3, 2))
	if k == 10 {
		k = check(weighted(c, 7, 4, 3, 2, 5, 2, 7, 6))
	}
	return k == dg(c, 8)
}

func ukNHS(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 10 || !allDigits(c) || sameDigits(c) {
		return false
	}
	return mod11Check(weighted(c, 10, 9, 8, 7, 6, 5, 4, 3, 2)) == dg(c, 9)
}

func inAadhaar(raw string) bool {
	c := compactAlnum(raw)
	return len(c) == 12 && allDigits(c) && c[0] >= '2' && verhoeff(c)
}

func jpMyNumber(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 12 || !allDigits(c) || sameDigits(c) {
		return false
	}
	sum := 0
	for n := 1; n <= 11; n++ {
		q := n + 1
		if n > 6 {
			q = n - 5
		}
		sum += dg(c, 11-n) * q
	}
	k := 0
	if r := sum % 11; r > 1 {
		k = 11 - r
	}
	return k == dg(c, 11)
}

func cnResidentID(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 18 || !allDigits(c[:17]) || !monthDayOK(num2(c, 10), num2(c, 12)) {
		return false
	}
	if y := num2(c, 6); y != 19 && y != 20 {
		return false
	}
	return "10X98765432"[weighted(c, 7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2)%11] == c[17]
}

func krRRN(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 13 || !allDigits(c) || !monthDayOK(num2(c, 2), num2(c, 4)) {
		return false
	}
	return (11-weighted(c, 2, 3, 4, 5, 6, 7, 8, 9, 2, 3, 4, 5)%11)%10 == dg(c, 12)
}

func sgNRIC(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 9 || !allDigits(c[1:8]) {
		return false
	}
	sum := weighted(c[1:], 2, 7, 6, 5, 4, 3, 2)
	if c[0] == 'T' || c[0] == 'G' {
		sum += 4
	}
	table := "JZIHGFEDCBA"
	if c[0] == 'F' || c[0] == 'G' {
		table = "XWUTRQPNMLK"
	}
	return table[sum%11] == c[8]
}

func hkHKID(raw string) bool {
	c := compactAlnum(raw)
	if len(c) < 8 || len(c) > 9 {
		return false
	}
	letters := len(c) - 7
	if !allDigits(c[letters : letters+6]) {
		return false
	}
	val := func(b byte) int { return int(b-'A') + 10 }
	sum := 36*9 + val(c[0])*8
	if letters == 2 {
		sum = val(c[0])*9 + val(c[1])*8
	}
	sum += weighted(c[letters:], 7, 6, 5, 4, 3, 2)
	k := 10
	if last := c[len(c)-1]; isDigitByte(last) {
		k = int(last - '0')
	} else if last != 'A' {
		return false
	}
	return (sum+k)%11 == 0
}

func zaID(raw string) bool {
	c := compactAlnum(raw)
	return len(c) == 13 && allDigits(c) && monthDayOK(num2(c, 2), num2(c, 4)) && c[10] <= '2' && luhnAny(c)
}

func trTCKN(raw string) bool {
	c := compactAlnum(raw)
	if len(c) != 11 || !allDigits(c) || c[0] == '0' {
		return false
	}
	odd := dg(c, 0) + dg(c, 2) + dg(c, 4) + dg(c, 6) + dg(c, 8)
	even := dg(c, 1) + dg(c, 3) + dg(c, 5) + dg(c, 7)
	if ((odd*7-even)%10+10)%10 != dg(c, 9) {
		return false
	}
	return (odd+even+dg(c, 9))%10 == dg(c, 10)
}

func clabeOK(c string) bool {
	if len(c) != 18 || !allDigits(c) {
		return false
	}
	w := []int{3, 7, 1}
	sum := 0
	for i := 0; i < 17; i++ {
		sum += dg(c, i) * w[i%3] % 10
	}
	return (10-sum%10)%10 == dg(c, 17)
}

const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

func base58Check(s string) bool {
	n := new(big.Int)
	radix := big.NewInt(58)
	zeros := 0
	leading := true
	for i := 0; i < len(s); i++ {
		v := strings.IndexByte(base58Alphabet, s[i])
		if v < 0 {
			return false
		}
		if leading && v == 0 {
			zeros++
		} else {
			leading = false
		}
		n.Mul(n, radix)
		n.Add(n, big.NewInt(int64(v)))
	}
	raw := append(make([]byte, zeros), n.Bytes()...)
	if len(raw) != 25 {
		return false
	}
	first := sha256.Sum256(raw[:21])
	second := sha256.Sum256(first[:])
	return string(second[:4]) == string(raw[21:])
}

var ibanLengths = map[string]int{
	"AD": 24, "AE": 23, "AL": 28, "AT": 20, "AZ": 28, "BA": 20, "BE": 16, "BG": 22, "BH": 22, "BR": 29, "BY": 28, "CH": 21, "CR": 22, "CY": 28, "CZ": 24,
	"DE": 22, "DK": 18, "DO": 28, "EE": 20, "EG": 29, "ES": 24, "FI": 18, "FO": 18, "FR": 27, "GB": 22, "GE": 22, "GI": 23, "GL": 18, "GR": 27, "GT": 28,
	"HR": 21, "HU": 28, "IE": 22, "IL": 23, "IQ": 23, "IS": 26, "IT": 27, "JO": 30, "KW": 30, "KZ": 20, "LB": 28, "LC": 32, "LI": 21, "LT": 20, "LU": 20,
	"LV": 21, "LY": 25, "MC": 27, "MD": 24, "ME": 22, "MK": 19, "MR": 27, "MT": 31, "MU": 30, "NL": 18, "NO": 15, "PK": 24, "PL": 28, "PS": 29, "PT": 25,
	"QA": 29, "RO": 24, "RS": 22, "SA": 24, "SC": 31, "SE": 24, "SI": 19, "SK": 24, "SM": 27, "ST": 25, "SV": 28, "TL": 23, "TN": 24, "TR": 26, "UA": 29,
	"VA": 22, "VG": 24, "XK": 20,
}
