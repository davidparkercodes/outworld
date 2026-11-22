package dateshift

import (
	"strings"
	"testing"
)

type shiftCase struct {
	in, want string
}

func checkShift(t *testing.T, order Order, days int, cases []shiftCase) {
	t.Helper()
	for _, c := range cases {
		if got := Shift(c.in, order, days); got != c.want {
			t.Errorf("Shift(%q,%d)=%q want %q", c.in, days, got, c.want)
		}
	}
}

var roundTripOffsets = []int{-400, -365, -91, -30, -1, 1, 29, 40, 45, 180, 366, 400}

func checkRoundTrip(t *testing.T, order Order, inputs []string) {
	t.Helper()
	for _, in := range inputs {
		for _, d := range roundTripOffsets {
			out := Shift(in, order, d)
			if out == in && d%365 != 0 {
				t.Errorf("no change for %q by %d", in, d)
				continue
			}
			if back := Shift(out, order, -d); back != in {
				t.Errorf("roundtrip %d: %q -> %q -> %q", d, in, out, back)
			}
		}
	}
}

func checkUnchanged(t *testing.T, order Order, inputs []string) {
	t.Helper()
	for _, in := range inputs {
		for _, d := range []int{40, 77, -123} {
			if got := Shift(in, order, d); got != in {
				t.Errorf("Shift(%q,%d) changed to %q", in, d, got)
			}
		}
		if ms := Find(in, order); len(ms) != 0 {
			t.Errorf("Find(%q) = %q", in, ms[0].Text)
		}
	}
}

func TestOtherLanguageMonths(t *testing.T) {
	checkShift(t, MDY, 40, []shiftCase{
		{"el 3 de diciembre de 2026", "el 12 de enero de 2027"},
		{"vence el 15 de noviembre", "vence el 25 de diciembre"},
		{"jueves, 3 de diciembre de 2026", "martes, 12 de enero de 2027"},
		{"MIÉRCOLES 4 DE MARZO DE 2026", "LUNES 13 DE ABRIL DE 2026"},
		{"am 28. Oktober 2026 um 10 Uhr", "am 7. Dezember 2026 um 10 Uhr"},
		{"Mittwoch, 28. Oktober 2026", "Montag, 7. Dezember 2026"},
		{"bis zum 3. März", "bis zum 12. April"},
		{"le 9 novembre", "le 19 décembre"},
		{"le 12 août 2026", "le 21 septembre 2026"},
		{"em 3 de dezembro de 2026", "em 12 de janeiro de 2027"},
		{"il 5 ottobre 2026", "il 14 novembre 2026"},
		{"op 3 december 2026", "op 12 januari 2027"},
		{"op 5 mei", "op 14 juni"},
		{"el 3 de dic. de 2026", "el 12 de ene. de 2027"},
	})
	checkRoundTrip(t, MDY, []string{
		"el 3 de diciembre de 2026 y el 15 de noviembre",
		"jueves, 3 de diciembre de 2026",
		"am 28. Oktober 2026, dann 3. März und 1. Mai 2027",
		"Mittwoch, 28. Oktober 2026",
		"le 9 novembre, puis le 12 août 2026 et le 2 février 2027",
		"em 3 de dezembro de 2026 e 7 de março",
		"il 7 gennaio 2026 e il 21 dicembre",
		"op 3 december 2026 en 5 mei",
	})
}

func TestDayMonYearAndCompactNames(t *testing.T) {
	checkShift(t, MDY, 40, []shiftCase{
		{"12-Nov-2026", "22-Dec-2026"},
		{"12-nov-26", "22-dec-26"},
		{"12/Nov/2026", "22/Dec/2026"},
		{"05-NOV-2026", "15-DEC-2026"},
		{"row 14-Nov ok", "row 24-Dec ok"},
		{"DEP 02NOV26 LHR", "DEP 12DEC26 LHR"},
		{"RET 14NOV", "RET 24DEC"},
		{"02Nov2026", "12Dec2026"},
		{"25DEC", "03FEB"},
	})
	checkRoundTrip(t, MDY, []string{"12-Nov-2026", "3-Mar-26", "12/Nov/2026", "14-Nov", "02NOV26", "14NOV", "31DEC99", "02Nov2026", "9-September-2026"})
}

