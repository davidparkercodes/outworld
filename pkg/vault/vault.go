package vault

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/davidparkercodes/outworld/internal/textutil"
)

type Record struct {
	Kind      string `json:"k"`
	Real      string `json:"r"`
	Surrogate string `json:"s"`
}

type Backend interface {
	Load() (secret []byte, records []Record, err error)
	Update(fn func(newer []Record) ([]Record, error)) error
	Close() error
}

var ErrExhausted = errors.New("vault: could not find a collision-free surrogate")

var ErrEmptyReal = errors.New("vault: value is empty after normalisation")

type Vault struct {
	mu           sync.RWMutex
	backend      Backend
	secret       []byte
	records      []Record
	byReal       map[string]int
	bySur        map[string]int
	byPhone      map[string]int
	reals        map[string]bool
	reserved     map[string]bool
	realMatcher  *textutil.Matcher
	surMatcher   *textutil.Matcher
	marked       bool
	legacy       int
	forms        map[string][]int
	formCount    map[string]int
	formPrefix   map[string]bool
	formByReal   map[string]int
	formBySur    map[string]int
	exactReals   map[string]bool
	maxForm      int
	droppedForm  int
	nameForm     map[string]int
	nameForms    map[string]int
	nameFormNorm map[string]string
}

const formatKind = "vault.format"

var formatRecord = Record{Kind: formatKind, Real: "#", Surrogate: "##"}

func sameName(a Record, kind, real string) bool {
	return a.Kind != kind && strings.HasPrefix(a.Kind, "person.") && strings.HasPrefix(kind, "person.") && normKey(a.Real) == normKey(real)
}

func Open(b Backend) (*Vault, error) {
	secret, recs, err := b.Load()
	if err != nil {
		return nil, err
	}
	if len(secret) < 16 {
		b.Close()
		return nil, errors.New("vault: missing or short vault secret")
	}
	v := &Vault{
		backend: b, secret: secret,
		byReal: map[string]int{}, bySur: map[string]int{}, byPhone: map[string]int{}, reals: map[string]bool{}, reserved: map[string]bool{},
		forms: map[string][]int{}, formCount: map[string]int{}, formPrefix: map[string]bool{}, formByReal: map[string]int{}, formBySur: map[string]int{}, exactReals: map[string]bool{}, nameForm: map[string]int{}, nameForms: map[string]int{}, nameFormNorm: map[string]string{},
		realMatcher: textutil.NewMatcher(), surMatcher: textutil.NewMatcher(),
	}
	for _, r := range recs {
		v.index(r)
	}
	return v, nil
}

func NormalizeReal(kind, s string) string {
	switch kind {
	case "phone":
		f := textutil.FoldForms(s)
		var b strings.Builder
		for _, r := range f {
			if r >= '0' && r <= '9' {
				b.WriteRune(r)
			}
		}
		if b.Len() == 0 {
			return normKey(s)
		}
		return b.String()
	default:
		return normKey(s)
	}
}

func normKey(s string) string { return textutil.Normalize(textutil.FoldForms(s)) }

func realKey(kind, real string) string { return kind + "\x00" + NormalizeReal(kind, real) }

func (v *Vault) index(r Record) {
	if r.Kind == phoneFormKind {
		v.indexForm(r)
		return
	}
	if r.Kind == nameFormKind {
		v.indexNameForm(r)
		return
	}
	if r.Kind == formatKind {
		if !v.marked {
			v.marked, v.legacy = true, len(v.records)
		}
		return
	}
	if prev, ok := v.byReal[realKey(r.Kind, r.Real)]; ok {
		if v.records[prev].Surrogate == r.Surrogate {
			return
		}
		if _, known := v.bySur[normKey(r.Surrogate)]; known {
			return
		}
		v.records = append(v.records, r)
		id := len(v.records) - 1
		v.bySur[normKey(r.Surrogate)] = id
		v.surMatcher.Add(r.Surrogate, id, textutil.AnyCase)
		return
	}
	if id, ok := v.bySur[normKey(r.Surrogate)]; ok {
		if sameName(v.records[id], r.Kind, r.Real) {
			v.records = append(v.records, r)
			v.byReal[realKey(r.Kind, r.Real)] = len(v.records) - 1
		}
		return
	}
	v.records = append(v.records, r)
	id := len(v.records) - 1
	v.byReal[realKey(r.Kind, r.Real)] = id
	v.bySur[normKey(r.Surrogate)] = id
	if r.Kind == "phone" {
		if k := phoneKey(r.Surrogate); k != "" {
			if _, ok := v.byPhone[k]; !ok {
				v.byPhone[k] = id
			}
		}
	}
	v.reals[normKey(r.Real)] = true
	v.exactReals[r.Real] = true
	v.realMatcher.Add(r.Real, id, textutil.RuleFor(r.Real))
	v.surMatcher.Add(r.Surrogate, id, textutil.AnyCase)
}

