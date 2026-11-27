package detect

import (
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func TestIDChecksums(t *testing.T) {
	cases := []struct {
		name  string
		check func(string) bool
		good  string
		bad   string
	}{
		{"fr_nir", frNIR, "2 84 05 75 116 042 17", "2 84 05 75 116 042 18"},
		{"fr_nir_corsica", frNIR, "1 91 11 2A 004 127 48", "1 91 11 2A 004 127 49"},
		{"de_steuerid", deSteuerID, "86095742719", "86095742710"},
		{"de_ausweis", deAusweis, "L01X00T471", "L01X00T472"},
		{"nl_bsn", nlBSN, "111222333", "111222334"},
		{"be_nn", beNN, "85.07.30-033.28", "85.07.30-033.29"},
		{"be_nn_2000", beNN, "01041227723", "01041227724"},
		{"pt_nif", ptNIF, "245318798", "245318799"},
		{"es_cif", esCIF, "B58219932", "B58219933"},
		{"pl_pesel", plPESEL, "44051401359", "44051401358"},
		{"se_personnummer", sePersonnummer, "850712-1237", "850712-1238"},
		{"dk_cpr", dkCPR, "070661-4117", "070661-4118"},
		{"no_fnr", noFnr, "15089231288", "15089231289"},
		{"fi_hetu", fiHETU, "120464-126J", "120464-126K"},
		{"ie_pps", iePPS, "1234567T", "1234567U"},
		{"ie_pps_two_letters", iePPS, "7654321PA", "7654321QA"},
		{"ch_ahv", chAHV, "756.4821.9037.57", "756.4821.9037.58"},
		{"at_svnr", atSVNR, "1237010180", "1238010180"},
		{"br_cpf", brCPF, "529.982.247-25", "529.982.247-26"},
		{"br_cnpj", brCNPJ, "11.222.333/0001-81", "11.222.333/0001-82"},
		{"ar_cuil", arCUIL, "27-28431907-4", "27-28431907-5"},
		{"cl_rut", clRUT, "14.582.937-2", "14.582.937-3"},
		{"ca_sin", luhn9, "046 454 286", "046 454 287"},
		{"au_tfn", auTFN, "123 456 782", "123 456 783"},
		{"au_medicare", auMedicare, "2123 45670 1", "2123 45671 1"},
		{"nz_ird", nzIRD, "049-091-850", "049-091-851"},
		{"uk_nhs", ukNHS, "943 476 5919", "943 476 5918"},
		{"in_aadhaar", inAadhaar, "4821 9375 0285", "4821 9375 0286"},
		{"jp_my_number", jpMyNumber, "4821 9375 0286", "4821 9375 0285"},
		{"cn_resident_id", cnResidentID, "11010519491231002X", "110105194912310021"},
		{"kr_rrn", krRRN, "920314-1234569", "920314-1234568"},
		{"sg_nric", sgNRIC, "S8219375H", "S8219375J"},
		{"sg_fin", sgNRIC, "F4821937M", "F4821937N"},
		{"hk_hkid", hkHKID, "K482193(9)", "K482193(8)"},
		{"hk_hkid_two_letters", hkHKID, "AB987654(3)", "AB987654(4)"},
		{"za_id", zaID, "9203145832086", "9203145832087"},
		{"tr_tckn", trTCKN, "48219375076", "48219375077"},
		{"mx_clabe", clabeOK, "032180000118359719", "032180000118359718"},
	}
	for _, c := range cases {
		if !c.check(c.good) {
			t.Errorf("%s: %q should validate", c.name, c.good)
		}
		if c.check(c.bad) {
			t.Errorf("%s: %q should not validate", c.name, c.bad)
		}
	}
	if !verhoeff("4821937502850"[:12]) || verhoeff("482193750286") {
		t.Error("verhoeff")
	}
	if !base58Check("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa") || base58Check("1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNb") {
		t.Error("base58check")
	}
}

func TestNationalIDsUnlabelled(t *testing.T) {
	runPositives(t, []classCase{
		{"elle a donné le 2 84 05 75 116 042 17 au guichet", entity.GovID, "2 84 05 75 116 042 17"},
		{"sécu: 284057511604217", entity.GovID, "284057511604217"},
		{"meine Nummer lautet 86095742719, danke", entity.GovID, "86095742719"},
		{"ricevuto da RSSMRA85T10A562S ieri", entity.GovID, "RSSMRA85T10A562S"},
		{"rijksregister 85.07.30-033.28 genoteerd", entity.GovID, "85.07.30-033.28"},
		{"she wrote 850712-1237 on the form", entity.GovID, "850712-1237"},
		{"han skrev 070661-4117 i feltet", entity.GovID, "070661-4117"},
		{"oppgitt 150892 31288 i skjemaet", entity.GovID, "150892 31288"},
		{"tunnus 120464-126J kirjattu", entity.GovID, "120464-126J"},
		{"she gave 1234567T at the desk", entity.GovID, "1234567T"},
		{"Versicherter 756.4821.9037.57 gemeldet", entity.GovID, "756.4821.9037.57"},
		{"cliente 529.982.247-25 confirmado", entity.GovID, "529.982.247-25"},
		{"fornecedor 11.222.333/0001-81 ativo", entity.GovID, "11.222.333/0001-81"},
		{"alta de GOMC850712HDFRRL09 hoy", entity.GovID, "GOMC850712HDFRRL09"},
		{"empleada 27-28431907-4 registrada", entity.GovID, "27-28431907-4"},
		{"arrendatario 14.582.937-2 firmó", entity.GovID, "14.582.937-2"},
		{"holder ABCPK4821L verified", entity.GovID, "ABCPK4821L"},
		{"居民 11010519491231002X 已登记", entity.GovID, "11010519491231002X"},
		{"resident 920314-1234569 enrolled", entity.GovID, "920314-1234569"},
		{"tenant S8219375H signed", entity.GovID, "S8219375H"},
		{"holder K482193(9) present", entity.GovID, "K482193(9)"},
		{"applicant 9203145832086 approved", entity.GovID, "9203145832086"},
	})
	runPositives(t, []classCase{
		{"mijn BSN is, zoals gevraagd, 111222333", entity.GovID, "111222333"},
		{"PESEL podany wczoraj: 44051401359", entity.GovID, "44051401359"},
		{"Aadhaar on the form reads 4821 9375 0285", entity.GovID, "4821 9375 0285"},
		{"her TFN, for the payroll form, is 123 456 782", entity.GovID, "123 456 782"},
		{"the IRD they have on record is 049-091-850", entity.GovID, "049-091-850"},
		{"NHS records show 943 476 5919 for her", entity.GovID, "943 476 5919"},
		{"TCKN olarak 48219375076 verildi", entity.GovID, "48219375076"},
		{"RFC de la empresa: GOMC850712AB1", entity.GovID, "GOMC850712AB1"},
		{"el RUT que dio fue 14582937-2", entity.GovID, "14582937-2"},
		{"Personalausweis liegt vor, Nummer L01X00T471", entity.GovID, "L01X00T471"},
		{"CIF de la sociedad, según registro, B58219932", entity.GovID, "B58219932"},
	})
	runNegatives(t, entity.GovID, []string{
		"elle a donné le 2 84 05 75 116 042 18 au guichet",
		"meine Nummer lautet 86095742710, danke",
		"she wrote 850712-1238 on the form",
		"oppgitt 150892 31289 i skjemaet",
		"tunnus 120464-126K kirjattu",
		"she gave 1234567U at the desk",
		"Versicherter 756.4821.9037.58 gemeldet",
		"cliente 529.982.247-26 confirmado",
		"empleada 27-28431907-5 registrada",
		"arrendatario 14.582.937-3 firmó",
		"tenant S8219375J signed",
		"holder K482193(8) present",
		"applicant 9203145832087 approved",
		"order 111222333 shipped",
		"invoice 44051401359 paid",
		"tracking 4821 9375 0285 delivered",
		"ticket 123 456 782 closed",
		"ref 049-091-850 archived",
		"lot GOMC850712AB1 passed QA",
		"part L01X00T471 in stock",
		"epoch_ms=1712251234567 offset=-28800",
		"timestamp 1741012930482 ms",
		"version 85.07.30-033.28-rc1",
		"build-15089231288 deployed",
		"sku/15089231288/black",
		"id #86095742719 created",
		"commit 86095742719a3f2c",
		"uuid 550e8400-e29b-41d4-a716-446655440000",
		"call 86095 742719 now",
		"row 12 86095742719 34",
		"card 4821 9375 0285 1234",
		"2024-05-12 850712-1237.log",
		"range 850712-1237-99",
		"model SM-G998B/DS firmware G998BXXU9FWK1",
		"flight LH2473 seat 23A",
		"ISBN 978-3-16-148410-0",
		"serial C02XK1ZZJGH5",
	})
}

func TestLabelledIDsMultilingual(t *testing.T) {
	runPositives(t, []classCase{
		{"renovación de pasaporte, el mío es el G20417755) y listo", entity.GovID, "G20417755"},
		{"Passport on file (passenger 1): N01847729", entity.GovID, "N01847729"},
		{"Tax ID (EIN): 00-4417826", entity.GovID, "00-4417826"},
		{"Reisepass Nr. C4FK9T2L8", entity.GovID, "C4FK9T2L8"},
		{"numéro de passeport : 19AB48213", entity.GovID, "19AB48213"},
		{"passaporto n. YA4821937", entity.GovID, "YA4821937"},
		{"Führerschein: B072RRE2I55", entity.GovID, "B072RRE2I55"},
		{"permis de conduire n° 120475300129", entity.GovID, "120475300129"},
		{"carnet de conducir 48219375-K", entity.GovID, "48219375-K"},
		{"Steuer-ID: 12 345 678 903", entity.GovID, "12 345 678 903"},
		{"Sozialversicherungsnummer 65 170839 J 008", entity.GovID, "65 170839 J 008"},
		{"numéro de sécurité sociale : 1 85 07 75 123 456 99", entity.GovID, "1 85 07 75 123 456 99"},
		{"codice fiscale: BNCLRA90A41H501X", entity.GovID, "BNCLRA90A41H501X"},
		{"cédula de ciudadanía 1.023.456.789", entity.GovID, "1.023.456.789"},
		{"CPF 123.456.789-00", entity.GovID, "123.456.789-00"},
		{"CURP: XEXX010101HNEXXXA4", entity.GovID, "XEXX010101HNEXXXA4"},
		{"Aadhaar: 9999 8888 7777", entity.GovID, "9999 8888 7777"},
		{"personnummer 19900101-0017", entity.GovID, "19900101-0017"},
		{"PESEL: 90010112345", entity.GovID, "90010112345"},
		{"NRIC T0012345Z", entity.GovID, "T0012345Z"},
		{"AHV-Nr. 756.1234.5678.90", entity.GovID, "756.1234.5678.90"},
		{"BSN: 123456789", entity.GovID, "123456789"},
		{"resident registration no. 900101-1234567", entity.GovID, "900101-1234567"},
	})
	runNegatives(t, entity.GovID, []string{
		"passport renewal takes 6-8 weeks; bring 2 photos (35x45 mm)",
		"Passport Office opening hours 08:30-16:00",
		"Driver's license renewal fee: $38.50, valid 8 years",
		"Tax year 2025-2026, filing deadline 2026-04-15",
		"The BSN is required on form 2026-A.",
		"PAN India rollout planned for Q3 2026",
		"The SIN wave has period 6.2832",
		"RUT 66 closed between exits 12 and 14",
		"CPF contributions rose to 37% in 2026",
		"NHS waiting list reached 7600000 in 2024",
		"Medicare Part B premium is 174.70 per month",
		"social security office: 8am to 4pm",
		"pasaporte válido hasta el 12/05/2031",
		"Reisepass gültig bis 12.05.2031",
		"passport issued 2019-05-01",
		"national identity and belonging in 1848",
		"tax id: required",
		"PAN 16GB RAM",
		"NIP/NIE required for non-residents",
	})
}

func TestLabelledIDMasksWholeValue(t *testing.T) {
	runPositives(t, []classCase{
		{"Personalausweisnummer: T22000129-3-6408125-2010315-D", entity.GovID, "T22000129-3-6408125-2010315-D"},
		{"Passport: A 1234567", entity.GovID, "A 1234567"},
		{"dowód osobisty: ABC 123456", entity.GovID, "ABC 123456"},
		{"Passport NO 1234567 issued", entity.GovID, "1234567"},
		{"Passport NR: X1234567", entity.GovID, "X1234567"},
		{"Reisepass C01X00T47, gültig", entity.GovID, "C01X00T47"},
	})
	runNegatives(t, entity.GovID, []string{
		"passport, USA 2019 edition",
		"Personalausweisnummer: T22000129-3-6408125-2010315-D-AAAAAAAAAAAAAAAAAAAAAAAA-3-6408125-2010315",
		"passport photo taken 12/05/2019 at B 12",
		"Passport: see ID 4 of 10",
	})
}

func TestIDFollowedByUnrelatedNumber(t *testing.T) {
	runPositives(t, []classCase{
		{"Codice fiscale RSSMRA85T10A562S 12 anni", entity.GovID, "RSSMRA85T10A562S"},
		{"Führerschein B072RRE2I55 2 Punkte", entity.GovID, "B072RRE2I55"},
		{"RSSMRA85T10A562S, 3 figli", entity.GovID, "RSSMRA85T10A562S"},
	})
	for _, c := range []struct {
		text string
		e    int
		want bool
	}{
		{"AB123456 7", 8, true},
		{"AB123456, 7", 8, true},
		{"AB123456-7", 8, false},
		{"AB123456/7", 8, false},
		{"AB123456.7", 8, false},
		{"12345678 9", 8, false},
		{"12 345 678 901 2", 14, false},
		{"A 1234567 8", 9, false},
	} {
		if got := idValueEnd(c.text, 0, c.e); got != c.want {
			t.Errorf("idValueEnd(%q, %d) = %v, want %v", c.text, c.e, got, c.want)
		}
	}
}

func TestPANWithCardNumberIsACard(t *testing.T) {
	for _, text := range []string{"PAN: 4111111111111111", "PAN 4111 1111 1111 1111", "PAN: 378282246310005"} {
		spans := detectAll(t, text)
		if len(spanTexts(spans, entity.GovID)) != 0 || len(spanTexts(spans, entity.Financial)) != 1 {
			t.Errorf("%q: want one card and no ID, got %+v", text, spans)
		}
	}
	runPositives(t, []classCase{
		{"PAN: ABCPE1234F", entity.GovID, "ABCPE1234F"},
		{"PAN: 4111111111111112", entity.GovID, "4111111111111112"},
		{"Passport: 4111111111111111", entity.GovID, "4111111111111111"},
	})
}
