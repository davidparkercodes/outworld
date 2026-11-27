package vault

import (
	"bytes"
	"strings"
	"testing"
)

func TestDroppedFormsAreCountedOnLoad(t *testing.T) {
	be := NewMemoryBackend(bytes.Repeat([]byte{3}, 32))
	if err := be.Update(func([]Record) ([]Record, error) {
		recs := []Record{formatRecord, {Kind: "phone", Real: "804 555 1234", Surrogate: "555 867 5309"}}
		for i := 1; i <= MaxPhoneFormsInContext+3; i++ {
			zw := strings.Repeat("\u200b", i)
			recs = append(recs, Record{Kind: phoneFormKind, Real: "804" + zw + "555 1234", Surrogate: "555" + zw + "867 5309"})
		}
		return recs, nil
	}); err != nil {
		t.Fatal(err)
	}
	v, err := Open(be)
	if err != nil {
		t.Fatal(err)
	}
	if got := v.DroppedForms(); got != 3 {
		t.Fatalf("dropped %d forms on load, want 3", got)
	}
}
