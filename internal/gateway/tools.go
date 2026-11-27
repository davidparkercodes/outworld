package gateway

import (
	"bytes"
	"context"
	"encoding/base64"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/config"
	"github.com/davidparkercodes/outworld/internal/namedata"
	"github.com/davidparkercodes/outworld/internal/textutil"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/outworld"
)

type leafMode int

const (
	modeMask leafMode = iota
	modeCheck
	modeNarrow
	modeID
	modeName
	modeProvider
	modeModel
	modeVault
	modeRoute
	modeBilling
	modeBinary
	modeBinaryRoute
	modeBinaryPII
	modeCBORPII
	modeNoisyFrame
	modeDecoded
	modeToken
	modeOversize
)

type leaf struct {
	msg  int
	path string
	mode leafMode
	text string
	set  func(string)
	cuts []int
}

type loc struct {
	msg  int
	path string
}

type claimKey struct {
	m uintptr
	k string
}

type fields struct {
	allowImages  bool
	keepMetadata bool
	leaves       []leaf
	commits      []func()
	claims       map[claimKey]bool
	partial      map[claimKey]bool
	serverIDs    map[string]string
	noRoute      map[claimKey]bool
	decoded      map[string][]byte
	frames       map[frameKey]frame
	maxOpaque    int
	encoded      string
	cborScan     func([]byte) ([]string, []string, bool)
	placed       map[loc]int
	inflate      *detect.InflateBudget
	err          error
}

var openAIInputFields = []string{"prompt", "suffix", "input"}

type shape func(any) bool

func enum(vals ...string) shape {
	set := map[string]bool{}
	for _, v := range vals {
		set[v] = true
	}
	return func(v any) bool {
		s, ok := v.(string)
		return ok && set[s]
	}
}

var versionedType = regexp.MustCompile(`^[a-z_]+_20\d{6}$`)

func typeEnum(vals ...string) shape {
	e := enum(vals...)
	return func(v any) bool {
		if e(v) {
			return true
		}
		s, ok := v.(string)
		return ok && versionedType.MatchString(s)
	}
}

func isBool(v any) bool {
	_, ok := v.(bool)
	return ok
}

var (
	modelRE     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/@\[\]-]{0,127}$`)
	mediaTypeRE = regexp.MustCompile(`^[a-z]+/[a-z0-9.+-]+$`)
)

var (
	shapeServiceTier = enum("auto", "default", "flex", "priority", "scale", "standard_only")
	shapeEffort      = enum("none", "minimal", "low", "medium", "high", "xhigh")
	shapeVerbosity   = enum("low", "medium", "high")
	shapeChoice      = enum("auto", "none", "required", "any")
	shapeThinking    = enum("enabled", "disabled", "adaptive")
	shapeRole        = enum("system", "developer", "user", "assistant", "tool", "function")
)

var pageAgeRE = regexp.MustCompile(`^(?:[A-Z][a-z]+ \d{1,2}, \d{4}|\d{4}-\d{2}-\d{2}|\d+ (?:second|minute|hour|day|week|month|year)s? ago)$`)

var schemaNameMaps = map[string]bool{"properties": true, "patternProperties": true, "$defs": true, "definitions": true, "dependentSchemas": true}

var schemaCheck = map[string]bool{
	"pattern": true, "$ref": true, "$id": true, "$anchor": true, "$dynamicRef": true, "$dynamicAnchor": true,
	"type": true, "format": true, "$schema": true, "contentEncoding": true, "contentMediaType": true, "required": true, "dependentRequired": true,
}

var schemaData = map[string]bool{"enum": true, "examples": true, "example": true, "const": true, "default": true}

var schemaKeywords = map[string]bool{
	"title": true, "description": true, "$comment": true, "items": true, "prefixItems": true, "additionalProperties": true,
	"additionalItems": true, "contains": true, "propertyNames": true, "allOf": true, "anyOf": true, "oneOf": true, "not": true,
	"if": true, "then": true, "else": true, "unevaluatedItems": true, "unevaluatedProperties": true, "contentSchema": true,
	"minimum": true, "maximum": true, "exclusiveMinimum": true, "exclusiveMaximum": true, "multipleOf": true, "minLength": true,
	"maxLength": true, "minItems": true, "maxItems": true, "uniqueItems": true, "minProperties": true, "maxProperties": true,
	"minContains": true, "maxContains": true, "readOnly": true, "writeOnly": true, "deprecated": true, "nullable": true,
}

var billingHeaderRE = regexp.MustCompile(`^x-anthropic-billing-header:(?: (?:cc_version=[0-9]+(?:\.[0-9]+){2}(?:\.[0-9a-f]{1,8})?|cc_entrypoint=[a-z][a-z0-9-]{0,31}|cch=[0-9a-f]{1,16});)+$`)

var (
	idPrefixRE = regexp.MustCompile(`^(?:(?:toolu|srvtoolu|mcptoolu|msg)_(?:vrtx_|bdrk_)?|(?:call|file|container|user|resp|fc|rs|item|req|run|thread|asst|batch|msgbatch|skill|ws|ctc|cs|mcp|srv|tool)_)([A-Za-z0-9]{16,})$|^(?:chatcmpl|file)-([A-Za-z0-9]{16,})$`)
	idUUIDRE   = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	idULIDRE   = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)
	idHexRE    = regexp.MustCompile(`^[0-9a-f]{16,128}$`)
	idCCUserRE = regexp.MustCompile(`^user_[0-9a-f]{64}_account_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}_session_[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

const minNameWord = 4

func tableGiven(w string) bool {
	_, ok := namedata.LookupGiven(w)
	return ok
}

func anySurname(w string) bool {
	_, ok := namedata.LookupSurname(w)
	return ok
}

const digitChars = "0123456789"

func isLowerASCII(c byte) bool { return c >= 'a' && c <= 'z' }

func isUpperASCII(c byte) bool { return c >= 'A' && c <= 'Z' }

func isDigitASCII(c byte) bool { return c >= '0' && c <= '9' }

func wordAt(s string, i int) int {
	j := i + 1
	for j < len(s) && isLowerASCII(s[j]) {
		j++
	}
	return j
}

func surnamePrefix(w string) (string, bool) {
	for n := len(w); n >= minNameWord; n-- {
		if anySurname(w[:n]) {
			return w[:n], true
		}
	}
	return "", false
}

var pairNouns = wordList("ball bell bird brook day field fish flower gold jade light love moon park pearl power rock salt sand shield silver spring steel storm street sun water well wind")

func wordList(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(s) {
		m[w] = true
	}
	return m
}

func commonNameWord(w string) bool { return textutil.IsCommonWord(w) || pairNouns[strings.ToLower(w)] }

func namePair(given, surname string) bool {
	if _, ok := namedata.LookupGiven(given); !ok || len(given) < minNameWord {
		return false
	}
	return !(commonNameWord(given) && commonNameWord(surname))
}

func surnameAt(s, given string, k int) bool {
	if k >= len(s) || !isUpperASCII(s[k]) {
		return false
	}
	sur, ok := surnamePrefix(s[k:wordAt(s, k)])
	return ok && namePair(given, sur)
}

func givenThenSurname(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !isUpperASCII(c) && !(isLowerASCII(c) && (i == 0 || !isLowerASCII(s[i-1]) && !isUpperASCII(s[i-1]))) {
			continue
		}
		j := wordAt(s, i)
		given := s[i:j]
		if len(given) < minNameWord || !tableGiven(given) {
			continue
		}
		if surnameAt(s, given, j) {
			return true
		}
		if j < len(s) && (isDigitASCII(s[j]) || isUpperASCII(s[j])) && surnameAt(s, given, j+1) {
			return true
		}
	}
	return false
}

func caseSegments(span string) []string {
	var segs []string
	start := 0
	for i := 1; i < len(span); i++ {
		camel := isLowerASCII(span[i-1]) && isUpperASCII(span[i])
		capsThenLower := isUpperASCII(span[i-1]) && isLowerASCII(span[i]) && i-start >= 2
		if camel || capsThenLower {
			segs = append(segs, span[start:i])
			start = i
		}
	}
	return append(segs, span[start:])
}

