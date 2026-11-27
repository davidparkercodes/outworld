package outworld

import (
	"context"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

var detectionPassRestoreCases = []string{
	"phone=(804)555-1234",
	"tel=(804)555〜1234",
	"PHONE=(804)555・1234",
	"phone_number=(804)555-1234",
	"?phone=(804)555-1234&x=1",
	"contact=[(804)555／1234]",
	"see [(804)555-1234] now",
	"call804-555-1234",
	"call804〜555〜1234",
	"Phone804.555.1234",
	"call​804-555-1234",
	"_Phone: 804-555-1234_",
	"**804-555-1234**",
	"`call 804・555・1234`",
	"phone=804~555~1234",
	"phone_number: 804〜555〜1234",
	"Call 804-555-1234です",
	"電話番号は804-555-1234です",
	"전화 804-555-1234로",
	"+1 804~555~1234",
	"+1〜804〜555〜1234",
	"Call 804​555​1234 now",
}

func TestDetectionPassFormsMaskAndRestore(t *testing.T) {
	for _, text := range detectionPassRestoreCases {
		e := newEngine(t, nil)
		s := e.NewSession("dp")
		m, err := s.Mask(context.Background(), text)
		if err != nil || s.Err() != nil {
			t.Fatalf("%q: %v %v", text, err, s.Err())
		}
		if strings.Contains(m, "1234") {
			t.Errorf("%q: masked %q keeps the number", text, m)
			continue
		}
		if back := s.Rehydrate(m); back != text {
			t.Errorf("%q: masked %q restores %q", text, m, back)
		}
	}
}

func TestEngineDetectSeesZeroWidthGluedCues(t *testing.T) {
	e := newEngine(t, nil)
	for _, c := range []struct {
		text string
		typ  entity.Type
	}{
		{"SSN​123-45-6789", entity.GovID},
		{"call​804-555-1234", entity.Phone},
		{"SSN123-45-6789", entity.GovID},
		{"call804-555-1234", entity.Phone},
	} {
		spans, err := e.Detect(context.Background(), c.text)
		if err != nil {
			t.Fatal(err)
		}
		found := false
		for _, sp := range spans {
			if sp.Type == c.typ {
				found = true
			}
		}
		if !found {
			t.Errorf("%q: Engine.Detect found no %s", c.text, c.typ)
		}
	}
}
