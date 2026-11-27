package detect

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/entity"
)

const handleChars = `[A-Za-z][A-Za-z0-9._\-]{1,30}`

var (
	todoHandleRE = regexp.MustCompile(`\b(?:TODO|FIXME|XXX|HACK|NOTE|BUG|OPTIMIZE|REVIEW)[ \t]?\(@?(` + handleChars + `)\)`)

	homeHandleRE = regexp.MustCompile(`(?:/home/|/Users/|/export/home/|\\+Users\\+|\\+home\\+)(` + handleChars + `)`)

	branchHandleRE = regexp.MustCompile(`(?i:\b(?:branch|branches|checkout|switch|merge|merged|rebase|rebased|worktree)[ \t]+(?:(?:-{1,2}[a-z][a-z\-]*|the|my|his|her|their|our|a|into|onto|from|in|to|of|called|named)[ \t]+){0,3})["'\x60]?([a-z][a-z0-9._\-]{1,30})/[A-Za-z0-9]`)

	remoteHandleRE = regexp.MustCompile(`\b(?:origin|upstream|remotes/[A-Za-z0-9_\-]+|refs/heads|refs/remotes/[A-Za-z0-9_\-]+)/([a-z][a-z0-9._\-]{1,30})/[A-Za-z0-9]`)

	gitHandleRE = regexp.MustCompile(`(?:\bgit[ \t]+(?:push|pull|fetch)(?:[ \t]+-{1,2}[A-Za-z][A-Za-z\-]*)*[ \t]+[A-Za-z0-9_.\-]+|\bgit[ \t]+(?:branch|switch|rebase|checkout)(?:[ \t]+-{1,2}[A-Za-z][A-Za-z\-]*)*|\bgh[ \t]+pr[ \t]+checkout|(?i:\b(?:PR|pull[ \t]+request|MR|merge[ \t]+request)(?:[ \t]+[#!]?[0-9]{1,7})?[ \t]+from|\brebased[ \t]+onto|\bbased[ \t]+on|\boff[ \t]+of))[ \t]+["'\x60]?([a-z][a-z0-9._\-]{1,30})/[A-Za-z0-9]`)

	gitPathHandleRE = regexp.MustCompile(`\bgit[ \t]+(?:log|diff)(?:[ \t]+-{1,2}[A-Za-z][A-Za-z\-=0-9]*)*[ \t]+(?:[A-Za-z0-9_\-]+\.{2,3})?(?:origin/|upstream/)?([a-z][a-z0-9._\-]{1,30})/[A-Za-z0-9]`)

	forkHandleRE = regexp.MustCompile(`(?:^|[\s"'\x60(\[,])([a-z][a-z0-9\-]{2,30}):[a-z0-9][A-Za-z0-9._\-]*(?:/[A-Za-z0-9._\-]+)*`)

	mentionHandleRE = regexp.MustCompile(`(?:^|[\s(\[{,;:>"'\x60*])@([a-z][a-z0-9._\-]{1,30})`)

	proseHandleRE = regexp.MustCompile(`(?i:\b(ask|ping|cc|assigned to|assignee|reviewer|owner|thanks|per|from|by|with|poke|loop in|tag|blame))(?::[ \t]?|[ \t])@?([a-z][a-z0-9]{3,23})`)

	trailerRE = regexp.MustCompile(`(?im)^[ \t>*\-]*(?:reviewed|co-authored|signed-off|acked|tested|reported|suggested|helped|authored|approved|committed|requested|merged|co-developed)-by:[ \t]*([^\n<]*)`)
)

var handleStop = wordSet(`root admin administrator sysadmin superuser ubuntu debian centos fedora alpine kali arch ec2-user azureuser opc vagrant runner runneradmin deploy deployer deployment www www-data wwwroot http httpd nginx apache git gitlab gitlab-runner github jenkins circleci travis buildkite teamcity bamboo drone node npm postgres postgresql mysql mongodb redis oracle hadoop spark elastic elasticsearch kibana grafana prometheus ansible terraform docker podman container containers user users username usernames yourname yourusername your-username name myname myuser me you shared public default guest guests everyone all here channel team staff nobody daemon system service services svc app apps application web site sites ftp sftp mail backup backups build builder builds ci cd test tests tester testing testuser demo example sample sandbox tmp temp cache data db dev devs developer develop development prod production staging stage stable release releases main master trunk feature features feat fix fixes bugfix hotfix chore docs doc refactor perf performance style revert wip topic task story epic spike experiment experimental exp maintenance maint support next canary nightly beta alpha preview pull merge heads tags origin upstream local home work workspace workspaces project projects code src lib bin opt var etc usr pi jovyan vscode coder codespace gitpod linuxbrew steam minecraft bot bots robot dependabot renovate renovate-bot greenkeeper snyk snyk-bot codecov mergify copilot codex claude cursor devin sweep actions github-actions security api ui ux i18n l10n a11y auth config types lint cleanup later soon now urgent important optional question sync migration migrate deps dependencies param params returns return throws type typedef see since deprecated author version override media import charset keyframes supports property staticmethod classmethod dataclass fixture component injectable input output interface implementation end synthesize available escaping published state binding environment setter getter deleter font-face namespace layer tailwind apply use forward mixin include extend each function else if for while debug warn error content page scope counter-style`)