func caseRunPair(run string) bool {
	for i := minNameWord; i+minNameWord <= len(run); i++ {
		if !tableGiven(run[:i]) {
			continue
		}
		if sur, ok := surnamePrefix(run[i:]); ok && namePair(run[:i], sur) {
			return true
		}
	}
	return false
}

func segmentPair(span string) bool {
	segs := caseSegments(span)
	for i := 0; i+1 < len(segs); i++ {
		g, rest := segs[i], segs[i+1]
		if len(g) < minNameWord || !tableGiven(g) {
			continue
		}
		if sur, ok := surnamePrefix(capitalize(rest)); ok && namePair(capitalize(g), sur) {
			return true
		}
	}
	return false
}

func capitalize(w string) string {
	if w == "" {
		return w
	}
	return strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
}

func letterSpans(s string) []string {
	var spans []string
	start := -1
	for i := 0; i <= len(s); i++ {
		letter := i < len(s) && (isLowerASCII(s[i]) || isUpperASCII(s[i]))
		if letter && start < 0 {
			start = i
		}
		if !letter && start >= 0 {
			spans = append(spans, s[start:i])
			start = -1
		}
	}
	return spans
}

func nameTail(tail string) bool {
	if givenThenSurname(tail) {
		return true
	}
	for i := 0; i < len(tail); {
		j := i
		upper := isUpperASCII(tail[i])
		for j < len(tail) && (upper && isUpperASCII(tail[j]) || !upper && isLowerASCII(tail[j])) {
			j++
		}
		if j-i >= 2*minNameWord && (upper || i == 0 || !isLowerASCII(tail[i-1]) && !isUpperASCII(tail[i-1])) && caseRunPair(tail[i:j]) {
			return true
		}
		if j == i {
			j++
		}
		i = j
	}
	for _, span := range letterSpans(tail) {
		if len(span) >= 2*minNameWord && segmentPair(span) {
			return true
		}
	}
	return false
}

func digitCount(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if isDigitASCII(s[i]) {
			n++
		}
	}
	return n
}

func idShaped(v any) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	if m := idPrefixRE.FindStringSubmatch(s); m != nil {
		return !nameTail(m[1] + m[2])
	}
	if idUUIDRE.MatchString(s) || idCCUserRE.MatchString(s) {
		return true
	}
	if _, text := hexTextual(s); text {
		return false
	}
	return idULIDRE.MatchString(s) || idHexRE.MatchString(s) && digitCount(s) < len(s)
}

func billingLine(t string) bool {
	if !billingHeaderRE.MatchString(t) {
		return false
	}
	for _, k := range []string{"cc_version=", "cc_entrypoint=", "cch="} {
		if strings.Count(t, " "+k) > 1 {
			return false
		}
	}
	return true
}

func idKey(k string) bool {
	l := strings.ToLower(k)
	return strings.HasSuffix(l, "id") || strings.HasSuffix(l, "key") || strings.HasSuffix(l, "identifier")
}

func idMode(k string, v any) (leafMode, bool) {
	if !idKey(k) || !idShaped(v) {
		return modeCheck, false
	}
	l := strings.ToLower(k)
	if strings.HasSuffix(l, "key") && l != "prompt_cache_key" && l != "idempotency_key" {
		return modeNarrow, true
	}
	return modeID, true
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func keyPath(path string, i int) string { return path + ".#" + strconv.Itoa(i) }

func mapID(m map[string]any) uintptr { return reflect.ValueOf(m).Pointer() }

func (f *fields) claim(m map[string]any, keys ...string) {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			f.claims[claimKey{mapID(m), k}] = true
		}
	}
}

func (f *fields) within(m map[string]any, keys ...string) {
	for _, k := range keys {
		if _, ok := m[k]; ok {
			f.claims[claimKey{mapID(m), k}] = true
			f.partial[claimKey{mapID(m), k}] = true
		}
	}
}

func (f *fields) claimed(m map[string]any, k string) bool { return f.claims[claimKey{mapID(m), k}] }

func (f *fields) protocol(m map[string]any, key string, ok shape) bool {
	v, has := m[key]
	if !has || !ok(v) {
		return false
	}
	f.claim(m, key)
	return true
}

func (f *fields) idField(msg int, path string, m map[string]any, key string) {
	v, ok := m[key].(string)
	if !ok {
		return
	}
	f.idExtras(msg, path, key, v)
	mode, idLike := idMode(key, v)
	if !idLike {
		mode = modeCheck
	}
	f.claim(m, key)
	f.add(msg, path, mode, v, nil)
}

func (f *fields) add(msg int, path string, mode leafMode, text string, set func(string)) {
	if mode == modeMask || mode == modeCheck {
		text = f.encodedValues(msg, path, text, mode == modeMask && set != nil)
	}
	f.leaves = append(f.leaves, leaf{msg: msg, path: f.place(msg, path), mode: mode, text: text, set: set})
}

func (f *fields) addCut(msg int, path string, mode leafMode, text string, cuts []int) {
	f.leaves = append(f.leaves, leaf{msg: msg, path: f.place(msg, path), mode: mode, text: text, cuts: cuts})
}

const leafSuffix = "(leaf "

func (f *fields) place(msg int, path string) string {
	if f.placed == nil {
		f.placed = map[loc]int{}
	}
	at := loc{msg, path}
	f.placed[at]++
	if n := f.placed[at]; n > 1 {
		return path + leafSuffix + strconv.Itoa(n) + ")"
	}
	return path
}

func baseField(field string) string {
	if i := strings.LastIndex(field, leafSuffix); i >= 0 && strings.HasSuffix(field, ")") {
		if _, err := strconv.Atoi(field[i+len(leafSuffix) : len(field)-1]); err == nil {
			return field[:i]
		}
	}
	return field
}

func touchesCut(cuts []int, v outworld.Violation) bool {
	_, atStart := slices.BinarySearch(cuts, v.Start)
	_, atEnd := slices.BinarySearch(cuts, v.End)
	return atStart || atEnd
}

func (f *fields) fail(err error) {
	if f.err == nil {
		f.err = err
	}
}

func (f *fields) badMedia() { f.fail(errBadMedia) }

func (f *fields) nonText() {
	if !f.allowImages {
		f.fail(errNonText)
	}
}

func (f *fields) maskField(msg int, path string, m map[string]any, key string) {
	if s, ok := m[key].(string); ok {
		f.claim(m, key)
		f.add(msg, path, modeMask, s, func(v string) { m[key] = v })
	}
}

func (f *fields) checkField(msg int, path string, m map[string]any, key string, mode leafMode) {
	if s, ok := m[key].(string); ok {
		f.claim(m, key)
		f.add(msg, path, mode, s, nil)
		if mode == modeName {
			if split := splitName(s); split != s {
				f.add(msg, path+"(split)", modeName, split, nil)
			}
		}
	}
}

func splitName(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		switch {
		case r == '_' || r == '-' || r == '.':
			b.WriteByte(' ')
			continue
		case i > 0 && unicode.IsUpper(r) && unicode.IsLower(rs[i-1]):
			b.WriteByte(' ')
		case i > 0 && unicode.IsLetter(r) != unicode.IsLetter(rs[i-1]) && (unicode.IsDigit(r) || unicode.IsDigit(rs[i-1])):
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}
	return b.String()
}

func (f *fields) value(msg int, path string, v any, mode leafMode, set func(any)) {
	switch t := v.(type) {
	case string:
		if set != nil && mode == modeMask {
			f.add(msg, path, modeMask, t, func(s string) { set(s) })
		} else {
			f.add(msg, path, mode, t, nil)
		}
	case []any:
		for i := range t {
			f.value(msg, path+"["+strconv.Itoa(i)+"]", t[i], mode, func(x any) { t[i] = x })
		}
	case map[string]any:
		keyMode := modeCheck
		if mode == modeNarrow {
			keyMode = modeNarrow
		}
		for i, k := range sortedKeys(t) {
			p := keyPath(path, i)
			f.add(msg, p+"(key)", keyMode, k, nil)
			if f.decodeKnown(msg, p, t, k) {
				f.add(msg, p, modeID, t[k].(string), nil)
				continue
			}
			f.idExtras(msg, p, k, t[k])
			if im, ok := idMode(k, t[k]); ok && mode == modeMask {
				f.add(msg, p, im, t[k].(string), nil)
				continue
			}
			f.value(msg, p, t[k], mode, func(x any) { t[k] = x })
		}
	}
}

