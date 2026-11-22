package detect

import (
	"context"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

func TestDateDetectorFormats(t *testing.T) {
	d := DateDetector{Order: dateshift.MDY}
	cases := []struct {
		text string
		want []string
	}{
		{"nos vemos el 3 de diciembre de 2026", []string{"3 de diciembre de 2026"}},
		{"Termin am 28. Oktober 2026", []string{"28. Oktober 2026"}},
		{"rendez-vous le 9 novembre", []string{"9 novembre"}},
		{"issued 12-Nov-2026, flight 02NOV26, back 14NOV", []string{"12-Nov-2026", "02NOV26", "14NOV"}},
		{"DTSTART:20261123T150000Z", []string{"20261123"}},
		{"stamp 20261127 ok", []string{"20261127"}},
		{"am 19.11.26 und 19.11.2026", []string{"19.11.26", "19.11.2026"}},
		{"december third, then the twenty-first of March", []string{"december third", "twenty-first of March"}},
		{"due 11/3, review by 10/14", []string{"11/3", "10/14"}},
		{"slots: 11/3, 11/10, 12/1", []string{"11/3", "11/10", "12/1"}},
		{"mix 3/4 cup, open 24/7, ratio 16/9", nil},
		{"v1.2.26 at 192.168.11.26, id 20261127, 0x20261127", nil},
		{"call 8045550123 or see src/11/3/a.go", nil},
	}
	for _, c := range cases {
		spans, err := d.Detect(context.Background(), c.text)
		if err != nil {
			t.Fatal(err)
		}
		if len(spans) != len(c.want) {
			t.Errorf("%q: got %d spans %+v, want %q", c.text, len(spans), spans, c.want)
			continue
		}
		for i, sp := range spans {
			if sp.Text != c.want[i] || sp.Type != entity.Date || c.text[sp.Start:sp.End] != sp.Text {
				t.Errorf("%q: span %d = %q want %q", c.text, i, sp.Text, c.want[i])
			}
		}
	}
}