const (
	handleNone = iota
	handleWeak
	handleStrong
)

func handleWordish(w string) bool {
	return handleStop[w] || techTerms[w] || stopCap[w] || notSurname[w] || notSurnameExtra[w] || dateAbbrev[w] || IsMonthOrWeekday(w)
}

func lettersOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 'a' || s[i] > 'z' {
			return false
		}
	}
	return s != ""
}

func handleBody(h string) (body, digits string) {
	h = textutil.Fold(h)
	i := len(h)
	for i > 0 && h[i-1] >= '0' && h[i-1] <= '9' {
		i--
	}
	return h[:i], h[i:]
}

func splitHandleBody(body string) (given, family string, strength int) {
	n := len(body)
	if n < 4 || n > 24 || !lettersOnly(body) || handleWordish(body) {
		return "", "", handleNone
	}
	if _, ok := namedata.LookupGiven(body); ok {
		return "", "", handleNone
	}
	if _, ok := namedata.LookupSurname(body); ok {
		return "", "", handleNone
	}
	var weakGiven, weakFamily string
	for i := n - 3; i >= 3; i-- {
		g, gok := namedata.LookupGiven(body[:i])
		_, sok := namedata.LookupSurname(body[i:])
		if !gok || !sok || handleWordish(body[i:]) || techTerms[body[:i]] {
			continue
		}
		if !g.Weak && n-i >= 4 {
			return body[:i], body[i:], handleStrong
		}
		if weakGiven == "" {
			weakGiven, weakFamily = body[:i], body[i:]
		}
	}
	if _, ok := namedata.LookupSurname(body[1:]); ok && !handleWordish(body[1:]) {
		if n >= 5 {
			return body[:1], body[1:], handleStrong
		}
		return body[:1], body[1:], handleWeak
	}
	if weakGiven != "" {
		return weakGiven, weakFamily, handleWeak
	}
	return "", "", handleNone
}

func SplitHandle(h string) (given, family, digits string, ok bool) {
	body, digits := handleBody(h)
	given, family, strength := splitHandleBody(body)
	return given, family, digits, strength != handleNone
}

func handleSignal(h string) int {
	lh := textutil.Fold(h)
	if handleStop[lh] || strings.HasSuffix(lh, "bot") || strings.Contains(lh, "-bot") {
		return handleNone
	}
	parts := strings.FieldsFunc(lh, func(r rune) bool { return r == '.' || r == '_' || r == '-' })
	if len(parts) == 0 || len(parts) > 3 {
		return handleNone
	}
	if len(parts) >= 2 {
		parts[len(parts)-1], _ = handleBody(parts[len(parts)-1])
		for _, p := range parts {
			if !lettersOnly(p) || handleWordish(p) {
				return handleNone
			}
		}
		first, last := parts[0], parts[len(parts)-1]
		g, gok := namedata.LookupGiven(first)
		gok = gok && len(first) >= 2
		_, sok := namedata.LookupSurname(last)
		sok = sok && len(last) >= 3
		gStrong := gok && !g.Weak && len(first) >= 3 && !namedata.IsHandleWord(first)
		switch {
		case gStrong && (sok || len(last) == 1):
			return handleStrong
		case len(first) == 1 && sok && len(last) >= 4 && !namedata.IsHandleWord(last):
			return handleStrong
		case gok && sok && !textutil.IsCommonWord(last):
			return handleStrong
		case gok || sok:
			return handleWeak
		}
		return handleNone
	}
	body, _ := handleBody(lh)
	if len(body) < 3 || !lettersOnly(body) || handleWordish(body) {
		return handleNone
	}
	if g, ok := namedata.LookupGiven(body); ok {
		if !g.Weak && len(body) >= 4 && !textutil.IsCommonWord(body) {
			return handleStrong
		}
		return handleWeak
	}
	if weak, ok := namedata.LookupSurname(body); ok {
		if !weak && len(body) >= 5 && !textutil.IsCommonWord(body) && !placeLike[body] {
			return handleStrong
		}
		if !weak {
			return handleWeak
		}
		return handleNone
	}
	if namedata.IsHandleWord(body) {
		return handleNone
	}
	_, _, strength := splitHandleBody(body)
	return strength
}