func (f *fields) data(msg int, path string, holder map[string]any, key string) {
	if v, ok := holder[key]; ok {
		f.claim(holder, key)
		f.value(msg, path, v, modeMask, func(x any) { holder[key] = x })
	}
}

func (f *fields) arguments(msg int, path string, fn map[string]any) {
	args, ok := fn["arguments"].(string)
	if !ok {
		return
	}
	f.claim(fn, "arguments")
	v, err := decodeAny([]byte(args))
	if err != nil {
		f.add(msg, path, modeMask, args, func(s string) { fn["arguments"] = s })
		return
	}
	orig := string(encode(v))
	holder := map[string]any{"v": v}
	f.value(msg, path, v, modeMask, func(x any) { holder["v"] = x })
	f.commits = append(f.commits, func() {
		if out := string(encode(holder["v"])); out != orig {
			fn["arguments"] = out
		}
	})
}

func (f *fields) schema(msg int, path string, v any) {
	switch t := v.(type) {
	case []any:
		for i, x := range t {
			p := path + "[" + strconv.Itoa(i) + "]"
			if str, ok := x.(string); ok {
				f.add(msg, p, modeMask, str, func(y string) { t[i] = y })
				continue
			}
			f.schema(msg, p, x)
		}
	case map[string]any:
		for i, k := range sortedKeys(t) {
			x := t[k]
			known := schemaNameMaps[k] || schemaCheck[k] || schemaData[k] || schemaKeywords[k]
			p := path + "." + k
			if !known {
				p = keyPath(path, i)
				f.add(msg, p+"(key)", modeCheck, k, nil)
			}
			switch {
			case schemaNameMaps[k]:
				if m, ok := x.(map[string]any); ok {
					for j, name := range sortedKeys(m) {
						f.add(msg, keyPath(p, j)+"(key)", modeCheck, name, nil)
						f.schema(msg, keyPath(p, j), m[name])
					}
					continue
				}
				f.schema(msg, p, x)
			case k == "required" || k == "dependentRequired":
				f.value(msg, p, x, modeCheck, nil)
			case schemaCheck[k]:
				f.value(msg, p, x, modeNarrow, nil)
			case schemaData[k]:
				f.value(msg, p, x, modeMask, func(y any) { t[k] = y })
			default:
				if s, ok := x.(string); ok {
					f.add(msg, p, modeMask, s, func(y string) { t[k] = y })
					continue
				}
				f.schema(msg, p, x)
			}
		}
	}
}

func (f *fields) schemaField(msg int, path string, holder map[string]any, key string) {
	if v, ok := holder[key]; ok {
		f.claim(holder, key)
		f.schema(msg, path, v)
	}
}

func (f *fields) customTool(msg int, path string, c map[string]any) {
	f.checkField(msg, path+".name", c, "name", modeName)
	f.maskField(msg, path+".description", c, "description")
	if fm, ok := c["format"].(map[string]any); ok {
		f.within(c, "format")
		f.protocol(fm, "type", enum("text", "grammar"))
		if g, ok := fm["grammar"].(map[string]any); ok {
			f.within(fm, "grammar")
			f.checkField(msg, path+".format.grammar.definition", g, "definition", modeCheck)
			f.protocol(g, "syntax", enum("lark", "regex"))
		}
	}
}

var toolTypes = typeEnum("function", "custom", "web_search", "web_search_preview", "file_search", "code_interpreter", "computer_use_preview", "image_generation", "mcp", "local_shell")

func (f *fields) tool(msg int, path string, v any) {
	t, ok := v.(map[string]any)
	if !ok {
		return
	}
	for i, k := range sortedKeys(t) {
		p := path + "." + k
		switch k {
		case "type":
			f.protocol(t, k, toolTypes)
		case "strict":
			f.protocol(t, k, isBool)
		case "name":
			f.checkField(msg, p, t, k, modeName)
		case "description":
			f.maskField(msg, p, t, k)
		case "parameters", "input_schema":
			f.schemaField(msg, p, t, k)
		case "function":
			f.within(t, k)
			f.tool(msg, p, t[k])
		case "custom":
			if c, ok := t[k].(map[string]any); ok {
				f.within(t, k)
				f.customTool(msg, p, c)
			}
		case "cache_control":
		default:
			f.claim(t, k)
			f.add(msg, keyPath(path, i)+"(key)", modeCheck, k, nil)
			f.value(msg, keyPath(path, i), t[k], modeMask, func(x any) { t[k] = x })
		}
	}
}

func (f *fields) tools(req map[string]any) {
	for _, key := range []string{"tools", "functions"} {
		list, ok := req[key].([]any)
		if !ok {
			continue
		}
		f.within(req, key)
		for i, t := range list {
			f.tool(-1, key+"["+strconv.Itoa(i)+"]", t)
		}
	}
}

func (f *fields) toolChoice(req map[string]any) {
	switch tc := req["tool_choice"].(type) {
	case string:
		f.protocol(req, "tool_choice", shapeChoice)
	case map[string]any:
		f.within(req, "tool_choice")
		f.protocol(tc, "type", enum("auto", "any", "tool", "none", "function", "custom", "required", "allowed_tools"))
		f.checkField(-1, "tool_choice.name", tc, "name", modeName)
		for _, k := range []string{"function", "custom"} {
			if m, ok := tc[k].(map[string]any); ok {
				f.within(tc, k)
				f.checkField(-1, "tool_choice."+k+".name", m, "name", modeName)
			}
		}
	}
}

var mediaMagic = map[string][]string{
	"image/png":       {"\x89PNG\r\n\x1a\n"},
	"image/jpeg":      {"\xff\xd8\xff"},
	"image/gif":       {"GIF87a", "GIF89a"},
	"image/webp":      {"RIFF"},
	"application/pdf": {"%PDF-"},
	"audio/wav":       {"RIFF"},
	"audio/x-wav":     {"RIFF"},
	"audio/mpeg":      {"ID3", "\xff\xfb", "\xff\xf3", "\xff\xf2"},
	"audio/mp3":       {"ID3", "\xff\xfb", "\xff\xf3", "\xff\xf2"},
}

var audioFormats = map[string]string{"wav": "audio/wav", "mp3": "audio/mpeg"}

func decodeB64(s string) ([]byte, bool) { return base64Decode(s, 0) }

func mediaMatches(mediaType, data string) bool {
	magics, ok := mediaMagic[mediaType]
	if !ok {
		return false
	}
	b, ok := decodeB64(data)
	if !ok {
		return false
	}
	for _, m := range magics {
		if bytes.HasPrefix(b, []byte(m)) {
			switch mediaType {
			case "image/webp":
				return len(b) >= 12 && string(b[8:12]) == "WEBP"
			case "audio/wav", "audio/x-wav":
				return len(b) >= 12 && string(b[8:12]) == "WAVE"
			}
			return true
		}
	}
	return false
}

func dataURLMedia(u string) bool {
	rest, ok := strings.CutPrefix(u, "data:")
	if !ok {
		return false
	}
	meta, data, ok := strings.Cut(rest, ",")
	if !ok {
		return false
	}
	mt, ok := strings.CutSuffix(meta, ";base64")
	return ok && mediaTypeRE.MatchString(mt) && mediaMatches(mt, data)
}

func (f *fields) urlField(msg int, path string, m map[string]any, key string) {
	s, ok := m[key].(string)
	if !ok {
		return
	}
	if strings.HasPrefix(s, "data:") {
		if dataURLMedia(s) {
			f.silent(m, key)
		} else {
			f.badMedia()
		}
		return
	}
	f.checkField(msg, path, m, key, modeCheck)
}

func (f *fields) openAIMedia(msg int, path string, part map[string]any, typ string) bool {
	inner, ok := part[typ].(map[string]any)
	if !ok {
		return false
	}
	switch typ {
	case "image_url":
		f.within(part, typ)
		f.urlField(msg, path+".image_url.url", inner, "url")
		f.protocol(inner, "detail", enum("auto", "low", "high"))
	case "input_audio":
		f.within(part, typ)
		format, _ := inner["format"].(string)
		mt, ok := audioFormats[format]
		if !ok {
			f.badMedia()
			break
		}
		f.claim(inner, "format")
		if d, ok := inner["data"].(string); ok && mediaMatches(mt, d) {
			f.silent(inner, "data")
		} else {
			f.badMedia()
		}
	case "file":
		f.within(part, typ)
		f.urlField(msg, path+".file.file_data", inner, "file_data")
		f.idField(msg, path+".file.file_id", inner, "file_id")
		f.checkField(msg, path+".file.filename", inner, "filename", modeCheck)
	default:
		return false
	}
	return true
}

