package corpus

import (
	"fmt"
	"math/rand/v2"
	"strings"
)

const base62 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"

func randFrom(r *rand.Rand, alphabet string, n int) string {
	var b strings.Builder
	for range n {
		b.WriteByte(alphabet[r.IntN(len(alphabet))])
	}
	return b.String()
}

func randDigits(r *rand.Rand, n int) string {
	return randFrom(r, "0123456789", n)
}

func apiKey(r *rand.Rand) (string, string) {
	switch r.IntN(4) {
	case 0:
		return "sk-" + randFrom(r, base62, 48), "sk-key"
	case 1:
		return "ghp_" + randFrom(r, base62, 36), "github-token"
	case 2:
		return "AKIA" + randFrom(r, "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567", 16), "aws-key-id"
	default:
		return randFrom(r, "0123456789abcdef", 32), "hex-key"
	}
}

func otp(r *rand.Rand) string {
	return fmt.Sprintf("%d%s", 1+r.IntN(9), randDigits(r, 5))
}

func resetLink(r *rand.Rand, domain string) (string, string) {
	token := randFrom(r, base62, 32)
	return "https://accounts." + domain + "/reset-password?token=" + token + "&uid=" + randDigits(r, 6), token
}

var pwWords = []string{"Plover", "Garnet", "Kettle", "Mossy", "Quartz", "Tundra", "Brine", "Fable", "Cobalt", "Juniper", "Lichen", "Saffron"}
var pwSymbols = []string{"!", "#", "$", "%", "&", "*", "@", "+"}

func password(r *rand.Rand) string {
	return pwWords[r.IntN(len(pwWords))] + pwSymbols[r.IntN(len(pwSymbols))] + pwWords[r.IntN(len(pwWords))] + randDigits(r, 2)
}

func ssn(r *rand.Rand) (string, string) {
	var area, group, serial string
	switch r.IntN(4) {
	case 0:
		area, group, serial = "987", "65", fmt.Sprintf("432%d", r.IntN(10))
	case 1:
		area, group, serial = "666", fmt.Sprintf("%02d", 1+r.IntN(98)), randDigits(r, 4)
	case 2:
		area, group, serial = fmt.Sprintf("%03d", 100+r.IntN(500)), "00", randDigits(r, 4)
	default:
		area, group, serial = fmt.Sprintf("%03d", 100+r.IntN(500)), fmt.Sprintf("%02d", 1+r.IntN(98)), "0000"
	}
	switch r.IntN(3) {
	case 0:
		return area + "-" + group + "-" + serial, "dashed"
	case 1:
		return area + " " + group + " " + serial, "spaced"
	default:
		return area + group + serial, "bare"
	}
}

var testCards = []string{
	"4111111111111111", "4242424242424242", "4000056655665556",
	"5555555555554444", "5105105105105100", "2223003122003222",
	"378282246310005", "371449635398431", "6011111111111117",
}

func luhnComplete(prefix string) string {
	for d := 0; d < 10; d++ {
		c := prefix + string(rune('0'+d))
		if Luhn(c) {
			return c
		}
	}
	panic("unreachable")
}

func card(r *rand.Rand) (string, string) {
	var n string
	if r.IntN(2) == 0 {
		n = testCards[r.IntN(len(testCards))]
	} else {
		n = luhnComplete("400000" + randDigits(r, 9))
	}
	if len(n) == 15 {
		switch r.IntN(2) {
		case 0:
			return n[:4] + " " + n[4:10] + " " + n[10:], "amex-spaced"
		default:
			return n, "bare"
		}
	}
	switch r.IntN(3) {
	case 0:
		return n[:4] + " " + n[4:8] + " " + n[8:12] + " " + n[12:], "spaced"
	case 1:
		return n[:4] + "-" + n[4:8] + "-" + n[8:12] + "-" + n[12:], "dashed"
	default:
		return n, "bare"
	}
}

func routing(r *rand.Rand) string {
	weights := []int{3, 7, 1, 3, 7, 1, 3, 7}
	d := fmt.Sprintf("%d%s", r.IntN(2), randDigits(r, 7))
	sum := 0
	for i := range 8 {
		sum += int(d[i]-'0') * weights[i]
	}
	return d + fmt.Sprintf("%d", (10-sum%10)%10)
}

func account(r *rand.Rand) string {
	return fmt.Sprintf("%d%s", 1+r.IntN(9), randDigits(r, 9+r.IntN(3)))
}