const (
	phoneFormKind          = "phone.form"
	MaxPhoneForms          = 32
	MaxPhoneFormsInContext = 256
	formKeyBytes           = 4
)

func (v *Vault) formUsable(real, emitted string, limit int) bool {
	if _, ok := v.formByReal[real]; ok || v.formCount[NormalizeReal("phone", real)] >= limit || len(emitted) < formKeyBytes {
		return false
	}
	if _, ok := v.formBySur[emitted]; ok {
		return false
	}
	if v.exactReals[emitted] {
		return false
	}
	n := normKey(emitted)
	if id, taken := v.bySur[n]; taken {
		r := v.records[id]
		return r.Kind == "phone" && NormalizeReal("phone", r.Real) == NormalizeReal("phone", real)
	}
	return !v.reals[n] && !v.reserved[n]
}

func (v *Vault) FormOwner(emitted string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if id, ok := v.formBySur[emitted]; ok {
		return v.records[id].Real, true
	}
	return "", false
}

func (v *Vault) indexForm(r Record) {
	if !v.formUsable(r.Real, r.Surrogate, MaxPhoneFormsInContext) {
		v.droppedForm++
		return
	}
	v.records = append(v.records, r)
	id := len(v.records) - 1
	v.formByReal[r.Real] = id
	v.formBySur[r.Surrogate] = id
	v.formCount[NormalizeReal("phone", r.Real)]++
	v.maxForm = max(v.maxForm, len(r.Surrogate))
	k := r.Surrogate[:formKeyBytes]
	v.forms[k] = append(v.forms[k], id)
	for n := 1; n < formKeyBytes; n++ {
		v.formPrefix[r.Surrogate[:n]] = true
	}
	v.realMatcher.Add(r.Real, id, textutil.RuleFor(r.Real))
}

const (
	nameFormKind = "name.form"
	MaxNameForms = 16
)

func nameFormBase(real string) string { return textutil.Fold(textutil.FoldForms(real)) }

func (v *Vault) nameFormUsable(real, emitted string) bool {
	if _, ok := v.nameForm[real]; ok || len(emitted) < formKeyBytes || v.nameForms[nameFormBase(real)] >= MaxNameForms {
		return false
	}
	if _, ok := v.formBySur[emitted]; ok || v.exactReals[emitted] {
		return false
	}
	n := normKey(emitted)
	if b, taken := v.nameFormNorm[n]; taken && b != nameFormBase(real) {
		return false
	}
	if id, taken := v.bySur[n]; taken {
		r := v.records[id]
		return strings.HasPrefix(r.Kind, "person.") && nameFormBase(r.Real) == nameFormBase(real)
	}
	return !v.reals[n] && !v.reserved[n]
}

func formWordRune(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r)
}

func nameFormBounded(text string, start, end int) bool {
	if start > 0 {
		if r, _ := utf8.DecodeLastRuneInString(text[:start]); formWordRune(r) {
			return false
		}
	}
	if end < len(text) {
		if r, _ := utf8.DecodeRuneInString(text[end:]); formWordRune(r) {
			return false
		}
	}
	return true
}

func (v *Vault) indexNameForm(r Record) {
	if !v.nameFormUsable(r.Real, r.Surrogate) {
		v.droppedForm++
		return
	}
	v.records = append(v.records, r)
	id := len(v.records) - 1
	v.nameForm[r.Real] = id
	v.nameFormNorm[normKey(r.Surrogate)] = nameFormBase(r.Real)
	v.formBySur[r.Surrogate] = id
	v.nameForms[nameFormBase(r.Real)]++
	v.maxForm = max(v.maxForm, len(r.Surrogate))
	k := r.Surrogate[:formKeyBytes]
	v.forms[k] = append(v.forms[k], id)
	for n := 1; n < formKeyBytes; n++ {
		v.formPrefix[r.Surrogate[:n]] = true
	}
}

func (v *Vault) NameForm(real string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if id, ok := v.nameForm[real]; ok {
		return v.records[id].Surrogate, true
	}
	return "", false
}

