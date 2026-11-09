package dateshift

import "testing"

func TestShiftFormats(t *testing.T) {
	cases := []struct {
		in, want string
		days     int
		order    Order
	}{
		{"on 2024-03-15.", "on 2024-04-14.", 30, MDY},
		{"on 2024/03/15", "on 2024/04/14", 30, MDY},
		{"due 03/15/2024", "due 04/14/2024", 30, MDY},
		{"due 3/5/24", "due 4/4/24", 30, MDY},
		{"due 15/03/2024", "due 14/04/2024", 30, MDY},
		{"due 05/03/2024", "due 04/04/2024", 30, DMY},
		{"due 15.03.2024", "due 14.04.2024", 30, DMY},
		{"March 15, 2024", "April 14, 2024", 30, MDY},
		{"Mar 15 2024", "Apr 14 2024", 30, MDY},
		{"Sept 1, 2024", "Oct 1, 2024", 30, MDY},
		{"March 15th, 2024", "April 14th, 2024", 30, MDY},
		{"March 1st", "March 31st", 30, MDY},
		{"15 March 2024", "14 April 2024", 30, MDY},
		{"the 3rd of March, 2024", "the 2nd of April, 2024", 30, MDY},
		{"Friday, March 15, 2024", "Sunday, April 14, 2024", 30, MDY},
		{"Fri, Mar 15, 2024", "Sun, Apr 14, 2024", 30, MDY},
		{"Feb 29, 2024", "Mar 30, 2024", 30, MDY},
		{"2024-12-20T10:00:00Z", "2025-01-19T10:00:00Z", 30, MDY},
		{"March 15, 2024", "February 14, 2024", -30, MDY},
	}
	for _, c := range cases {
		got := Shift(c.in, c.order, c.days)
		if got != c.want {
			t.Errorf("Shift(%q,%d)=%q want %q", c.in, c.days, got, c.want)
		}
	}
}

func TestShiftRoundTrip(t *testing.T) {
	inputs := []string{
		"Seen 2023-01-31, again 02/28/2023 and on March 1, 2023; next 1 Jan 2024. Also Dec 31st and July 4.",
		"Tuesday, November 5, 2024 at 10am; follow-up 11/12/24.",
		"Born 1987-06-09, married 06/21/2010.",
	}
	for _, in := range inputs {
		for _, d := range []int{-400, -365, -91, -30, 1, 29, 45, 180, 366, 400} {
			out := Shift(in, MDY, d)
			if out == in {
				t.Errorf("no change for %q by %d", in, d)
			}
			if back := Shift(out, MDY, -d); back != in {
				t.Errorf("roundtrip %d: %q -> %q -> %q", d, in, out, back)
			}
		}
	}
}

func TestLeavesNonDatesAlone(t *testing.T) {
	for _, s := range []string{
		"see you next Tuesday", "two weeks ago", "yesterday at noon", "version 1.2.3", "May I help?",
		"in May we", "call 555-0123", "chapter 3/4", "room 12-30", "Mark 15 was read",
	} {
		if got := Shift(s, MDY, 77); got != s {
			t.Errorf("Shift(%q) changed to %q", s, got)
		}
	}
}

func TestFindInvalid(t *testing.T) {
	if ms := Find("02/30/2024 and 13/13/2024 and 2024-02-30", MDY); len(ms) != 0 {
		t.Fatalf("invalid dates detected: %+v", ms)
	}
}
