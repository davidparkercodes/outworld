package dateshift

import (
	"strings"
	"testing"
)

func TestVersionLabelsAreStrict(t *testing.T) {
	kept := []string{
		"curl -H 'anthropic-version: 2023-06-01' https://api.example.com",
		"POST /deploy?api-version=2024-02-01",
		`{"apiVersion": "2024-05-01"}`,
		"OpenAI-Version: 2024-01-15",
		"x-api-version=2023-10-10",
		"app_version: 2024-03-01",
		"version: 2024-03-01",
		"ver=2024-03-01",
	}
	for _, in := range kept {
		if out := Shift(in, MDY, 40); out != in {
			t.Errorf("%q -> %q", in, out)
		}
		if out := ShiftLoose(in, MDY, -40); out != in {
			t.Errorf("loose %q -> %q", in, out)
		}
	}
	shifted := []string{
		"GET https://api.example.com/v1/2024-11-05/items",
		"see https://blog.example.com/2024/03/15/post",
		"https://example.com/patients/elena-petrova/1985-03-14",
		"https://example.com/form?dob=1985-03-14",
		"dobversion: 1985-03-14",
		"birthdate_version=1985-03-14",
		"DOB version: 1985-03-14",
		"Patient born on version: 1985-03-14",
		"Her birthday version: 1985-03-14",
		"subversion: 2024-03-01",
		"Meeting on 2024-11-05 at noon.",
	}
	for _, in := range shifted {
		if out := Shift(in, MDY, 40); out == in {
			t.Errorf("not shifted: %q", in)
		}
	}
}

func TestVersionScanOnlyTouchesLabelWindows(t *testing.T) {
	plain := strings.Repeat("The server went over every log line on 2024-03-05 and nothing was over budget. ", 4096)
	if w := versionWindows(plain); len(w) != 0 {
		t.Fatalf("label-free text produced %d windows", len(w))
	}
	if got := FindVersioned(plain, MDY, false); len(got) != 0 {
		t.Fatalf("label-free text produced %d versioned dates", len(got))
	}
	labelled := plain + "anthropic-version: 2023-06-01\n" + plain + "api-version=2024-02-01"
	covered := 0
	for _, w := range versionWindows(labelled) {
		covered += w[1] - w[0]
	}
	if covered == 0 || covered > 2*(versionWindow+versionValueSpan+len("anthropic-version: ")) {
		t.Fatalf("version scan covers %d bytes", covered)
	}
	if got := FindVersioned(labelled, MDY, false); len(got) != 2 {
		t.Fatalf("found %d versioned dates", len(got))
	}
}
