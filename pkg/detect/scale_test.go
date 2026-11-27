package detect

import (
	"context"
	"fmt"
	"github.com/davidparkercodes/outworld/internal/scaletest"
	"os"
	"strings"
	"testing"
	"time"
)

type scaleInput struct {
	name string
	text string
}

func repeatTo(unit string, size int) string {
	return strings.Repeat(unit, size/len(unit)+1)[:size/len(unit)*len(unit)]
}

func envFile(size int) string {
	var b strings.Builder
	for i := 0; b.Len() < size; i++ {
		fmt.Fprintf(&b, "SERVICE_%d_TOKEN=tk%dQx9mZ2vLp8Rt\ntoken=ab%dCdEfGh1jKl\ndb_password = Pw%d!xYz9\napi_key: \"k%dZk3Lm9Qp2Xv7Bn4\"\nLOG_LEVEL=info\n", i, i, i, i, i)
	}
	return b.String()
}

func minifiedJSON(size int) string {
	var b strings.Builder
	b.WriteString(`{"items":[`)
	for i := 0; b.Len() < size; i++ {
		fmt.Fprintf(&b, `{"id":%d,"name":"Item %d","email":"user%d@example.com","phone":"+44 20 7946 %04d","card":"4111 1111 1111 1111","exp":"12/27","cvv":"123","token":"tk%dQx9mZ2vLp8Rt","password":"Pw%d!xYz9","note":"backup codes a1b2c-3d4e5 login admin","iban":"DE89 3704 0044 0532 0130 00","cpf":"123.456.789-09","addr":"12 High Street, Leeds"},`, i, i, i, i%10000, i, i)
	}
	b.WriteString(`{}]}`)
	return b.String()
}

const pemGrowthFactor = 2

func scaleInputs() []scaleInput { return scaleInputsAt(1) }

func scaleInputsAt(div int) []scaleInput {
	line := scaletest.Size(64<<10) / div
	return []scaleInput{
		{"env_512k", envFile(scaletest.Size(512<<10) / div)},
		{"json_256k", minifiedJSON(scaletest.Size(256<<10) / div)},
		{"token_kv_line", repeatTo("token=ab12CdEfGh1jKl ", line)},
		{"labelled_secret_line", repeatTo("secret is Xk9mQ2vLp8 ", line)},
		{"card_expiry_line", repeatTo("exp 12/27 ", line)},
		{"recovery_codes_line", repeatTo("backup codes a1b2c-3d4e5 ", line)},
		{"credential_cue_line", repeatTo("password Hunter2x! login ", line)},
		{"card_cvv_line", repeatTo("cvv 123 ", line)},
		{"national_id_line", repeatTo("cpf 12345678909 ", line)},
		{"password_shape_line", repeatTo("Tr0ub4dor&3xQ! ", line)},
		{"high_entropy_line", repeatTo("Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5Ga ", line)},
		{"hex_line", repeatTo("5d41402abc4b2a76b9719d911017c592 ", line)},
		{"phone_pairs_line", repeatTo("01 23 45 67 89 ", line)},
		{"labelled_id_line", repeatTo("Passport: A1234567 ", line)},
		{"pem_begin_line", repeatTo("--"+"---BEGIN RSA PRIVATE KEY----- ", line)},
		{"pem_begin_one_end", "--" + "---BEGIN RSA PRIVATE KEY-----\nabc\n-----END RSA PRIVATE KEY-----\n" + repeatTo("--"+"---BEGIN RSA PRIVATE KEY----- ", line)},
		{"email_no_space_line", repeatTo("user@example.com,", line)},
		{"digit_groups_line", repeatTo("4111 1111 1111 1111 ", line)},
		{"digit_run_line", repeatTo("4111111111111111", line)},
		{"intl_phone_line", repeatTo("+44 20 7946 0958 ", line)},
		{"cue_lines", repeatTo("password\nlogin\nbackup codes\n", line)},
		{"kv_long_value", "db_password=" + repeatTo("a1B2$key=", line)},
		{"url_line", repeatTo("https://example.com/reset?token=abc123 ", line)},
		{"pem_public_line", "-----BEGIN PUBLIC KEY-----\n" + repeatTo("Zk3Lm9Qp2Xv7Bn4Rt8Ws1Yc6Hd0Jf5Ga ", pemGrowthFactor*line)},
	}
}

func TestRegexDetectorScalesLinearly(t *testing.T) {
	if testing.Short() {
		t.Skip("scale test skipped in short mode")
	}
	d := NewRegexDetector()
	small := scaleInputsAt(scaletest.Input)
	for k, in := range scaleInputs() {
		spans, err := d.Detect(context.Background(), in.text)
		if err != nil {
			t.Fatalf("%s: %v", in.name, err)
		}
		if len(spans) > len(in.text)/4 {
			t.Errorf("%s: %d spans for %d bytes", in.name, len(spans), len(in.text))
		}
		k := k
		checkGrowth(t, in.name, func(n int) string {
			if n < len(in.text) {
				return small[k].text
			}
			return in.text
		}, len(in.text), func(text string) {
			if _, err := d.Detect(context.Background(), text); err != nil {
				t.Fatalf("%s: %v", in.name, err)
			}
		})
		if os.Getenv("OUTWORLD_SCALE_PROFILE") == "" {
			continue
		}
		lower := strings.ToLower(in.text)
		for i := range d.pats {
			ps := time.Now()
			n := len(d.pats[i].find(in.text, lower))
			if el := time.Since(ps); el > 200*time.Millisecond {
				t.Logf("    %-20s %6d spans %v", d.pats[i].name, n, el.Round(time.Millisecond))
			}
		}
	}
}
