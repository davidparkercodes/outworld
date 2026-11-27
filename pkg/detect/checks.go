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
	return cardPrefixOK(d) && Luhn(d)
}

func cardPrefixOK(d string) bool {
	n := len(d)
	p2 := num2(d, 0)
	switch d[0] {
	case '4':
		return n == 13 || n == 16 || n == 19
	case '5':
		return n == 16 || n >= 16 && (p2 == 50 || p2 >= 56)
	case '2':
		p4 := p2*100 + num2(d, 2)
		return n == 16 && p4 >= 2221 && p4 <= 2720
	case '3':
		switch p2 {
		case 34, 37:
			return n == 15
		case 30, 36, 38, 39:
			return n == 14
		case 35:
			return n == 16
		}
		return false
	case '6':
		return n >= 16
	}
	return false
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
	return ip != nil && (ip.To4() == nil || !strings.Contains(s, "."))
}

func validIPv6Embedded(s string) bool {
	return strings.Count(s, ":") >= 2 && strings.Count(s, ".") == 3 && net.ParseIP(s) != nil
}

var (
	ipLinkLocal4 = &net.IPNet{IP: net.IPv4(169, 254, 0, 0).To4(), Mask: net.CIDRMask(16, 32)}
	ipResolvers  = []net.IP{
		net.ParseIP("8.8.8.8"), net.ParseIP("8.8.4.4"), net.ParseIP("1.1.1.1"), net.ParseIP("1.0.0.1"), net.ParseIP("9.9.9.9"),
		net.ParseIP("2001:4860:4860::8888"), net.ParseIP("2001:4860:4860::8844"),
		net.ParseIP("2606:4700:4700::1111"), net.ParseIP("2606:4700:4700::1001"),
		net.ParseIP("2620:fe::fe"), net.ParseIP("2620:fe::9"),
	}
)

func nonPersonalIP(s string) bool {
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	for _, r := range ipResolvers {
		if ip.Equal(r) {
			return true
		}
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 127 || ipLinkLocal4.Contains(v4) || v4.Equal(net.IPv4zero.To4()) {
			return true
		}
		if v4[0] == 255 {
			if ones, bits := net.IPMask(v4).Size(); bits == 32 && ones >= 8 {
				return true
			}
		}
		return false
	}
	return ip.Equal(net.IPv6loopback) || ip.Equal(net.IPv6unspecified) || ip.To4() == nil && ip.IsLinkLocalUnicast()
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