func (v *Vault) NameFormFits(real, emitted string, pending int) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.nameFormUsable(real, emitted) && v.nameForms[nameFormBase(real)]+pending < MaxNameForms
}

func (v *Vault) AddNameForms(forms []Record) (int, error) {
	if len(forms) == 0 {
		return 0, nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var add []Record
	err := v.backend.Update(func(newer []Record) ([]Record, error) {
		add = nil
		for _, r := range newer {
			v.index(r)
		}
		seen := map[string]bool{}
		counts := map[string]int{}
		for _, f := range forms {
			b := nameFormBase(f.Real)
			if seen[f.Surrogate] || !v.nameFormUsable(f.Real, f.Surrogate) || v.nameForms[b]+counts[b] >= MaxNameForms {
				continue
			}
			seen[f.Surrogate] = true
			counts[b]++
			add = append(add, Record{Kind: nameFormKind, Real: f.Real, Surrogate: f.Surrogate})
		}
		if len(add) > 0 && !v.marked {
			add = append([]Record{formatRecord}, add...)
		}
		return add, nil
	})
	if err != nil {
		return 0, err
	}
	n := 0
	for _, r := range add {
		v.index(r)
		if r.Kind == nameFormKind {
			n++
		}
	}
	return n, nil
}

func (v *Vault) PhoneForm(real string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if id, ok := v.formByReal[real]; ok {
		return v.records[id].Surrogate, true
	}
	return "", false
}

func (v *Vault) AddPhoneForm(real, emitted string) (bool, error) {
	return v.addPhoneForm(real, emitted, MaxPhoneForms)
}

func (v *Vault) AddPhoneFormInContext(real, emitted string) (bool, error) {
	return v.addPhoneForm(real, emitted, MaxPhoneFormsInContext)
}

func (v *Vault) addPhoneForm(real, emitted string, limit int) (bool, error) {
	if real == emitted {
		return false, nil
	}
	v.mu.RLock()
	usable := v.formUsable(real, emitted, limit)
	v.mu.RUnlock()
	if !usable {
		return false, nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var add []Record
	err := v.backend.Update(func(newer []Record) ([]Record, error) {
		add = nil
		for _, r := range newer {
			v.index(r)
		}
		if !v.formUsable(real, emitted, limit) {
			return nil, nil
		}
		add = []Record{{Kind: phoneFormKind, Real: real, Surrogate: emitted}}
		if !v.marked {
			add = append([]Record{formatRecord}, add...)
		}
		return add, nil
	})
	if err != nil {
		return false, err
	}
	for _, r := range add {
		v.index(r)
	}
	return len(add) > 0, nil
}

func (v *Vault) FindForms(text string, keep func(text string, start, end int) bool) []Hit {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if len(v.forms) == 0 {
		return nil
	}
	var out []Hit
	for i := 0; i+formKeyBytes <= len(text); {
		best := -1
		for _, id := range v.forms[text[i:i+formKeyBytes]] {
			sur := v.records[id].Surrogate
			if strings.HasPrefix(text[i:], sur) && (best < 0 || len(sur) > len(v.records[best].Surrogate)) && (keep == nil || keep(text, i, i+len(sur))) && (v.records[id].Kind != nameFormKind || nameFormBounded(text, i, i+len(sur))) {
				best = id
			}
		}
		if best < 0 {
			i++
			continue
		}
		end := i + len(v.records[best].Surrogate)
		out = append(out, Hit{Start: i, End: end, Record: v.records[best]})
		i = end
	}
	return out
}

func (v *Vault) FormPrefixAt(text string, i int) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	rest := text[i:]
	if len(rest) < formKeyBytes {
		return v.formPrefix[rest]
	}
	for _, id := range v.forms[rest[:formKeyBytes]] {
		sur := v.records[id].Surrogate
		if len(rest) < len(sur) && strings.HasPrefix(sur, rest) {
			return true
		}
		if v.records[id].Kind == nameFormKind && rest == sur {
			return true
		}
	}
	return false
}

func (v *Vault) Secret() []byte { return append([]byte(nil), v.secret...) }

func (v *Vault) Len() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.records)
}

func (v *Vault) Records() []Record {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return append([]Record(nil), v.records...)
}

func (v *Vault) RecordsFrom(n int) []Record {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if n < 0 || n > len(v.records) {
		n = 0
	}
	return append([]Record(nil), v.records[n:]...)
}

func (v *Vault) Reserve(values ...string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, s := range values {
		if n := normKey(s); n != "" {
			v.reserved[n] = true
		}
	}
}

