package outworld

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

var styledPhoneForms = []string{
	"call \uff18\uff10\uff14-555-1234 now",
	"call 804-555-\uff11\uff12\uff13\uff14 now",
	"call \U0001D7D6\U0001D7CE\U0001D7D2-555-1234 now",
	"call \uff18\uff10\uff14-\uff15\uff15\uff15-\uff11\uff12\uff13\uff14 now",
	"call \uff18\uff10\uff14\u30fb555\u2009\u0661\u0662\u0663\u0664 now",
	"call 8\uff10\u0664 555.12\U0001D7D1\u0664 now",
	"call \u0968\u0966\u096a\u2013\u096b\u096b\u096b\u2013\u0967\u0968\u0969\u096a now",
	"call \u2078\u2070\u2074-\u2075\u2075\u2075-\u00b9\u00b2\u00b3\u2074 now",
	"Tel\uff18\uff10\uff14\uff15\uff15\uff15\uff11\uff12\uff13\uff14",
}

func TestDigitStyleRoundTripsInBothOrders(t *testing.T) {
	ascii := []string{"call 804-555-1234 today", "call 8045551234 now", "x 804.555.1234 y"}
	for _, order := range []string{"ascii first", "styled first"} {
		for _, mode := range []string{"seed", "history"} {
			path := filepath.Join(t.TempDir(), "v.vault")
			key := bytes.Repeat([]byte{7}, 32)
			v, err := openTestVault(path, key)
			if err != nil {
				t.Fatal(err)
			}
			e, err := New(Options{Vault: v, Policy: DefaultPolicy()})
			if err != nil {
				t.Fatal(err)
			}
			if mode == "history" {
				phoneHistory(t, e, 60)
			}
			texts := append(append([]string{}, ascii...), styledPhoneForms...)
			if order == "styled first" {
				texts = append(append([]string{}, styledPhoneForms...), ascii...)
			}
			outs := map[string]string{}
			for _, in := range texts {
				out, err := e.NewSession("s0").Mask(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				if digitValues(out) == digitValues(in) || out == in {
					t.Errorf("%s %s: %q -> %q is not masked", order, mode, in, out)
					continue
				}
				outs[in] = out
			}
			for in, out := range outs {
				if back := e.NewSession("s0").Rehydrate(out); back != in {
					t.Errorf("%s %s: %q -> %q restores %q", order, mode, in, out, back)
					continue
				}
				for size := 1; size <= 24; size++ {
					if got := streamed(e.NewSession("s0"), out, []int{size}); got != in {
						t.Errorf("%s %s: %q -> %q streamed at %d gives %q", order, mode, in, out, size, got)
						break
					}
				}
			}
			if err := v.Close(); err != nil {
				t.Fatal(err)
			}
			v2, err := openTestVault(path, key)
			if err != nil {
				t.Fatal(err)
			}
			e2, err := New(Options{Vault: v2, Policy: DefaultPolicy()})
			if err != nil {
				t.Fatal(err)
			}
			for in, out := range outs {
				if back := e2.NewSession("s0").Rehydrate(out); back != in {
					t.Errorf("%s %s after reopen: %q -> %q restores %q", order, mode, in, out, back)
				}
			}
			v2.Close()
		}
	}
}

func TestStyleDigitKeepsEachDigitStyle(t *testing.T) {
	for _, like := range []rune{'7', 0xFF17, 0x0667, 0x0967, 0x1D7D5, 0x1D7DF, 0x00B9, 0x2077, 0x2087, 0x09ED} {
		for d := 0; d < 10; d++ {
			r := textutil.StyleDigit(like, d)
			if v, ok := textutil.DigitValue(r); !ok || v != d {
				t.Errorf("style of U+%04X with %d gives U+%04X worth %d", like, d, r, v)
			}
		}
	}
}
