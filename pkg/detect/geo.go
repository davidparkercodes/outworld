package detect

import (
	"sort"
	"strings"
)

var usStates = map[string]string{
	"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas", "CA": "California",
	"CO": "Colorado", "CT": "Connecticut", "DE": "Delaware", "FL": "Florida", "GA": "Georgia",
	"HI": "Hawaii", "ID": "Idaho", "IL": "Illinois", "IN": "Indiana", "IA": "Iowa",
	"KS": "Kansas", "KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
	"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi", "MO": "Missouri",
	"MT": "Montana", "NE": "Nebraska", "NV": "Nevada", "NH": "New Hampshire", "NJ": "New Jersey",
	"NM": "New Mexico", "NY": "New York", "NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio",
	"OK": "Oklahoma", "OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
	"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah", "VT": "Vermont",
	"VA": "Virginia", "WA": "Washington", "WV": "West Virginia", "WI": "Wisconsin", "WY": "Wyoming",
	"DC": "the District of Columbia",
}

func StateName(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if n, ok := usStates[strings.ToUpper(s)]; ok && len(s) == 2 {
		return n, true
	}
	for _, n := range usStates {
		if strings.EqualFold(n, s) {
			return n, true
		}
	}
	return "", false
}

func stateAlternation() string {
	var parts []string
	for abbr, name := range usStates {
		parts = append(parts, abbr)
		if !strings.HasPrefix(name, "the ") {
			parts = append(parts, strings.ReplaceAll(name, " ", `\s+`))
		}
	}
	sort.Slice(parts, func(i, j int) bool {
		if len(parts[i]) != len(parts[j]) {
			return len(parts[i]) > len(parts[j])
		}
		return parts[i] < parts[j]
	})
	return strings.Join(parts, "|")
}

func StateIn(s string) (string, bool) {
	i := strings.LastIndex(s, ",")
	if i < 0 {
		return "", false
	}
	tail := strings.TrimSpace(s[i+1:])
	f := strings.Fields(tail)
	for len(f) > 0 {
		if n, ok := StateName(strings.Join(f, " ")); ok {
			return n, true
		}
		f = f[:len(f)-1]
	}
	return "", false
}