func usernameLike(h string) bool {
	lh := textutil.Fold(h)
	if handleStop[lh] || strings.HasSuffix(lh, "bot") || strings.Contains(lh, "-bot") {
		return false
	}
	if handleSignal(h) != handleNone {
		return true
	}
	body, _ := handleBody(lh)
	if body == "" || handleStop[body] || handleWordish(body) || textutil.IsCommonWord(body) || namedata.IsHandleWord(body) {
		return false
	}
	for _, p := range strings.FieldsFunc(body, func(r rune) bool { return r == '.' || r == '_' || r == '-' }) {
		if handleWordish(p) || utf8.RuneCountInString(p) < 2 {
			return false
		}
	}
	return utf8.RuneCountInString(body) >= 3
}

var weakHandleCues = wordSet(`per from by with`)

var todoTopics = wordSet(`august january than carlo`)

func todoName(h string) bool {
	lh := textutil.Fold(h)
	if !lettersOnly(lh) || len(lh) < 3 || handleStop[lh] || techTerms[lh] || dateAbbrev[lh] || todoTopics[lh] {
		return false
	}
	_, ok := namedata.LookupGiven(lh)
	return ok
}

func SplitNameInitial(h string) (given, initial, digits string, ok bool) {
	body, digits := handleBody(h)
	if !lettersOnly(body) || len(body) < 5 || handleWordish(body) || textutil.IsCommonWord(body) || namedata.IsHandleWord(body) || KnownName(body) {
		return "", "", "", false
	}
	if _, _, strength := splitHandleBody(body); strength != handleNone {
		return "", "", "", false
	}
	first := body[:len(body)-1]
	g, known := namedata.LookupGiven(first)
	if !known || g.Weak || len(first) < 4 || handleWordish(first) || textutil.IsCommonWord(first) {
		return "", "", "", false
	}
	return first, body[len(body)-1:], digits, true
}

func proseHandle(h string, weakCue bool) bool {
	if handleStop[h] || strings.HasSuffix(h, "bot") || textutil.IsCommonWord(h) || namedata.IsHandleWord(h) || KnownName(h) {
		return false
	}
	body, _ := handleBody(h)
	if !lettersOnly(body) || handleWordish(body) || textutil.IsCommonWord(body) || namedata.IsHandleWord(body) || KnownName(body) {
		return false
	}
	given, family, strength := splitHandleBody(body)
	if strength == handleStrong || (strength == handleWeak && !weakCue && len(given) == 1 && len(family) >= 3) {
		return true
	}
	_, _, _, ok := SplitNameInitial(h)
	return ok
}

func proseHandleSpans(text string, emit func(s, e int)) {
	for _, m := range proseHandleRE.FindAllStringSubmatchIndex(text, -1) {
		s, e := m[4], m[5]
		if m[2] > 0 {
			r, _ := utf8.DecodeLastRuneInString(text[:m[2]])
			if strings.ContainsRune("@./_\\#$%&=+~-", r) {
				continue
			}
		}
		if e < len(text) {
			r, _ := utf8.DecodeRuneInString(text[e:])
			if textutil.IsWordRune(r) || strings.ContainsRune("@(_\\/=+~-[{<'’", r) {
				continue
			}
			if r == '.' && e+1 < len(text) && textutil.IsWordRune(rune(text[e+1])) {
				continue
			}
		}
		h := text[s:e]
		if !textutil.IsAllLower(h) {
			continue
		}
		if proseHandle(h, weakHandleCues[strings.ToLower(text[m[2]:m[3]])]) {
			emit(s, e)
		}
	}
}

func personalHandle(h string) bool {
	if handleSignal(h) != handleStrong {
		return false
	}
	if strings.ContainsAny(h, "._-") {
		return true
	}
	body, _ := handleBody(h)
	_, _, strength := splitHandleBody(body)
	return strength == handleStrong
}