func (f *fields) openAIParts(msg int, path string, holder map[string]any, key string) {
	switch c := holder[key].(type) {
	case string:
		f.maskField(msg, path, holder, key)
	case []any:
		f.within(holder, key)
		for j, p := range c {
			part, ok := p.(map[string]any)
			if !ok {
				continue
			}
			pp := path + "[" + strconv.Itoa(j) + "]"
			typ, _ := part["type"].(string)
			switch typ {
			case "text", "input_text", "output_text":
				f.claim(part, "type")
				f.maskField(msg, pp+".text", part, "text")
			case "refusal":
				f.claim(part, "type")
				f.maskField(msg, pp+".refusal", part, "refusal")
			case "image_url", "input_audio", "file":
				f.nonText()
				if f.openAIMedia(msg, pp, part, typ) {
					f.claim(part, "type")
				}
			default:
				f.nonText()
			}
		}
	}
}

func (f *fields) openAIToolCall(idx int, p string, m map[string]any) {
	f.protocol(m, "type", enum("function", "custom"))
	f.idField(idx, p+".id", m, "id")
	if fn, ok := m["function"].(map[string]any); ok {
		f.within(m, "function")
		f.checkField(idx, p+".function.name", fn, "name", modeName)
		f.arguments(idx, p+".function.arguments", fn)
	}
	if c, ok := m["custom"].(map[string]any); ok {
		f.within(m, "custom")
		f.checkField(idx, p+".custom.name", c, "name", modeName)
		f.maskField(idx, p+".custom.input", c, "input")
	}
}

func (f *fields) openAIMessage(msg map[string]any, idx int) {
	base := "messages[" + strconv.Itoa(idx) + "]"
	f.protocol(msg, "role", shapeRole)
	f.openAIParts(idx, base+".content", msg, "content")
	for _, k := range []string{"reasoning_content", "reasoning", "refusal"} {
		f.maskField(idx, base+"."+k, msg, k)
	}
	f.checkField(idx, base+".name", msg, "name", modeCheck)
	f.idField(idx, base+".tool_call_id", msg, "tool_call_id")
	if tcs, ok := msg["tool_calls"].([]any); ok {
		f.within(msg, "tool_calls")
		for j, tc := range tcs {
			if m, ok := tc.(map[string]any); ok {
				f.openAIToolCall(idx, base+".tool_calls["+strconv.Itoa(j)+"]", m)
			}
		}
	}
	if fn, ok := msg["function_call"].(map[string]any); ok {
		f.within(msg, "function_call")
		f.checkField(idx, base+".function_call.name", fn, "name", modeName)
		f.arguments(idx, base+".function_call.arguments", fn)
	}
	if anns, ok := msg["annotations"].([]any); ok {
		f.within(msg, "annotations")
		for j, a := range anns {
			am, ok := a.(map[string]any)
			if !ok {
				continue
			}
			f.protocol(am, "type", enum("url_citation"))
			if uc, ok := am["url_citation"].(map[string]any); ok {
				f.within(am, "url_citation")
				f.maskField(idx, base+".annotations["+strconv.Itoa(j)+"].url_citation.title", uc, "title")
			}
		}
	}
}

func (f *fields) modelField(msg int, m map[string]any) {
	if s, ok := m["model"].(string); ok && modelRE.MatchString(s) {
		f.claim(m, "model")
		f.add(msg, "model", modeModel, s, nil)
	}
}

func (f *fields) openAI(req map[string]any) {
	f.modelField(-1, req)
	f.protocol(req, "service_tier", shapeServiceTier)
	f.protocol(req, "reasoning_effort", shapeEffort)
	f.protocol(req, "verbosity", shapeVerbosity)
	f.claim(req, "user")
	if msgs, ok := req["messages"].([]any); ok {
		f.within(req, "messages")
		for i, m := range msgs {
			if msg, ok := m.(map[string]any); ok {
				f.openAIMessage(msg, i)
			}
		}
	}
	for _, k := range openAIInputFields {
		f.data(-1, k, req, k)
	}
	if pr, ok := req["prediction"].(map[string]any); ok {
		f.within(req, "prediction")
		f.protocol(pr, "type", enum("content"))
		f.openAIParts(-1, "prediction.content", pr, "content")
	}
	f.data(-1, "stop", req, "stop")
	f.data(-1, "metadata", req, "metadata")
	if ws, ok := req["web_search_options"].(map[string]any); ok {
		f.within(req, "web_search_options")
		f.data(-1, "web_search_options.user_location", ws, "user_location")
	}
	if rf, ok := req["response_format"].(map[string]any); ok {
		f.within(req, "response_format")
		f.protocol(rf, "type", enum("text", "json_object", "json_schema"))
		if js, ok := rf["json_schema"].(map[string]any); ok {
			f.within(rf, "json_schema")
			f.protocol(js, "strict", isBool)
			f.checkField(-1, "response_format.json_schema.name", js, "name", modeName)
			f.maskField(-1, "response_format.json_schema.description", js, "description")
			f.schemaField(-1, "response_format.json_schema.schema", js, "schema")
		}
	}
	f.toolChoice(req)
	f.tools(req)
}

var providerResultTypes = map[string]bool{
	"web_search_tool_result": true, "web_fetch_tool_result": true, "code_execution_tool_result": true,
	"bash_code_execution_tool_result": true, "text_editor_code_execution_tool_result": true, "tool_search_tool_result": true,
}

var citationTypes = enum("char_location", "page_location", "content_block_location", "web_search_result_location", "search_result_location")

func (f *fields) anthropicContent(msg int, path string, holder map[string]any, key string, role string) {
	switch c := holder[key].(type) {
	case string:
		f.maskField(msg, path, holder, key)
	case []any:
		f.within(holder, key)
		prev := f.serverIDs
		f.serverIDs = map[string]string{}
		for j, b := range c {
			if blk, ok := b.(map[string]any); ok {
				f.anthropicBlock(msg, path+"["+strconv.Itoa(j)+"]", blk, role)
				if id, ok := blk["id"].(string); ok && role == "assistant" && blk["type"] == "server_tool_use" {
					name, _ := blk["name"].(string)
					f.serverIDs[id] = name
				}
			}
		}
		f.serverIDs = prev
	}
}

func (f *fields) providerResult(msg int, path string, v any) {
	switch t := v.(type) {
	case string:
		f.add(msg, path, modeProvider, t, nil)
	case []any:
		for i, x := range t {
			f.providerResult(msg, path+"["+strconv.Itoa(i)+"]", x)
		}
	case map[string]any:
		typ, _ := t["type"].(string)
		for i, k := range sortedKeys(t) {
			p := keyPath(path, i)
			x := t[k]
			if k == "encrypted_content" && typ == "web_search_result" && f.opaque(msg, p, t, k) {
				continue
			}
			if k == "page_age" && typ == "web_search_result" {
				if sv, ok := x.(string); ok && pageAgeRE.MatchString(sv) {
					continue
				}
			}
			f.add(msg, p+"(key)", modeProvider, k, nil)
			f.idExtras(msg, p, k, x)
			if mode, ok := idMode(k, x); ok {
				f.add(msg, p, mode, x.(string), nil)
				continue
			}
			f.providerResult(msg, p, x)
		}
	}
}

func resultTypeFor(name string) string {
	if strings.HasPrefix(name, "tool_search_tool_") {
		return "tool_search_tool_result"
	}
	return name + "_tool_result"
}

