package detect

import (
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

type classCase struct {
	text string
	typ  entity.Type
	want string
}

func runPositives(t *testing.T, cases []classCase) {
	t.Helper()
	for _, c := range cases {
		spans := detectAll(t, c.text)
		if !hasSpan(spans, c.typ, c.want) {
			t.Errorf("%q: expected %s %q, got %+v", c.text, c.typ, c.want, spans)
		}
	}
}

func runNegatives(t *testing.T, typ entity.Type, texts []string) {
	t.Helper()
	for _, text := range texts {
		for _, s := range detectAll(t, text) {
			if s.Type == typ {
				t.Errorf("%q: unexpected %s %q from %s", text, typ, s.Text, s.Source)
			}
		}
	}
}

func TestNationalPhones(t *testing.T) {
	runPositives(t, []classCase{
		{"ring me on 07911 123456 tonight", entity.Phone, "07911 123456"},
		{"Office: 020 7123 4567.", entity.Phone, "020 7123 4567"},
		{"office line (020) 7123 4567", entity.Phone, "(020) 7123 4567"},
		{"Tel. 030 12345 678", entity.Phone, "030 12345 678"},
		{"Telefon 089-1234-5678", entity.Phone, "089-1234-5678"},
		{"appelez le 01 23 45 67 89 demain", entity.Phone, "01 23 45 67 89"},
		{"tél: 06.12.34.56.78", entity.Phone, "06.12.34.56.78"},
		{"mobile 0412 345 678, after six", entity.Phone, "0412 345 678"},
		{"bel 020 123 4567", entity.Phone, "020 123 4567"},
		{"dial 0044 20 7123 4567 from abroad", entity.Phone, "0044 20 7123 4567"},
		{"ring +44 7911 123456 please", entity.Phone, "+44 7911 123456"},
		{"call +49 30 12345 678", entity.Phone, "+49 30 12345 678"},
		{"fax +33 1 23 45 67 89", entity.Phone, "+33 1 23 45 67 89"},
		{"cell +61 412 345 678", entity.Phone, "+61 412 345 678"},
		{"Tel:07911 123456", entity.Phone, "07911 123456"},
	})
	runNegatives(t, entity.Phone, []string{
		"released 01.02.2024 in Berlin",
		"on 01-02-2024 we met",
		"dated 03 04 2021 0915",
		"host 010.020.030.040 is up",
		"version 0.12.345.6789 shipped",
		"build 02.10.33.4571",
		"total 0.123 456 789 units",
		"price $0412 345 678",
		"cost 1,020 7123 4567",
		"order 0001-2345-6789 shipped",
		"draw 01 02 03 04 05",
		"rows 00 01 02 03 04 05 06 07",
		"steps 02 04 06 08 10",
		"at 2024-05-01 09:30:00 0200",
		"id 00000 000000",
		"hash 0a12 3456 7890",
		"ratio 0412 345 678%",
		"file_0412 345 678",
		"0791 1123",
		"sku 0412-345-678-901-234",
		"Boston, MA 02110-1301 USA",
		"00000010  03 42 00 04 05 16 a0 ff  04 04 03 02 07 80 1c 2b",
		"bytes 04 04 03 02 07 80 0a 0b",
		"00000050  04 02 05 02 06 02",
		"0000000 141 142 143 012 041 117 132",
		"     041 117 132",
		"7f0a1000-7f0a2000 r-xp 00000000 fd:01 1318064 /usr/lib/libx.so",
	})
}

func TestSpokenPhones(t *testing.T) {
	runPositives(t, []classCase{
		{"my number is four one five five five five oh one four two ok", entity.Phone, "four one five five five five oh one four two"},
		{"call Eight Zero Four, two two two, nine one three seven now", entity.Phone, "Eight Zero Four, two two two, nine one three seven"},
		{"it's seven double five nine three one two", entity.Phone, "seven double five nine three one two"},
		{"dial zero seven nine one one one two three four five six", entity.Phone, "zero seven nine one one one two three four five six"},
		{"nine-one-seven-five-five-five-two-zero-four-six", entity.Phone, "nine-one-seven-five-five-five-two-zero-four-six"},
	})
	runNegatives(t, entity.Phone, []string{
		"count one two three four five six seven eight nine",
		"nine eight seven six five four three two one liftoff",
		"one two three, go",
		"pick two of five or six of nine",
		"oh one more thing, two or three of us",
		"zero zero zero zero zero zero zero",
		"one by one, four by four, nine to five",
	})
}

func TestLabelledIDs(t *testing.T) {
	runPositives(t, []classCase{
		{"Passport no: 533401872", entity.GovID, "533401872"},
		{"my passport number is LK4482019", entity.GovID, "LK4482019"},
		{"passport # P-7719-2284", entity.GovID, "P-7719-2284"},
		{"Driver's license: D410-7782-9931", entity.GovID, "D410-7782-9931"},
		{"driving licence number MORGA753116SM9IJ", entity.GovID, "MORGA753116SM9IJ"},
		{"DL# W4471902", entity.GovID, "W4471902"},
		{"EIN 47-2918834 on file", entity.GovID, "47-2918834"},
		{"Tax ID: 84-1173390", entity.GovID, "84-1173390"},
		{"TIN is 931-44-0182", entity.GovID, "931-44-0182"},
		{"national ID 880412-5521", entity.GovID, "880412-5521"},
		{"DNI: 48211973K", entity.GovID, "48211973K"},
		{"NIE X7719024B", entity.GovID, "X7719024B"},
		{"NIF B-58219934", entity.GovID, "B-58219934"},
		{"National Insurance number: JG 10 37 59 A", entity.GovID, "JG 10 37 59 A"},
		{"NINO QQ123456C", entity.GovID, "QQ123456C"},
		{"A-number A208774519", entity.GovID, "A208774519"},
		{"alien registration number 208-774-519", entity.GovID, "208-774-519"},
		{"SIN 046 454 286", entity.GovID, "046 454 286"},
		{"Medicare number 2123 45670 1", entity.GovID, "2123 45670 1"},
		{"medicare id 1EG4-TE5-MK73", entity.GovID, "1EG4-TE5-MK73"},
		{"MRN: 00482913", entity.GovID, "00482913"},
		{"Policy number is HX-4471-9920", entity.Financial, "HX-4471-9920"},
		{"member id = 9920441873", entity.Financial, "9920441873"},
		{"mi documento es 12345678Z gracias", entity.GovID, "12345678Z"},
		{"documento 48211973K adjunto", entity.GovID, "48211973K"},
		{"residente Y1234567X desde 2019", entity.GovID, "Y1234567X"},
		{"her number is AB 12 34 56 C", entity.GovID, "AB 12 34 56 C"},
		{"insurance no AB-12-34-56-C", entity.GovID, "AB-12-34-56-C"},
	})
	runNegatives(t, entity.GovID, []string{
		"passport number required at check-in",
		"my passport expires 2031-04-12",
		"driver's license renewal takes 10 business days",
		"MIT license 2019-2024",
		"call Medicare at 1-800-633-4227",
		"sin(123456) returns a float",
		"the tin can holds 500 grams",
		"EIN 12-34",
		"12-3456789 without any label",
		"build 20240115A deployed",
		"part-12345678B fits",
		"id=12345678C",
		"mass 12345678E+5",
		"X1234567A bracket",
		"DL 380 server",
		"national id card renewal",
	})
}

func TestLabelledSecrets(t *testing.T) {
	runPositives(t, []classCase{
		{"password: Marlin#Quay27", entity.Secret, "Marlin#Quay27"},
		{"passwd=tr0ub4dor&3", entity.Secret, "tr0ub4dor&3"},
		{"pwd is Otter-Canal-92", entity.Secret, "Otter-Canal-92"},
		{"the passphrase is copper-lantern-seven-orbit", entity.Secret, "copper-lantern-seven-orbit"},
		{"pass: hunter2xyz", entity.Secret, "hunter2xyz"},
		{"wifi pass = Gull!Harbor9", entity.Secret, "Gull!Harbor9"},
		{"the secret is Zephyr$4411.", entity.Secret, "Zephyr$4411"},
		{"token: a8Kq2mZx91", entity.Secret, "a8Kq2mZx91"},
		{"your token is 7Hq2-99Lm-x0Pd", entity.Secret, "7Hq2-99Lm-x0Pd"},
		{"api key = QX7-22-LMNOP", entity.Secret, "QX7-22-LMNOP"},
		{"API key is zx81kq02mm", entity.Secret, "zx81kq02mm"},
		{`{"refresh_token": "r9Tq0Lm2Xa"}`, entity.Secret, "r9Tq0Lm2Xa"},
		{"secret: HorseBatteryStaple", entity.Secret, "HorseBatteryStaple"},
		{"PIN: 4821", entity.Secret, "4821"},
		{"my pin is 482913", entity.Secret, "482913"},
		{"signing key 5d41402abc4b2a76b9719d911017c592 leaked", entity.Secret, "5d41402abc4b2a76b9719d911017c592"},
		{"use 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 to decrypt", entity.Secret, "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12"},
		{"x-auth: 9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08", entity.Secret, "9F86D081884C7D659A2FEAA0C55AD015A3BF4F1B2B0B822CD15D6C15B0F00A08"},
		{"value=\"e99a18c428cb38d5f260853678922e03\"", entity.Secret, "e99a18c428cb38d5f260853678922e03"},
	})
	runNegatives(t, entity.Secret, []string{
		"the password is required",
		"the password is case-sensitive",
		"password is optional here",
		"the token is base64-encoded",
		"the token is JWT-based and short lived",
		"the secret is OpenSSL generated",
		"token = response.json()[\"token\"]",
		"token = getToken",
		"token: process.env.TOKEN",
		"secret = os.environ",
		"token = api_token_v2",
		"secret: app-secret",
		"token: ${{ secrets.DEPLOY }}",
		"token: <your-token-here>",
		"pass = $PASSWORD",
		"token=********",
		"password: ********",
		"WHERE token = :token AND pass = ?",
		"secret = generateSecret(32)",
		"token: v1.24.3",
		"token is 2024-01-15T10:00:00Z",
		"secret is sha256 of the body",
		"pass: true",
		"token: string;",
		"the secret is simple",
		"pass 1 of 3 completed",
		"--- PASS: TestShuffle/case_1 (0.00s)",
		"PASS: TestOrphanCmd",
		"pass: TestOne",
		"until the token is encountered.”",
		"commit 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
		"commit 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12\nAuthor: build bot\n\n    fix flaky test",
		"2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 fix flaky test",
		"# 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 30 472 7",
		"Regression from 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
		"  * Upstream change 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
		"mask ffffffffffffffff0000000000000000",
		"group ff0200000000000000000001ffac891e joined",
		"Merge 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12 into main",
		"git checkout 2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
		"sha256: 9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08  release.tar.gz",
		"image@sha256:9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08",
		"MD5 (notes.txt) = 5d41402abc4b2a76b9719d911017c592",
		"checksum = \"9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08\"",
		"trace_id=5d41402abc4b2a76b9719d911017c592 status=200",
		"\"request_id\": \"5d41402abc4b2a76b9719d911017c592\"",
		"/assets/app.5d41402abc4b2a76b9719d911017c592.js",
		"id 550e8400-e29b-41d4-a716-446655440000",
		"00000000000000000000000000000000",
		"abcdefabcdefabcdefabcdefabcdefab",
		"5d41402abc4b2a76b9719d911017c5925d41402abc4b2a76b9719d911017",
	})
}

func TestBareHexRule(t *testing.T) {
	const hex = "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12"
	masked := []string{
		"the value " + hex + " was pasted in chat",
		"key: " + hex,
		"  " + hex,
	}
	for _, text := range masked {
		if !hasSpan(detectAll(t, text), entity.Secret, hex) {
			t.Errorf("%q: bare hex outside git or checksum context must be a secret", text)
		}
	}
	kept := []string{
		"commit " + hex,
		hex + " Merge branch 'main'",
		"parent " + hex,
		"sha1 " + hex,
		hex + "  archive.zip",
	}
	for _, text := range kept {
		if anyType(detectAll(t, text), entity.Secret) {
			t.Errorf("%q: git or checksum context must be left alone", text)
		}
	}
}

func TestInternationalAddresses(t *testing.T) {
	runPositives(t, []classCase{
		{"livré au 14 rue des Lilas, 69003 Lyon hier", entity.Address, "14 rue des Lilas, 69003 Lyon"},
		{"j'habite 7 bis avenue de l'Europe", entity.Address, "7 bis avenue de l'Europe"},
		{"at 3 place Bellecour, 69002 Lyon.", entity.Address, "3 place Bellecour, 69002 Lyon"},
		{"vivo en Calle Alcalde Moreno 21, 41004 Sevilla desde mayo", entity.Address, "Calle Alcalde Moreno 21, 41004 Sevilla"},
		{"Carrer de Balmes 118, 08008 Barcelona", entity.Address, "Carrer de Balmes 118, 08008 Barcelona"},
		{"Avenida da Liberdade 45, 1250-140 Lisboa", entity.Address, "Avenida da Liberdade 45, 1250-140 Lisboa"},
		{"wohnt in der Lindenstraße 8, 50674 Köln seit Mai", entity.Address, "Lindenstraße 8, 50674 Köln"},
		{"Adresse: Bergmannstrasse 102a\n10961 Berlin", entity.Address, "Bergmannstrasse 102a\n10961 Berlin"},
		{"Berliner Straße 44, 60311 Frankfurt am Main", entity.Address, "Berliner Straße 44, 60311 Frankfurt am Main"},
		{"Am Kupfergraben 6, 10117 Berlin", entity.Address, "Am Kupfergraben 6, 10117 Berlin"},
		{"An der Alster 12, 20099 Hamburg", entity.Address, "An der Alster 12, 20099 Hamburg"},
		{"woont op Herengracht 401, 1017 BP Amsterdam nu", entity.Address, "Herengracht 401, 1017 BP Amsterdam"},
		{"Oudegracht 77\n3511 AD Utrecht", entity.Address, "Oudegracht 77\n3511 AD Utrecht"},
		{"Kerkstraat 9", entity.Address, "Kerkstraat 9"},
		{"Via Garibaldi 30, 10122 Torino", entity.Address, "Via Garibaldi 30, 10122 Torino"},
		{"send it to Flat 4, 27 Glenorchy Road, Perth PH1 5XY please", entity.Address, "Flat 4, 27 Glenorchy Road, Perth PH1 5XY"},
		{"Flat 9, 3 Tollcross Lane", entity.Address, "Flat 9, 3 Tollcross Lane"},
		{"lives at 12 The Shambles, York YO1 7LX now", entity.Address, "12 The Shambles, York YO1 7LX"},
		{"88 Canal Street\nManchester\nM1 3HE", entity.Address, "88 Canal Street\nManchester\nM1 3HE"},
		{"SHIP TO:\n2207 HOLLOWAY RD STE 14\nRALEIGH NC 27603", entity.Address, "2207 HOLLOWAY RD STE 14\nRALEIGH NC 27603"},
		{"915 N ORCHARD AVE, BOISE ID 83706-1120", entity.Address, "915 N ORCHARD AVE, BOISE ID 83706-1120"},
	})
	runNegatives(t, entity.Address, []string{
		"sent via Slack 2 days ago",
		"Via Zoom 3 people joined",
		"Am Montag 12 Uhr",
		"Spring 5 is out",
		"Monitoring 24 hosts",
		"String 12 of the array",
		"Buffer 4096 bytes",
		"Arcade 5 tokens",
		"we rue the day 5 times",
		"3 place settings were laid",
		"version 2 route Tables changed",
		"ERROR 500 INTERNAL SERVER\nRETRY IN 30000",
		"Platz 3 im Rennen",
		"node 7 Weg 3",
	})
}
