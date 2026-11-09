package vault

import (
	"errors"
	"fmt"
	"strings"
	"sync"

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

type Vault struct {
	mu          sync.RWMutex
	backend     Backend
	secret      []byte
	records     []Record
	byReal      map[string]int
	bySur       map[string]int
	byPhone     map[string]int
	reals       map[string]bool
	reserved    map[string]bool
	realMatcher *textutil.Matcher
	surMatcher  *textutil.Matcher
}

func Open(b Backend) (*Vault, error) {
	secret, recs, err := b.Load()
	if err != nil {
		return nil, err
	}
	if len(secret) < 16 {
		return nil, errors.New("vault: missing or short vault secret")
	}
	v := &Vault{
		backend: b, secret: secret,
		byReal: map[string]int{}, bySur: map[string]int{}, byPhone: map[string]int{}, reals: map[string]bool{}, reserved: map[string]bool{},
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
		var b strings.Builder
		for _, r := range s {
			if r >= '0' && r <= '9' {
				b.WriteRune(r)
			}
		}
		return b.String()
	default:
		return textutil.Normalize(s)
	}
}

func realKey(kind, real string) string { return kind + "\x00" + NormalizeReal(kind, real) }

func (v *Vault) index(r Record) {
	if _, ok := v.byReal[realKey(r.Kind, r.Real)]; ok {
		return
	}
	if _, ok := v.bySur[textutil.Normalize(r.Surrogate)]; ok {
		return
	}
	v.records = append(v.records, r)
	id := len(v.records) - 1
	v.byReal[realKey(r.Kind, r.Real)] = id
	v.bySur[textutil.Normalize(r.Surrogate)] = id
	if r.Kind == "phone" {
		if k := phoneKey(r.Surrogate); k != "" {
			if _, ok := v.byPhone[k]; !ok {
				v.byPhone[k] = id
			}
		}
	}
	v.reals[textutil.Normalize(r.Real)] = true
	v.realMatcher.Add(r.Real, id, textutil.RuleFor(r.Real))
	v.surMatcher.Add(r.Surrogate, id, textutil.AnyCase)
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

func (v *Vault) Reserve(values ...string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, s := range values {
		if n := textutil.Normalize(s); n != "" {
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

func (v *Vault) Reverse(surrogate string) (Record, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if id, ok := v.bySur[textutil.Normalize(surrogate)]; ok {
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
	n := textutil.Normalize(s)
	return v.reals[n] || v.reserved[n]
}

func (v *Vault) IsSurrogate(s string) bool {
	_, ok := v.Reverse(s)
	return ok
}

func (v *Vault) usable(real, cand string) bool {
	n := textutil.Normalize(cand)
	if n == "" || n == textutil.Normalize(real) || v.reals[n] || v.reserved[n] {
		return false
	}
	_, taken := v.bySur[n]
	return !taken
}

func (v *Vault) Assign(kind, real string, gen func(attempt int) string) (string, error) {
	if s, ok := v.Get(kind, real); ok {
		return s, nil
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	var out string
	err := v.backend.Update(func(newer []Record) ([]Record, error) {
		for _, r := range newer {
			v.index(r)
		}
		if id, ok := v.byReal[realKey(kind, real)]; ok {
			out = v.records[id].Surrogate
			return nil, nil
		}
		for attempt := 0; attempt < 20000; attempt++ {
			cand := gen(attempt)
			if v.usable(real, cand) {
				rec := Record{Kind: kind, Real: real, Surrogate: cand}
				v.index(rec)
				out = cand
				return []Record{rec}, nil
			}
		}
		return nil, fmt.Errorf("%w for kind %q", ErrExhausted, kind)
	})
	return out, err
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
	return v.surMatcher.MaxLen()
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