func (f *fields) serverResult(blk map[string]any) bool {
	id, ok := blk["tool_use_id"].(string)
	if !ok || !idShaped(id) {
		return false
	}
	name, ok := f.serverIDs[id]
	if !ok || resultTypeFor(name) != blk["type"] {
		return false
	}
	if blk["type"] != "web_search_tool_result" {
		return true
	}
	items, ok := blk["content"].([]any)
	if !ok {
		return false
	}
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok || m["type"] != "web_search_result" {
			return false
		}
		enc, _ := m["encrypted_content"].(string)
		if _, ok := f.decodeOnce(enc); !ok {
			return false
		}
	}
	return true
}

func (f *fields) mediaSource(msg int, path string, blk map[string]any, allowText bool) bool {
	src, ok := blk["source"].(map[string]any)
	if !ok {
		return false
	}
	st, _ := src["type"].(string)
	mt, _ := src["media_type"].(string)
	switch st {
	case "base64":
		if !mediaTypeRE.MatchString(mt) {
			f.badMedia()
			return false
		}
		f.within(blk, "source")
		f.claim(src, "type", "media_type")
		if d, ok := src["data"].(string); ok {
			if mt == "text/plain" {
				if raw, ok := decodeB64(d); !allowText || !ok || !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 || bytes.HasPrefix(raw, []byte("\xef\xbb\xbf")) || bytes.HasPrefix(raw, []byte("\xff\xfe")) || bytes.HasPrefix(raw, []byte("\xfe\xff")) {
					f.badMedia()
				} else {
					text := string(raw)
					f.silent(src, "data")
					f.add(msg, path+".source.data", modeMask, text, func(out string) {
						if out != text {
							src["data"] = base64.StdEncoding.EncodeToString([]byte(out))
						}
					})
				}
			} else if mediaMatches(mt, d) {
				f.silent(src, "data")
			} else {
				f.badMedia()
			}
		}
	case "url":
		f.within(blk, "source")
		f.claim(src, "type")
		f.checkField(msg, path+".source.url", src, "url", modeCheck)
	case "file":
		f.within(blk, "source")
		f.claim(src, "type")
		f.idField(msg, path+".source.file_id", src, "file_id")
	default:
		return false
	}
	return true
}

func (f *fields) thinkingBlock(msg int, path string, blk map[string]any) {
	th, isText := blk["thinking"].(string)
	if !isText || !f.opaque(msg, path+".signature", blk, "signature") {
		return
	}
	f.claim(blk, "thinking")
	f.add(msg, path+".thinking", modeVault, th, nil)
}

func (f *fields) providerBlock(msg int, path string, blk map[string]any) {
	f.idField(msg, path+".tool_use_id", blk, "tool_use_id")
	for i, k := range sortedKeys(blk) {
		if k == "type" || k == "tool_use_id" || k == "cache_control" {
			continue
		}
		f.claim(blk, k)
		f.add(msg, keyPath(path, i)+"(key)", modeProvider, k, nil)
		f.providerResult(msg, keyPath(path, i), blk[k])
	}
}

func (f *fields) textBlock(msg int, path string, blk map[string]any, role string) {
	f.maskField(msg, path+".text", blk, "text")
	cites, ok := blk["citations"].([]any)
	if !ok {
		return
	}
	f.within(blk, "citations")
	for i, c := range cites {
		m, ok := c.(map[string]any)
		if !ok {
			continue
		}
		p := path + ".citations[" + strconv.Itoa(i) + "]"
		f.protocol(m, "type", citationTypes)
		if role == "assistant" {
			f.opaque(msg, p+".encrypted_index", m, "encrypted_index")
		}
		for _, k := range []string{"cited_text", "document_title", "title"} {
			f.maskField(msg, p+"."+k, m, k)
		}
	}
}

func (f *fields) documentBlock(msg int, path string, blk map[string]any, role string) {
	f.maskField(msg, path+".title", blk, "title")
	f.maskField(msg, path+".context", blk, "context")
	src, ok := blk["source"].(map[string]any)
	if !ok {
		f.nonText()
		return
	}
	switch st, _ := src["type"].(string); st {
	case "text":
		f.within(blk, "source")
		f.claim(src, "type")
		f.protocol(src, "media_type", enum("text/plain"))
		f.maskField(msg, path+".source.data", src, "data")
	case "content":
		f.within(blk, "source")
		f.claim(src, "type")
		f.anthropicContent(msg, path+".source.content", src, "content", role)
	default:
		f.nonText()
		f.mediaSource(msg, path, blk, true)
	}
}

func (f *fields) anthropicBlock(msg int, path string, blk map[string]any, role string) {
	typ, _ := blk["type"].(string)
	known := true
	switch {
	case typ == "thinking" && role == "assistant":
		f.thinkingBlock(msg, path, blk)
	case typ == "redacted_thinking" && role == "assistant":
		f.opaque(msg, path+".data", blk, "data")
	case providerResultTypes[typ] && f.serverResult(blk):
		f.providerBlock(msg, path, blk)
	case typ == "text":
		f.textBlock(msg, path, blk, role)
	case isToolUseBlock(typ):
		f.idField(msg, path+".id", blk, "id")
		f.checkField(msg, path+".name", blk, "name", modeName)
		f.data(msg, path+".input", blk, "input")
	case typ == "tool_result" || typ == "mcp_tool_result":
		f.idField(msg, path+".tool_use_id", blk, "tool_use_id")
		f.anthropicContent(msg, path+".content", blk, "content", role)
	case typ == "search_result":
		f.maskField(msg, path+".title", blk, "title")
		f.maskField(msg, path+".source", blk, "source")
		f.anthropicContent(msg, path+".content", blk, "content", role)
	case typ == "document":
		f.documentBlock(msg, path, blk, role)
	case typ == "image":
		f.nonText()
		f.mediaSource(msg, path, blk, false)
	default:
		known = false
		if _, has := blk["source"]; has {
			f.nonText()
		}
	}
	if known {
		f.claim(blk, "type")
	}
}

func (f *fields) anthropic(req map[string]any) {
	f.modelField(-1, req)
	f.protocol(req, "service_tier", shapeServiceTier)
	switch sys := req["system"].(type) {
	case string:
		f.maskField(-1, "system", req, "system")
	case []any:
		f.within(req, "system")
		for j, b := range sys {
			blk, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if typ, _ := blk["type"].(string); typ != "text" {
				continue
			}
			f.claim(blk, "type")
			p := "system[" + strconv.Itoa(j) + "].text"
			if t, _ := blk["text"].(string); j == 0 && billingLine(t) {
				f.claim(blk, "text")
				f.add(-1, p, modeBilling, t, nil)
				continue
			}
			f.maskField(-1, p, blk, "text")
		}
	}
	if msgs, ok := req["messages"].([]any); ok {
		f.within(req, "messages")
		for i, m := range msgs {
			if msg, ok := m.(map[string]any); ok {
				f.protocol(msg, "role", shapeRole)
				role, _ := msg["role"].(string)
				f.anthropicContent(i, "messages["+strconv.Itoa(i)+"].content", msg, "content", role)
			}
		}
	}
	f.data(-1, "stop_sequences", req, "stop_sequences")
	if md, ok := req["metadata"].(map[string]any); ok {
		f.within(req, "metadata")
		if f.keepMetadata {
			f.idField(-1, "metadata.user_id", md, "user_id")
		} else {
			f.claim(md, "user_id")
		}
	}
	if th, ok := req["thinking"].(map[string]any); ok {
		f.within(req, "thinking")
		f.protocol(th, "type", shapeThinking)
	}
	if of, ok := req["output_format"].(map[string]any); ok {
		f.within(req, "output_format")
		f.protocol(of, "type", enum("json_schema"))
		f.schemaField(-1, "output_format.schema", of, "schema")
	}
	if servers, ok := req["mcp_servers"].([]any); ok {
		f.within(req, "mcp_servers")
		for i, sv := range servers {
			if m, ok := sv.(map[string]any); ok {
				f.protocol(m, "type", enum("url"))
				f.claim(m, "authorization_token")
				f.checkField(-1, "mcp_servers["+strconv.Itoa(i)+"].name", m, "name", modeName)
			}
		}
	}
	f.container(req)
	if cm, ok := req["context_management"].(map[string]any); ok {
		f.within(req, "context_management")
		if edits, ok := cm["edits"].([]any); ok {
			f.within(cm, "edits")
			for _, e := range edits {
				if em, ok := e.(map[string]any); ok {
					f.protocol(em, "type", typeEnum())
				}
			}
		}
	}
	f.toolChoice(req)
	f.tools(req)
}