func TestCompactISOAndICalendar(t *testing.T) {
	checkShift(t, MDY, 40, []shiftCase{
		{"20261127", "20270106"},
		{"DTSTART:20261123T150000Z", "DTSTART:20270102T150000Z"},
		{"DTEND;VALUE=DATE:20261124", "DTEND;VALUE=DATE:20270103"},
		{"export_20261127.csv", "export_20270106.csv"},
		{"20261127,20261128", "20270106,20270107"},
		{"at 20261123T1500 local", "at 20270102T1500 local"},
	})
	checkRoundTrip(t, MDY, []string{"20261127", "DTSTART:20261123T150000Z", "19991231 and 20000229", "export_20261127.csv"})
	checkUnchanged(t, MDY, []string{
		"12345678", "20261340", "20260230", "99991231", "18991231", "21000101",
		"0x20261127", "deadbeef20261127", "20261127abc", "120261127", "202611279",
		"id 20261127", "order #20261127", "ISBN 9780306406157", "isbn 20261127",
		"+12026112712", "call 2026112700", "1.20261127", "20261127.5", "555-20261127", "20261127-01",
		"sha 20261127cafe", "20261127Tango",
	})
}

func TestDottedNumericDates(t *testing.T) {
	checkShift(t, MDY, 40, []shiftCase{
		{"am 19.11.26 um 9", "am 29.12.26 um 9"},
		{"19.11.2026", "29.12.2026"},
		{"05.03.2024", "14.04.2024"},
		{"due 5.3.26", "due 14.4.26"},
		{"since 1.12.26", "since 10.1.27"},
	})
	checkRoundTrip(t, MDY, []string{"19.11.26", "01.02.26 bis 28.02.26", "19.11.2026", "due 5.3.26", "05.03.2024"})
	checkRoundTrip(t, DMY, []string{"19.11.26", "05.03.2024"})
	checkUnchanged(t, MDY, []string{
		"version 10.11.26", "v19.11.26", "release 12.10.26", "build 10.11.26", "1.2.26", "3.11.26 is out",
		"192.168.11.26", "10.11.12.13", "host 19.11.26.4", "32.11.26", "19.13.26", "1.2.3", "v2.10.11",
	})
}

func TestSpokenEnglish(t *testing.T) {
	checkShift(t, MDY, 40, []shiftCase{
		{"december third", "january twelfth"},
		{"the twenty-first of March", "the thirtieth of April"},
		{"The Twenty-First of March, 2026", "The Thirtieth of April, 2026"},
		{"on may first", "on june tenth"},
		{"June the second, 2026", "July the twelfth, 2026"},
		{"the third day of april", "the thirteenth day of may"},
		{"see you march 5", "see you april 14"},
		{"monday, march 6th", "saturday, april 15th"},
		{"5th of november", "15th of december"},
		{"on 3 october 2026", "on 12 november 2026"},
		{"oct 3, 2026", "nov 12, 2026"},
	})
	checkRoundTrip(t, MDY, []string{
		"december third", "the twenty-first of March", "the first of january, 2026", "see you march 5",
		"on may first", "due may 5 for sure", "September Thirtieth", "monday, march 6th", "5th of november",
	})
	checkUnchanged(t, MDY, []string{
		"you may first need to log in", "they march 5 miles a day", "it may 3 times fail", "2 may be enough",
		"May second-guess it", "wait a second of silence", "the first of many", "march first, ask later",
		"we may 10 of them", "first march then rest",
	})
}

