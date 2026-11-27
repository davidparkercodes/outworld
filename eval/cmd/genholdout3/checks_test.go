package main

import "testing"

func TestValidatorsAcceptPublishedExamples(t *testing.T) {
	examples := map[string][]string{
		"au-tfn":      {"123 456 782"},
		"gb-nhs":      {"943 476 5919"},
		"sg-nric":     {"S1234567D"},
		"hk-hkid":     {"A123456(3)"},
		"tw-id":       {"A123456789"},
		"br-cpf":      {"529.982.247-25"},
		"gb-iban":     {"GB82 WEST 1234 5698 7654 32"},
		"de-iban":     {"DE89 3704 0044 0532 0130 00"},
		"pl-pesel":    {"44051401359"},
		"nl-bsn":      {"111222333"},
		"es-dni":      {"12345678Z"},
		"es-nie":      {"X1234567L"},
		"it-cf":       {"RSSMRA85T10A562S"},
		"mx-curp":     {"HEGG560427MVZRRL04"},
		"ch-ahv":      {"756.9217.0769.85"},
		"de-steuerid": {"86095742719", "47036892816"},
		"fr-nir":      {"2 69 05 49 588 157 80"},
		"cl-rut":      {"12.345.678-5"},
		"ar-cuil":     {"20-12345678-6"},
		"ca-sin":      {"046 454 286"},
		"fi-hetu":     {"131052-308T"},
		"au-medicare": {"2123 45670 1"},
		"nz-ird":      {"49-091-850"},
		"ie-pps":      {"1234567T"},
		"cn-rid":      {"11010519491231002X"},
		"mx-clabe":    {"032180000118359719"},
		"ru-snils":    {"112-233-445 95"},
		"ru-inn":      {"500100732259"},
		"tr-tckn":     {"10000000146"},
		"card-visa":   {"4242 4242 4242 4242"},
		"in-aadhaar":  {"2341 2341 2346"},
	}
	for variant, values := range examples {
		valid := validatorFor(variant)
		if valid == nil {
			t.Errorf("no validator for %s", variant)
			continue
		}
		for _, v := range values {
			if !valid(v) {
				t.Errorf("%s rejects the published example %q", variant, v)
			}
			b := []byte(v)
			for i := len(b) - 1; i >= 0; i-- {
				if b[i] >= '0' && b[i] <= '8' {
					b[i]++
					break
				}
			}
			if valid(string(b)) && string(b) != v {
				t.Logf("%s also accepts the altered value %q", variant, b)
			}
		}
	}
}