func (f *fields) fallbackMap(msg int, path string, m map[string]any) {
	for i, k := range sortedKeys(m) {
		if f.claimed(m, k) {
			if !f.partial[claimKey{mapID(m), k}] {
				if !f.noRoute[claimKey{mapID(m), k}] {
					f.route(msg, path+"."+k, m[k])
				}
				continue
			}
			if sub, ok := m[k].(map[string]any); ok {
				f.fallbackMap(msg, path+"."+k, sub)
			} else if arr, ok := m[k].([]any); ok {
				f.fallbackArray(msg, path+"."+k, arr)
			}
			continue
		}
		f.claim(m, k)
		p := keyPath(path, i)
		f.add(msg, p+"(key)", modeCheck, k, nil)
		mode, _ := idMode(k, m[k])
		if f.decodeKnown(msg, p, m, k) {
			mode = modeID
		}
		f.idExtras(msg, p, k, m[k])
		f.fallbackValue(msg, p, m[k], mode)
	}
}

func (f *fields) route(msg int, path string, v any) {
	switch t := v.(type) {
	case string:
		f.add(msg, path, modeRoute, t, nil)
	case []any:
		for _, x := range t {
			f.route(msg, path, x)
		}
	case map[string]any:
		for _, k := range sortedKeys(t) {
			f.route(msg, path, t[k])
		}
	}
}

func (f *fields) silent(m map[string]any, key string) {
	f.claim(m, key)
	f.noRoute[claimKey{mapID(m), key}] = true
}

func (f *fields) fallbackArray(msg int, path string, arr []any) {
	for i, x := range arr {
		p := path + "[" + strconv.Itoa(i) + "]"
		switch t := x.(type) {
		case map[string]any:
			f.fallbackMap(msg, p, t)
		case []any:
			f.fallbackArray(msg, p, t)
		case string:
			f.add(msg, p, modeCheck, t, nil)
		}
	}
}

func (f *fields) idExtras(msg int, path, k string, v any) {
	s, ok := v.(string)
	if !ok || !idKey(k) || idShaped(s) {
		return
	}
	if split := splitName(s); split != s {
		f.add(msg, path+"(split)", modeCheck, split, nil)
	}
	if b, ok := hexTextual(s); ok {
		f.add(msg, path+"(decoded)", modeCheck, string(b), nil)
		if split := splitName(string(b)); split != string(b) {
			f.add(msg, path+"(decoded split)", modeCheck, split, nil)
		}
	}
}

func (f *fields) fallbackValue(msg int, path string, v any, mode leafMode) {
	switch t := v.(type) {
	case string:
		f.add(msg, path, mode, t, nil)
	case []any:
		for i, x := range t {
			f.fallbackValue(msg, path+"["+strconv.Itoa(i)+"]", x, modeCheck)
		}
	case map[string]any:
		for i, k := range sortedKeys(t) {
			p := keyPath(path, i)
			f.add(msg, p+"(key)", modeCheck, k, nil)
			m, _ := idMode(k, t[k])
			if f.decodeKnown(msg, p, t, k) {
				m = modeID
			}
			f.idExtras(msg, p, k, t[k])
			f.fallbackValue(msg, p, t[k], m)
		}
	}
}

func (f *fields) fallback(req map[string]any) {
	f.fallbackMap(-1, "request", req)
}

func opaqueCap(n int) int {
	if n <= 0 {
		return config.DefaultMaxOpaqueBytes
	}
	return n
}

func encodedMode(m string) string {
	if m == "" {
		return config.EncodedScan
	}
	return m
}

func (s *Server) collect(req map[string]any, api string) *fields {
	f := &fields{allowImages: s.cfg.Policy.AllowImages, keepMetadata: s.cfg.Policy.KeepMetadata, claims: map[claimKey]bool{}, partial: map[claimKey]bool{}, serverIDs: map[string]string{}, noRoute: map[claimKey]bool{}, decoded: map[string][]byte{}, frames: map[frameKey]frame{}, maxOpaque: opaqueCap(s.cfg.MaxOpaqueBytes), encoded: encodedMode(s.cfg.Policy.EncodedValues), cborScan: s.cborScan, inflate: detect.NewInflateBudgetFor(s.cfg.InflateLimits())}
	if api == config.APIAnthropic {
		f.anthropic(req)
	} else {
		f.openAI(req)
	}
	f.fallback(req)
	return f
}

type routeText struct {
	full, narrow, raw, decoded string
	binary                     []string
}

func (s *Server) routingText(req map[string]any, api string) routeText {
	f := s.collect(req, api)
	var full, narrow, raw, decoded, binary []string
	for _, l := range f.leaves {
		switch l.mode {
		case modeMask, modeCheck:
			full = append(full, l.text)
		case modeVault, modeRoute:
			raw = append(raw, l.text)
		case modeBinary, modeBinaryRoute, modeBinaryPII, modeCBORPII:
			binary = append(binary, l.text)
		case modeDecoded, modeToken, modeNoisyFrame:
			decoded = append(decoded, l.text)
		default:
			narrow = append(narrow, l.text)
		}
	}
	return routeText{full: strings.Join(full, "\n"), narrow: strings.Join(narrow, "\n"), raw: strings.Join(raw, "\n"), decoded: strings.Join(decoded, "\n"), binary: binary}
}

func (s *Server) maskFields(ctx context.Context, sess *outworld.Session, req map[string]any, api string) (map[loc]leaf, error) {
	f := s.collect(req, api)
	if f.err != nil {
		return nil, f.err
	}
	narrow := map[loc]leaf{}
	for _, l := range f.leaves {
		sess.SetLocation(l.msg, l.path)
		switch l.mode {
		case modeRoute, modeBinaryRoute:
			continue
		case modeOversize:
			narrow[loc{l.msg, l.path}] = l
			continue
		case modeMask:
		default:
			narrow[loc{l.msg, l.path}] = l
			sess.Check(ctx, l.text)
			continue
		}
		out, err := sess.Mask(ctx, l.text)
		if err != nil {
			return nil, err
		}
		l.set(out)
	}
	for _, c := range f.commits {
		c()
	}
	return narrow, nil
}

const (
	labelWindow = 16
	dateWindow  = 48
)

var (
	numberCues = []string{"ssn", "social", "text", "reach", "passport", "account", "acct", "member", "routing"}
	dateCues   = []string{"dob", "d.o.b", "birth", "born", "birthday", "дата рождения", "fecha de nacimiento", "geburtsdatum", "date de naissance", "nato il", "nata il", "nascimento"}
	gatedIDs   = map[string]bool{"regex:routing": true, "regex:account": true, "regex:policy_number": true}
)

func cueBefore(rs []rune, start, width int, cues []string) bool {
	window := []rune(strings.ToLower(string(rs[max(0, start-width):start])))
	for _, cue := range cues {
		cr := []rune(cue)
		for i := 0; i+len(cr) <= len(window); i++ {
			if string(window[i:i+len(cr)]) != cue {
				continue
			}
			before := i == 0 || !unicode.IsLetter(window[i-1])
			after := i+len(cr) == len(window) || !unicode.IsLetter(window[i+len(cr)])
			if before && after {
				return true
			}
		}
	}
	return false
}

const minCueWord = 3

const maxCueGap = 3

var labelNumberSuffixes = []string{"extension", "ext", "numbers", "number", "num", "nr", "no"}

func keyParts(w string) []string {
	var parts []string
	start := 0
	for i := 1; i <= len(w); i++ {
		if i == len(w) || w[i] == '_' || w[i] == '-' || isLowerASCII(w[i-1]) && isUpperASCII(w[i]) {
			if part := strings.Trim(w[start:i], "_-"); part != "" {
				parts = append(parts, part)
			}
			start = i
		}
	}
	return parts
}

func lastKeyPart(w string) string {
	parts := keyParts(w)
	if len(parts) < 2 {
		return ""
	}
	for len(parts) > 1 && slices.Contains(labelNumberSuffixes, strings.ToLower(parts[len(parts)-1])) {
		parts = parts[:len(parts)-1]
	}
	return strings.ToLower(parts[len(parts)-1])
}

