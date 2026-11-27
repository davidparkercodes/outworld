package detect

import (
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func TestBankDetails(t *testing.T) {
	runPositives(t, []classCase{
		{"IBAN de89 3704 0044 0532 0130 00 bitte", entity.Financial, "de89 3704 0044 0532 0130 00"},
		{"pay to GB29NWBK60161331926819 today", entity.Financial, "GB29NWBK60161331926819"},
		{"FR14 2004 1010 0505 0001 3M02 606 EUR", entity.Financial, "FR14 2004 1010 0505 0001 3M02 606"},
		{"NL91-ABNA-0417-1643-00", entity.Financial, "NL91-ABNA-0417-1643-00"},
		{"BIC DEUTDEFF500", entity.Financial, "DEUTDEFF500"},
		{"SWIFT/BIC: COBADEFFXXX", entity.Financial, "COBADEFFXXX"},
		{"swift code is NWBKGB2L", entity.Financial, "NWBKGB2L"},
		{"sort code 40-47-84", entity.Financial, "40-47-84"},
		{"Kontonummer: 1234567890", entity.Financial, "1234567890"},
		{"BLZ 37040044", entity.Financial, "37040044"},
		{"numéro de compte 00012345678", entity.Financial, "00012345678"},
		{"número de cuenta: 0200051332", entity.Financial, "0200051332"},
		{"rekeningnummer 417164300", entity.Financial, "417164300"},
		{"conto corrente n. 000123456789", entity.Financial, "000123456789"},
		{"CLABE 032180000118359719", entity.Financial, "032180000118359719"},
		{"BSB 062-000", entity.Financial, "062-000"},
		{"IFSC: HDFC0001234", entity.Financial, "HDFC0001234"},
		{"transit number 12345", entity.Financial, "12345"},
		{"routing 021000021", entity.Financial, "021000021"},
		{"card 4111 1111 1111 1111 04/27 123", entity.Financial, "4111 1111 1111 1111"},
		{"card 4111 1111 1111 1111 04/27 123", entity.Financial, "04/27"},
		{"card 4111 1111 1111 1111 04/27 123", entity.Financial, "123"},
		{"ref 77 4242-4242-4242-4242 ok", entity.Financial, "4242-4242-4242-4242"},
		{"amex 3782 822463 10005", entity.Financial, "3782 822463 10005"},
		{"tarjeta 5555-5555-5555-4444 caduca 12/28", entity.Financial, "12/28"},
		{"CVV: 842", entity.Financial, "842"},
		{"Visa card, expiry 09/29", entity.Financial, "09/29"},
		{"send to 0x52908400098527886E0F7030069857D2E4169EE7 please", entity.Financial, "0x52908400098527886E0F7030069857D2E4169EE7"},
		{"btc 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa", entity.Financial, "1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNa"},
		{"wallet bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq", entity.Financial, "bc1qar0srrr7xfkvy5l643lydnw9re59gtzzwf5mdq"},
	})
	runNegatives(t, entity.Financial, []string{
		"IBAN de89 3704 0044 0532 0130 01 bitte",
		"IBAN validation uses mod 97; example length DE=22, GB=22, FR=27",
		"BIC codes identify banks; SWIFT TRANSFERS take 1-3 days",
		"SWIFT messages MT103 and MT202 are supported",
		"Sort code format is 00-00-00",
		"Sort order: 10-20-30 ascending",
		"Routing numbers have 9 digits.",
		"Tener en cuenta que hay 1200000 usuarios activos.",
		"Le compte rendu de la réunion du 12 mars est joint.",
		"Das Konto wurde am 3. März eröffnet.",
		"Rekening houden met 30 dagen levertijd.",
		"In transit 12345 parcels, 482 delayed.",
		"IMEI 356938035643809",
		"Balance sheet row: 1024 2048 4096 8192 16384",
		"Subtotal 4500 1200 3300 7800 1250",
		"00000010  03 00 3e 00 01 00 00 00  40 10 00 00 00 00 00 00",
		"Tracking number: 9400 1000 0000 0000 0000 00",
		"order 1234567890123456 shipped",
		"Coupon SPRING2026 expires 04/30",
		"my passport expires 04/30",
		"Security code is required for card payments over 30 EUR.",
		"gas price 0x3b9aca00 wei",
		"topic 0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
		"commit 3f2a9c1d8e7b6a5f4c3d2e1f0a9b8c7d6e5f4a3b",
		"id 01HZX3K8R9T2V4W6Y8A0B1C2D3",
		"btc 1A1zP1eP5QGefi2DMPTfTL5SLmv7DivfNb",
		"sha 0x0000000000000000000000000000000000000000",
	})
}

func TestSeveralCardsInOneRun(t *testing.T) {
	count := func(text, want string) int {
		n := 0
		for _, s := range detectAll(t, text) {
			if s.Type == entity.Financial && s.Text == want {
				n++
			}
		}
		return n
	}
	for _, c := range []struct {
		text, want string
		n          int
	}{
		{"4111 1111 1111 1111 4111 1111 1111 1111", "4111 1111 1111 1111", 2},
		{"4111-1111-1111-1111 4111-1111-1111-1111 4111-1111-1111-1111", "4111-1111-1111-1111", 3},
		{"cards 4111 1111 1111 1111 5555 5555 5555 4444 ok", "5555 5555 5555 4444", 1},
		{"cards 4111 1111 1111 1111 5555 5555 5555 4444 ok", "4111 1111 1111 1111", 1},
		{"4111111111111111 4111111111111111", "4111111111111111", 2},
		{"12 4111 1111 1111 1111 9", "4111 1111 1111 1111", 1},
		{"4111 1111 1111 1111 04/27 123", "4111 1111 1111 1111", 1},
		{"3782 822463 10005 4111 1111 1111 1111", "3782 822463 10005", 1},
	} {
		if got := count(c.text, c.want); got != c.n {
			t.Errorf("%q: %d spans %q, want %d", c.text, got, c.want, c.n)
		}
	}
	runNegatives(t, entity.Financial, []string{
		"1111 1111 1111 1111 1111 1111 1111 1111",
		"row 1000 2000 3000 4000 5000 6000 7000 8000",
		"4111 1111 1111 1111x",
		"id=4111111111111111",
	})
}
