package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestDigitStyleSurvivesCLIRoundTripsInBothOrders(t *testing.T) {
	styled := []string{
		"call \uff18\uff10\uff14-555-1234 now",
		"call 804-555-\uff11\uff12\uff13\uff14 now",
		"call \U0001D7D6\U0001D7CE\U0001D7D2-555-1234 now",
		"call \uff18\uff10\uff14-\uff15\uff15\uff15-\uff11\uff12\uff13\uff14 now",
	}
	ascii := "call 804-555-1234 today"
	for _, order := range [][]string{append([]string{ascii}, styled...), append(append([]string{}, styled...), ascii)} {
		cfg := setup(t)
		outs := map[string]string{}
		for _, in := range order {
			var masked bytes.Buffer
			if err := maskCmd([]string{"--config", cfg}, strings.NewReader(in), &masked); err != nil {
				t.Fatal(err)
			}
			outs[in] = masked.String()
		}
		for in, out := range outs {
			if out == in {
				t.Errorf("%q was not masked", in)
				continue
			}
			var back bytes.Buffer
			if err := unmaskCmd([]string{"--config", cfg}, strings.NewReader(out), &back); err != nil {
				t.Fatal(err)
			}
			if back.String() != in {
				t.Errorf("%q -> %q unmasks to %q", in, out, back.String())
			}
		}
	}
}