func (v *Vault) Get(kind, real string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if id, ok := v.byReal[realKey(kind, real)]; ok {
		return v.records[id].Surrogate, true
	}
	return "", false
}

func (v *Vault) Legacy(kind, real string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	id, ok := v.byReal[realKey(kind, real)]
	return ok && (!v.marked || id < v.legacy)
}

func (v *Vault) Reverse(surrogate string) (Record, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if id, ok := v.bySur[normKey(surrogate)]; ok {
		return v.records[id], true
	}
	return Record{}, false
}

func phoneKey(s string) string {
	d := NormalizeReal("phone", s)
	if len(d) == 11 && d[0] == '1' {
		d = d[1:]
	}
	if len(d) < 7 {
		return ""
	}
	return d
}

func (v *Vault) ReversePhone(s string) (Record, bool) {
	k := phoneKey(s)
	if k == "" {
		return Record{}, false
	}
	v.mu.RLock()
	defer v.mu.RUnlock()
	if id, ok := v.byPhone[k]; ok {
		return v.records[id], true
	}
	return Record{}, false
}

func (v *Vault) IsReal(s string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	n := normKey(s)
	return v.reals[n] || v.reserved[n]
}

func (v *Vault) IsSurrogate(s string) bool {
	_, ok := v.Reverse(s)
	return ok
}

func (v *Vault) usable(kind, real, cand string) bool {
	n := normKey(cand)
	if n == "" || n == normKey(real) {
		return false
	}
	if id, taken := v.bySur[n]; taken {
		return sameName(v.records[id], kind, real)
	}
	return !v.reals[n] && !v.reserved[n]
}

func (v *Vault) Assign(kind, real string, gen func(attempt int) string) (string, error) {
	if NormalizeReal(kind, real) == "" {
		return "", fmt.Errorf("%w for kind %q", ErrEmptyReal, kind)
	}
	if s, ok := v.Get(kind, real); ok {
		return s, nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var out string
	var add []Record
	err := v.backend.Update(func(newer []Record) ([]Record, error) {
		add = nil
		for _, r := range newer {
			v.index(r)
		}
		if id, ok := v.byReal[realKey(kind, real)]; ok {
			out = v.records[id].Surrogate
			return nil, nil
		}
		for attempt := 0; attempt < 20000; attempt++ {
			cand := gen(attempt)
			if v.usable(kind, real, cand) {
				rec := Record{Kind: kind, Real: real, Surrogate: cand}
				add = []Record{rec}
				if !v.marked {
					add = []Record{formatRecord, rec}
				}
				out = cand
				return add, nil
			}
		}
		return nil, fmt.Errorf("%w for kind %q", ErrExhausted, kind)
	})
	if err != nil {
		return "", err
	}
	for _, r := range add {
		v.index(r)
	}
	return out, nil
}

type Hit struct {
	Start, End int
	Record     Record
}

func (v *Vault) FindReals(text string) []Hit {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.hits(v.realMatcher, text)
}

func (v *Vault) FindSurrogates(text string) []Hit {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.hits(v.surMatcher, text)
}

func (v *Vault) hits(m *textutil.Matcher, text string) []Hit {
	var out []Hit
	for _, mm := range m.FindAll(text) {
		out = append(out, Hit{Start: mm.Start, End: mm.End, Record: v.records[mm.ID]})
	}
	return out
}

func (v *Vault) SurrogatePrefixAt(text string, i int) bool {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.surMatcher.IsPrefixAt(text, i)
}

func (v *Vault) MaxSurrogateLen() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return max(v.surMatcher.MaxLen(), v.maxForm)
}

func (v *Vault) DroppedForms() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.droppedForm
}

func (v *Vault) FormsFull(real string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.formCount[NormalizeReal("phone", real)] >= MaxPhoneForms
}

func (v *Vault) Close() error { return v.backend.Close() }

type MemoryBackend struct {
	mu      sync.Mutex
	secret  []byte
	records []Record
}

func NewMemoryBackend(secret []byte) *MemoryBackend {
	return &MemoryBackend{secret: append([]byte(nil), secret...)}
}

func (m *MemoryBackend) Load() ([]byte, []Record, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.secret, append([]Record(nil), m.records...), nil
}

func (m *MemoryBackend) Update(fn func([]Record) ([]Record, error)) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	add, err := fn(nil)
	if err != nil {
		return err
	}
	m.records = append(m.records, add...)
	return nil
}

func (m *MemoryBackend) Close() error { return nil }