func numberSuffixTrimmed(w string) string {
	lw := strings.ToLower(w)
	for _, suf := range labelNumberSuffixes {
		if strings.HasSuffix(lw, suf) && len(lw) > len(suf) {
			return strings.TrimRight(lw[:len(lw)-len(suf)], "_-")
		}
	}
	return ""
}

func labelStart(rs []rune, start int) int {
	lo := max(0, start-labelWindow)
	for lo > 0 && lo > start-2*labelWindow && (unicode.IsLetter(rs[lo-1]) || rs[lo-1] == '_') {
		lo--
	}
	return lo
}

func phoneLabelBefore(rs []rune, start int) bool {
	window := string(rs[labelStart(rs, start):start])
	words := strings.FieldsFunc(window, func(r rune) bool { return !unicode.IsLetter(r) && r != '_' })
	gap := window
	if i := strings.LastIndexFunc(window, unicode.IsLetter); i >= 0 {
		_, n := utf8.DecodeRuneInString(window[i:])
		gap = window[i+n:]
	}
	adjacent := len(gap) <= maxCueGap && strings.Trim(gap, ": .#=-") == ""
	for i, w := range words {
		for _, part := range []string{w, numberSuffixTrimmed(w), lastKeyPart(w)} {
			if !detect.PhoneLabelWord(part) {
				continue
			}
			if len(part) >= minCueWord || i == len(words)-1 && adjacent {
				return true
			}
		}
	}
	return false
}

func spanMaps(rs []rune, v outworld.Violation) bool {
	return v.Start >= 0 && v.End <= len(rs) && v.Start < v.End
}

var uuidInTextRE = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

const uuidLen = 36

func insideUUID(rs []rune, start, end int) bool {
	lo, hi := max(0, start-uuidLen), min(len(rs), end+uuidLen)
	window := string(rs[lo:hi])
	for _, m := range uuidInTextRE.FindAllStringIndex(window, -1) {
		ms, me := utf8.RuneCountInString(window[:m[0]])+lo, utf8.RuneCountInString(window[:m[1]])+lo
		if ms <= start && end <= me {
			return true
		}
	}
	return false
}

func hexToken(s string) bool {
	return len(s) >= hexTextMinLen && idHexRE.MatchString(strings.ToLower(s)) && strings.Trim(s, digitChars) != "" && strings.ToLower(s) == s
}

func decodedKept(rs []rune, v outworld.Violation) bool {
	if !spanMaps(rs, v) {
		return true
	}
	if exemptGenericSecret(rs, v) || opaqueNoise(v.Source) {
		return false
	}
	if strings.ContainsRune(string(rs[v.Start:v.End]), 0) {
		return false
	}
	if v.Source == "date" {
		return cueBefore(rs, v.Start, dateWindow, dateCues)
	}
	match := string(rs[v.Start:v.End])
	if hexToken(match) || insideUUID(rs, v.Start, v.End) || v.Type == entity.Person && !decodedPersonLine(rs, v) {
		return false
	}
	if v.Source == "regex:context_token" {
		return secretKeyBefore(rs, v.Start)
	}
	if strings.Trim(match, digitChars) != "" || v.Source == "regex:credit_card" {
		return true
	}
	if strings.HasSuffix(v.Source, "_context") || strings.HasPrefix(v.Source, "regex:labelled_") || gatedIDs[v.Source] {
		return true
	}
	return cueBefore(rs, v.Start, labelWindow, numberCues) || phoneLabelBefore(rs, v.Start)
}

var (
	secretKeyRE      = regexp.MustCompile(`(?i)\b(?:credentials?|keys?|tokens?|secrets?|passwords?|passphrases?|signature|apikey)\b`)
	secretLeadRE     = regexp.MustCompile(`^(?:[^\s]*[:=]["'\x60]?|["'\x60(\[<{]|-?>|)$`)
	secretTightRE    = regexp.MustCompile(`^[:="'\x60(\[<{>-]*$`)
	secretBlankGapRE = regexp.MustCompile(`^[\s:=]*$`)
	plainKeyRE       = regexp.MustCompile(`(?i)(?:cache|primary|foreign|sort|partition|idempotency|dedupe|row|map|dict|object|lookup)[\s_-]+$`)
)

const secretKeyWindow = 48

func secretKeyBefore(rs []rune, start int) bool {
	window := string(rs[max(0, start-secretKeyWindow):start])
	lead := secretLead(window)
	for _, m := range secretKeyRE.FindAllStringIndex(window, -1) {
		word := strings.ToLower(window[m[0]:m[1]])
		if strings.HasPrefix(word, "key") && plainKeyRE.MatchString(window[:m[0]]) {
			continue
		}
		gap := window[m[1]:]
		if secretBlankGapRE.MatchString(gap) || lead && !strings.ContainsRune(gap, '\n') {
			return true
		}
	}
	return false
}

const frameTagChars = "BJRZbjrz2*\""

func strayTagBefore(before string) bool {
	last, n := utf8.DecodeLastRuneInString(before)
	if n == 0 || !strings.ContainsRune(frameTagChars, last) {
		return false
	}
	prev, m := utf8.DecodeLastRuneInString(before[:len(before)-n])
	return m == 0 || !unicode.IsLetter(prev) && !unicode.IsDigit(prev)
}

var (
	secretKeyWords = map[string]bool{"key": true, "keys": true, "token": true, "tokens": true, "secret": true, "secrets": true,
		"password": true, "passwords": true, "passphrase": true, "credential": true, "credentials": true, "apikey": true}
)

func secretIdentifier(key, sep string) bool {
	parts := keyParts(key)
	if len(parts) == 0 || !secretKeyWords[strings.ToLower(parts[len(parts)-1])] {
		return false
	}
	if len(parts) == 1 {
		return strings.ContainsAny(sep, ":=")
	}
	return !plainIdentifierParts[strings.ToLower(parts[len(parts)-2])]
}

const (
	identifierKey = `([A-Za-z][A-Za-z0-9_\-]{0,63})`
	keyValueSep   = `["'\x60]?\s{0,8}(?:->|[:=]|\s{1,8}is)?\s{0,8}["'\x60]?\s{0,8}`
)

var identifierTailRE = regexp.MustCompile(identifierKey + keyValueSep + `$`)

func plainIdentifierBefore(rs []rune, start int) bool {
	m := identifierTailRE.FindStringSubmatch(string(rs[max(0, start-secretKeyWindow):start]))
	if m == nil {
		return false
	}
	parts := keyParts(m[1])
	n := len(parts)
	return n >= 2 && plainIdentifierParts[strings.ToLower(parts[n-2])] && keyOrTokenParts[strings.ToLower(parts[n-1])]
}

var genericKeySources = map[string]bool{
	"regex:context_token": true, "regex:kv_secret": true, "regex:labelled_secret": true, "regex:assigned_secret": true,
	"regex:env_secret": true, "regex:credential_cue": true, "regex:query_secret": true, "regex:high_entropy": true,
}

var vendorValueRE = regexp.MustCompile(`^(?:` + vendorPrefixRE.String() + `)`)

func exemptGenericSecret(rs []rune, v outworld.Violation) bool {
	if !genericKeySources[v.Source] {
		return false
	}
	ok := spanMaps(rs, v)
	if !ok || vendorValueRE.MatchString(strings.Trim(string(rs[v.Start:v.End]), "\"' ")) {
		return false
	}
	return plainIdentifierBefore(rs, v.Start)
}

var keyOrTokenParts = map[string]bool{"key": true, "keys": true, "token": true, "tokens": true}

var plainIdentifierParts = map[string]bool{
	"cache": true, "primary": true, "foreign": true, "sort": true, "partition": true, "idempotency": true, "dedupe": true,
	"row": true, "map": true, "dict": true, "object": true, "lookup": true, "context": true, "page": true, "next": true,
	"continuation": true, "cancellation": true, "pagination": true, "cursor": true, "sync": true, "resume": true,
	"request": true, "trace": true, "correlation": true, "max": true, "input": true, "output": true, "prompt": true,
	"completion": true, "total": true, "cached": true, "num": true, "shard": true, "dedup": true, "s3": true, "hash": true,
	"range": true, "public": true, "group": true, "composite": true, "unique": true, "natural": true, "surrogate": true,
	"bucket": true,
}

