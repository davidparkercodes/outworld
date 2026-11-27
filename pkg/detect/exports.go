package detect

import (
	"regexp"
	"strings"
)

func CardNumber(digits string) bool { return validCard(digits) }

var separatedPhoneRE = regexp.MustCompile(`^(?:\+?1[-. ])?(?:\([2-9][0-9]{2}\) ?|[2-9][0-9]{2}[-. ])[2-9][0-9]{2}[-. ][0-9]{4}$`)

func SeparatedPhone(s string) bool {
	return separatedPhoneRE.MatchString(s) && nanpRun(digitsOnly(s))
}

func PhoneLabelWord(w string) bool { return phoneLabelWords[strings.ToLower(w)] }