func TestYearlessNumericRule(t *testing.T) {
	accepted := []shiftCase{
		{"due 11/3", "due 12/13"},
		{"by 10/14 please", "by 11/23 please"},
		{"the week of 12/1", "the week of 1/10"},
		{"scheduled for 03/04", "scheduled for 04/13"},
		{"moved until 11/3.", "moved until 12/13."},
		{"as of 11/3", "as of 12/13"},
		{"Mon 11/3, Tue 11/4", "Mon 12/13, Tue 12/14"},
		{"appt 3/4 at 2pm", "appt 4/13 at 2pm"},
		{"11/3 @ 9", "12/13 @ 9"},
		{"11/3 (Tue) standup", "12/13 (Tue) standup"},
		{"out 11/3-11/7", "out 12/13-12/17"},
		{"from 11/3 to 11/7", "from 12/13 to 12/17"},
		{"| 11/3 | dentist |\n| 11/10 | vet |\n| 11/17 | bank |", "| 12/13 | dentist |\n| 12/20 | vet |\n| 12/27 | bank |"},
		{"options: 11/3, 11/10, 11/17", "options: 12/13, 12/20, 12/27"},
		{"filed 2026-11-01, hearing 11/3", "filed 2026-12-11, hearing 12/13"},
		{"hearing 11/3, filed November 1, 2026", "hearing 12/13, filed December 11, 2026"},
	}
	checkShift(t, MDY, 40, accepted)
	var inputs []string
	for _, c := range accepted {
		inputs = append(inputs, c.in)
	}
	checkRoundTrip(t, MDY, inputs)
	checkShift(t, DMY, 40, []shiftCase{{"due 3/11", "due 13/12"}, {"until 14/10", "until 23/11"}})
	checkRoundTrip(t, DMY, []string{"due 3/11", "until 14/10"})
	checkUnchanged(t, MDY, []string{
		"3/4 cup", "add 1/2 tsp", "on 3/4 of the days", "open 24/7", "on call 24/7", "ratio 3/2", "a 1/2 split",
		"reduce by 1/3", "cut from 1/2", "scored 3/2", "score was 3/2", "won 4/1", "page 11/12", "step 3/7",
		"version 1/2", "chapter 3/4", "1/2", "11/3", "pick 5/6", "5/8\" pipe", "10/12 passed", "1/2-inch",
		"src/11/3/main.go", "/var/11/3", "C:\\logs\\11/3", "11/3/x", "a/11/3", "v11/3", "$11/3", "11/3%",
		"due 13/14", "due 2/30", "due 0/5", "due 14/10", "x=11/3", "3.11/3", "11/3.5",
		"two of 1/4 and 3/8", "192.168.1.1/24", "10.0.0.0/8", "1.2.3", "v2.10.11", "555-0123", "0xff/12",
	})
	for _, s := range []string{"11/3", "a 1/2 split"} {
		ms := FindLoose(s, MDY)
		if s == "11/3" && (len(ms) != 1 || !ms[0].Weak) {
			t.Errorf("FindLoose(%q) = %+v, want one weak match", s, ms)
		}
		if s != "11/3" && len(ms) != 0 {
			t.Errorf("FindLoose(%q) matched a guarded token", s)
		}
	}
	if got := ShiftLoose("11/3", MDY, 40); got != "12/13" {
		t.Errorf("ShiftLoose bare token = %q", got)
	}
	for _, m := range Find("due 11/3", MDY) {
		if m.Weak {
			t.Errorf("supported match marked weak")
		}
	}
}

func TestGeneralNegatives(t *testing.T) {
	checkUnchanged(t, MDY, []string{
		"version 1.2.3", "semver 2.10.11-rc.1", "v1.12.0", "go1.22.3", "10.0.0.5", "fe80::1", "0xDEADBEEF",
		"call 804-555-0123", "(804) 555-0123", "5551234567", "978-0-306-40615-7", "ISBN 0306406152",
		"/usr/lib/12/3.so", "a/b/c", "3 out of 4", "set 5 items", "5 set", "2 ago", "10 dec places", "sat 5 hours",
		"mag 7", "mar 3 times a week", "10 dec places", "3 jan people", "room 12-30", "12-30", "14NOVA", "A14NOV", "14nov", "2x4", "1/1/1/1",
	})
}

func TestVocab(t *testing.T) {
	for _, w := range []string{"diciembre", "Oktober", "août", "twenty", "first", "de", "NOV"} {
		if !Vocab(w, false) {
			t.Errorf("Vocab(%q) false", w)
		}
	}
	if !Vocab("dicie", true) || Vocab("weather", true) || Vocab("dicie", false) {
		t.Errorf("Vocab partial matching wrong")
	}
	if strings.Contains(Shift("le 9 novembre", MDY, 40), "December") {
		t.Errorf("language not preserved")
	}
}