func secretLead(window string) bool {
	cut := strings.LastIndexFunc(window, unicode.IsSpace)
	if cut >= 0 && window[cut] == '\n' {
		return false
	}
	tail := window
	if cut >= 0 {
		_, n := utf8.DecodeRuneInString(window[cut:])
		tail = window[cut+n:]
	}
	if tail == "" && strayTagBefore(window[:max(cut, 0)]) {
		return false
	}
	if !strings.ContainsFunc(tail, unicode.IsLetter) || secretKeyRE.FindStringIndex(tail) == nil {
		return secretLeadRE.MatchString(tail)
	}
	m := secretKeyRE.FindAllStringIndex(tail, -1)
	return secretTightRE.MatchString(tail[m[len(m)-1][1]:])
}

var noisyFrameSources = map[string]bool{
	"names:first": true, "names:lower": true, "names:handle": true, "names:signoff": true, "names:last": true,
	"regex:honorific_person": true, "regex:greeting_person": true, "regex:ipv4": true, "regex:ipv6": true,
	"regex:labelled_secret": true, "regex:credential_cue": true, "regex:kv_secret": true, "regex:street_address": true,
	"regex:id_document": true, "regex:high_entropy": true,
}

const (
	frameGlue      = "\t\r\n"
	keyedGapWindow = 8
)

var keyedSecretSources = map[string]bool{"regex:assigned_secret": true, "regex:kv_secret": true, "regex:env_secret": true}

func noisyFrameKept(rs []rune, v outworld.Violation) bool {
	if !spanMaps(rs, v) {
		return true
	}
	if noisyFrameSources[v.Source] || !decodedKept(rs, v) || strings.ContainsAny(string(rs[v.Start:v.End]), frameGlue) {
		return false
	}
	if keyedSecretSources[v.Source] {
		return !strings.ContainsAny(string(rs[max(0, v.Start-keyedGapWindow):v.Start]), frameGlue)
	}
	if v.Source != "regex:context_token" {
		return true
	}
	window := string(rs[max(0, v.Start-secretKeyWindow):v.Start])
	all := secretKeyRE.FindAllStringIndex(window, -1)
	return len(all) > 0 && secretBlankGapRE.MatchString(window[all[len(all)-1][1]:])
}

func cborPIIKept(rs []rune, v outworld.Violation) bool {
	if strings.HasPrefix(v.Source, "regex:phone_") && spanMaps(rs, v) {
		return phoneLabelBefore(rs, v.Start)
	}
	return binaryPIIKept(rs, v)
}

func binaryPIIKept(rs []rune, v outworld.Violation) bool {
	ok := spanMaps(rs, v)
	if !ok {
		return true
	}
	if strings.ContainsRune(string(rs[v.Start:v.End]), 0) {
		return false
	}
	switch {
	case v.Source == "regex:email":
		return runEmailRE.MatchString(string(rs[v.Start:v.End]))
	case v.Source == "vault" && v.Type == entity.Person:
		return len(strings.Fields(string(rs[v.Start:v.End]))) >= 2
	case v.Source == "regex:credit_card":
		return true
	case v.Source == "regex:ssn":
		return strings.ContainsRune(string(rs[v.Start:v.End]), '-')
	case strings.HasPrefix(v.Source, "regex:phone_"):
		lo := v.Start
		for lo > 0 && rs[lo-1] != 0 {
			lo--
		}
		return phoneLabelBefore(rs[lo:], v.Start-lo) || gluedLabelBefore(rs[max(lo, labelStart(rs, v.Start)):v.Start])
	}
	return false
}

const maxCueGlue = 2

func gluedLabelBefore(window []rune) bool {
	words := strings.FieldsFunc(string(window), func(r rune) bool { return !unicode.IsLetter(r) })
	if len(words) == 0 {
		return false
	}
	w := strings.ToLower(words[len(words)-1])
	for cut := len(w); cut >= max(0, len(w)-maxCueGlue); cut-- {
		for i := 0; i+minCueWord <= cut; i++ {
			if detect.PhoneLabelWord(w[i:cut]) || detect.PhoneLabelWord(numberSuffixTrimmed(w[i:cut])) {
				return true
			}
		}
	}
	return false
}

var addressNoise = map[string]bool{"regex:ipv4": true, "regex:ipv6": true}

var phraseSources = map[string]bool{"names:full": true, "regex:honorific_person": true, "regex:greeting_person": true}

func payloadAround(rs []rune, start, end int) string {
	text := string(rs)
	lo := strings.LastIndex(string(rs[:start]), payloadSeparator)
	hi := strings.Index(string(rs[end:]), payloadSeparator)
	from := 0
	if lo >= 0 {
		from = lo + len(payloadSeparator)
	}
	to := len(text)
	if hi >= 0 {
		to = len(string(rs[:end])) + hi
	}
	return text[from:to]
}

func tokenKept(rs []rune, v outworld.Violation) bool {
	ok := spanMaps(rs, v)
	if !ok {
		return true
	}
	if addressNoise[v.Source] {
		return v.Start > 0 && rs[v.Start-1] == '@'
	}
	if phraseSources[v.Source] {
		return strings.ContainsRune(payloadAround(rs, v.Start, v.End), ' ')
	}
	return narrowSource(v.Source) && v.Source != "names:handle"
}

func opaqueNoise(src string) bool { return src == "regex:high_entropy" || src == "regex:bare_hex" }

func narrowSource(src string) bool {
	return strings.HasPrefix(src, "regex:") || src == "vault" || src == "dictionary" || src == "date" || strings.HasSuffix(src, ":error")
}

func keptFor(l leaf, rs []rune, v outworld.Violation) bool {
	src := v.Source
	if strings.HasSuffix(src, ":error") {
		return true
	}
	if touchesCut(l.cuts, v) {
		return false
	}
	switch l.mode {
	case modeNarrow, modeName:
		return narrowSource(src)
	case modeProvider, modeModel:
		return narrowSource(src) && src != "date"
	case modeVault:
		return src == "vault"
	case modeBinary:
		return binaryFinding(v)
	case modeBinaryPII:
		return binaryPIIKept(rs, v)
	case modeCBORPII:
		return cborPIIKept(rs, v)
	case modeNoisyFrame:
		return noisyFrameKept(rs, v)
	case modeDecoded:
		return decodedKept(rs, v)
	case modeToken:
		return tokenKept(rs, v) && decodedKept(rs, v)
	case modeBilling:
		return narrowSource(src) && !opaqueNoise(src)
	case modeID:
		if !narrowSource(src) || opaqueNoise(src) {
			return false
		}
		return src != "date" || strings.ContainsAny(l.text, "-_:/. ")
	}
	return true
}

func nameTokens(text string) [][2]int {
	var toks [][2]int
	rs := []rune(text)
	start := -1
	for i, r := range rs {
		alnum := unicode.IsLetter(r) || unicode.IsDigit(r)
		boundary := alnum && start >= 0 && unicode.IsUpper(r) && i > 0 && unicode.IsLower(rs[i-1])
		if (!alnum || boundary) && start >= 0 {
			toks = append(toks, [2]int{start, i})
			start = -1
		}
		if alnum && start < 0 {
			start = i
		}
	}
	if start >= 0 {
		toks = append(toks, [2]int{start, len(rs)})
	}
	return toks
}

func vaultNameTokens(text string, vs []outworld.Violation) int {
	covered := map[int]bool{}
	for _, v := range vs {
		for i, t := range nameTokens(text) {
			if v.Start < t[1] && t[0] < v.End {
				covered[i] = true
			}
		}
	}
	return len(covered)
}

func (f *fields) container(req map[string]any) {
	switch c := req["container"].(type) {
	case string:
		if idShaped(c) {
			f.claim(req, "container")
			f.add(-1, "container", modeID, c, nil)
		}
	case map[string]any:
		f.within(req, "container")
		f.idField(-1, "container.id", c, "id")
		if skills, ok := c["skills"].([]any); ok {
			f.within(c, "skills")
			for i, sk := range skills {
				if m, ok := sk.(map[string]any); ok {
					f.protocol(m, "type", enum("anthropic", "custom"))
					f.idField(-1, "container.skills["+strconv.Itoa(i)+"].skill_id", m, "skill_id")
				}
			}
		}
	}
}