func forkRef(text string, m []int) bool {
	ref := text[m[3]+1 : m[1]]
	if m[1] < len(text) && strings.ContainsRune(":/@=\\", rune(text[m[1]])) {
		return false
	}
	if strings.HasPrefix(ref, "/") || !textutil.HasLetter(ref) {
		return false
	}
	return strings.ContainsAny(ref, "-_/") || handleStop[ref]
}

func trimHandle(text string, s, e int) int {
	for e > s && strings.ContainsRune("._-", rune(text[e-1])) {
		e--
	}
	return e
}

func handleSpans(text string) []entity.Span {
	var out []entity.Span
	emit := func(s, e int) {
		e = trimHandle(text, s, e)
		if e-s < 2 {
			return
		}
		out = append(out, entity.Span{Start: s, End: e, Type: entity.Person, Label: "person", Text: text[s:e], Source: "names:handle", Priority: 27, Score: 1})
	}
	if strings.Contains(text, "(") {
		for _, m := range todoHandleRE.FindAllStringSubmatchIndex(text, -1) {
			if h := text[m[2]:m[3]]; handleSignal(h) != handleNone || todoName(h) {
				emit(m[2], m[3])
			}
		}
	}
	if strings.Contains(text, "sers") || strings.Contains(text, "home") {
		for _, m := range homeHandleRE.FindAllStringSubmatchIndex(text, -1) {
			e := trimHandle(text, m[2], m[3])
			if e < len(text) && !strings.ContainsRune("/\\ \t\n\r\"'`:;,)]}>|", rune(text[e])) && !(text[e] == '.' && e+1 == len(text)) {
				continue
			}
			if usernameLike(text[m[2]:e]) {
				emit(m[2], e)
			}
		}
	}
	if strings.Contains(text, "/") {
		for _, re := range []*regexp.Regexp{branchHandleRE, remoteHandleRE, gitHandleRE} {
			for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
				if handleSignal(text[m[2]:m[3]]) == handleStrong {
					emit(m[2], m[3])
				}
			}
		}
		for _, m := range gitPathHandleRE.FindAllStringSubmatchIndex(text, -1) {
			if personalHandle(text[m[2]:m[3]]) {
				emit(m[2], m[3])
			}
		}
	}
	if strings.Contains(text, ":") {
		for _, m := range forkHandleRE.FindAllStringSubmatchIndex(text, -1) {
			if forkRef(text, m) && personalHandle(text[m[2]:m[3]]) {
				emit(m[2], m[3])
			}
		}
	}
	if strings.Contains(text, "@") {
		for _, m := range mentionHandleRE.FindAllStringSubmatchIndex(text, -1) {
			e := trimHandle(text, m[2], m[3])
			rest := text[e:]
			if rest != "" && strings.ContainsRune("/(@[=\\", rune(rest[0])) {
				continue
			}
			if r, _ := utf8.DecodeRuneInString(text[m[3]:]); m[3] < len(text) && (textutil.IsWordRune(r) || r == '_') {
				continue
			}
			if strings.HasPrefix(rest, ":") && len(rest) > 1 && rest[1] != ' ' && rest[1] != '\n' {
				continue
			}
			if t := strings.TrimLeft(rest, " \t"); strings.HasPrefix(t, "=") || strings.HasPrefix(t, "||=") {
				continue
			}
			if handleSignal(text[m[2]:e]) == handleStrong {
				emit(m[2], e)
			}
		}
	}
	proseHandleSpans(text, emit)
	if strings.Contains(text, "-by:") || strings.Contains(text, "-By:") || strings.Contains(text, "-BY:") {
		for _, m := range trailerRE.FindAllStringSubmatchIndex(text, -1) {
			s, e := m[2], m[3]
			for e > s && (text[e-1] == ' ' || text[e-1] == '\t' || text[e-1] == '\r') {
				e--
			}
			if s < e && text[s] == '@' {
				s++
			}
			if s >= e || strings.Contains(text[s:e], "[bot]") {
				continue
			}
			words := strings.Fields(text[s:e])
			if len(words) > 4 {
				continue
			}
			ok := true
			for _, w := range words {
				lw := strings.ToLower(strings.Trim(w, ".,"))
				if handleStop[lw] || techTerms[lw] || stopCap[lw] || strings.HasSuffix(lw, "bot") || !textutil.HasLetter(w) || strings.ContainsAny(w, "@/:()[]{}=\"") {
					ok = false
				}
			}
			if !ok {
				continue
			}
			if len(words) == 1 && textutil.IsAllLower(words[0]) && !usernameLike(words[0]) {
				continue
			}
			emit(s, e)
		}
	}
	return out
}
