package detect

import (
	"math"
	"math/big"
	"net"
	"strings"
	"unicode"
)

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func Luhn(number string) bool {
	d := digitsOnly(number)
	if len(d) < 12 {
		return false
	}
	sum := 0
	double := false
	for i := len(d) - 1; i >= 0; i-- {
		n := int(d[i] - '0')
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

func validCard(s string) bool {
	d := digitsOnly(s)
	if len(d) < 13 || len(d) > 19 {
		return false
	}
	if !strings.ContainsAny(d[:1], "23456") {
		return false
	}
	return Luhn(d)
}

func ValidIBAN(s string) bool {
	c := strings.ToUpper(strings.ReplaceAll(s, " ", ""))
	if len(c) < 15 || len(c) > 34 {
		return false
	}
	r := c[4:] + c[:4]
	var b strings.Builder
	for _, ch := range r {
		switch {
		case ch >= '0' && ch <= '9':
			b.WriteRune(ch)
		case ch >= 'A' && ch <= 'Z':
			b.WriteString(big.NewInt(int64(ch - 'A' + 10)).String())
		default:
			return false
		}
	}
	n, ok := new(big.Int).SetString(b.String(), 10)
	if !ok {
		return false
	}
	return new(big.Int).Mod(n, big.NewInt(97)).Int64() == 1
}

func ValidABA(s string) bool {
	d := digitsOnly(s)
	if len(d) != 9 {
		return false
	}
	w := []int{3, 7, 1, 3, 7, 1, 3, 7, 1}
	sum := 0
	for i := 0; i < 9; i++ {
		sum += int(d[i]-'0') * w[i]
	}
	return sum%10 == 0 && sum > 0
}

func validIPv4(s string) bool {
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() != nil && strings.Count(s, ".") == 3
}

func validIPv6(s string) bool {
	if strings.Count(s, ":") < 2 {
		return false
	}
	ip := net.ParseIP(s)
	return ip != nil && ip.To4() == nil
}

func ShannonEntropy(s string) float64 {
	if s == "" {
		return 0
	}
	freq := map[rune]float64{}
	n := 0.0
	for _, r := range s {
		freq[r]++
		n++
	}
	h := 0.0
	for _, c := range freq {
		p := c / n
		h -= p * math.Log2(p)
	}
	return h
}

func looksRandomToken(s string) bool {
	var upper, lower, digit bool
	hexOnly := true
	for _, r := range s {
		switch {
		case unicode.IsUpper(r):
			upper = true
		case unicode.IsLower(r):
			lower = true
		case unicode.IsDigit(r):
			digit = true
		}
		if !strings.ContainsRune("0123456789abcdefABCDEF-", r) {
			hexOnly = false
		}
	}
	if hexOnly || !digit || !(upper || lower) {
		return false
	}
	classes := 0
	for _, b := range []bool{upper, lower, digit} {
		if b {
			classes++
		}
	}
	if classes < 3 && len(s) < 32 {
		return false
	}
	return ShannonEntropy(s) >= 3.9
}
