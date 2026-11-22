package outworld

import (
	"container/list"
	"crypto/sha256"
	"sync"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

const DefaultMemoBytes = 64 << 20

type memoKey [sha256.Size]byte

type memoEntry struct {
	key   memoKey
	out   string
	stats Stats
	dates map[string]string
	names []string
	size  int64
}

type Memo struct {
	mu   sync.Mutex
	max  int64
	used int64
	ll   *list.List
	idx  map[memoKey]*list.Element
	hits uint64
	miss uint64
}

func NewMemo(maxBytes int64) *Memo {
	if maxBytes <= 0 {
		return nil
	}
	return &Memo{max: maxBytes, ll: list.New(), idx: map[memoKey]*list.Element{}}
}

func memoKeyFor(scope, text string) memoKey {
	h := sha256.New()
	h.Write([]byte(scope))
	h.Write([]byte{0})
	h.Write([]byte(text))
	var k memoKey
	h.Sum(k[:0])
	return k
}

func (m *Memo) get(k memoKey) (memoEntry, bool) {
	if m == nil {
		return memoEntry{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	el, ok := m.idx[k]
	if !ok {
		m.miss++
		return memoEntry{}, false
	}
	m.hits++
	m.ll.MoveToFront(el)
	return *el.Value.(*memoEntry), true
}

func (m *Memo) put(e memoEntry) {
	if m == nil {
		return
	}
	e.size = int64(len(e.out)) + sha256.Size + 96
	for k, v := range e.dates {
		e.size += int64(len(k) + len(v) + 32)
	}
	for _, n := range e.names {
		e.size += int64(len(n) + 16)
	}
	if e.size > m.max {
		return
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if el, ok := m.idx[e.key]; ok {
		m.used -= el.Value.(*memoEntry).size
		el.Value = &e
		m.used += e.size
		m.ll.MoveToFront(el)
	} else {
		m.idx[e.key] = m.ll.PushFront(&e)
		m.used += e.size
	}
	for m.used > m.max {
		last := m.ll.Back()
		if last == nil {
			break
		}
		old := last.Value.(*memoEntry)
		m.ll.Remove(last)
		delete(m.idx, old.key)
		m.used -= old.size
	}
}

type MemoStats struct {
	Entries int    `json:"entries"`
	Bytes   int64  `json:"bytes"`
	Hits    uint64 `json:"hits"`
	Misses  uint64 `json:"misses"`
}

func (m *Memo) Stats() MemoStats {
	if m == nil {
		return MemoStats{}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return MemoStats{Entries: m.ll.Len(), Bytes: m.used, Hits: m.hits, Misses: m.miss}
}

func (e *Engine) MemoStats() MemoStats { return e.memo.Stats() }

func (s *Session) memoLookup(text string) (string, bool) {
	ent, ok := s.e.memo.get(memoKeyFor(s.scope, text))
	if !ok {
		return "", false
	}
	s.learnNames(ent.names)
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.priming {
		return text, true
	}
	s.stats.Masked += ent.stats.Masked
	s.stats.Stripped += ent.stats.Stripped
	s.stats.Shifted += ent.stats.Shifted
	s.stats.Generalized += ent.stats.Generalized
	for t, n := range ent.stats.ByType {
		s.stats.ByType[t] += n
	}
	for k, v := range ent.dates {
		if _, dup := s.dates[k]; !dup {
			s.dates[k] = v
		}
	}
	return ent.out, true
}

type memoMark struct {
	stats      Stats
	dates      map[string]bool
	violations int
}

func (s *Session) memoMark() memoMark {
	if s.e.memo == nil {
		return memoMark{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	mk := memoMark{stats: s.stats, dates: make(map[string]bool, len(s.dates)), violations: len(s.violations)}
	mk.stats.ByType = make(map[entity.Type]int, len(s.stats.ByType))
	for t, n := range s.stats.ByType {
		mk.stats.ByType[t] = n
	}
	for k := range s.dates {
		mk.dates[k] = true
	}
	return mk
}

func (s *Session) memoStore(text, out string, names []string, mk memoMark) {
	if s.e.memo == nil {
		return
	}
	s.mu.Lock()
	if s.priming || len(s.violations) != mk.violations {
		s.mu.Unlock()
		return
	}
	ent := memoEntry{key: memoKeyFor(s.scope, text), out: out, names: names}
	ent.stats = Stats{
		Masked:      s.stats.Masked - mk.stats.Masked,
		Stripped:    s.stats.Stripped - mk.stats.Stripped,
		Shifted:     s.stats.Shifted - mk.stats.Shifted,
		Generalized: s.stats.Generalized - mk.stats.Generalized,
		ByType:      map[entity.Type]int{},
	}
	for t, n := range s.stats.ByType {
		if d := n - mk.stats.ByType[t]; d != 0 {
			ent.stats.ByType[t] = d
		}
	}
	for k, v := range s.dates {
		if !mk.dates[k] {
			if ent.dates == nil {
				ent.dates = map[string]string{}
			}
			ent.dates[k] = v
		}
	}
	s.mu.Unlock()
	s.e.memo.put(ent)
}

func (s *Session) Remember(rehydrated, upstream string) {
	if s.e.memo == nil || rehydrated == "" {
		return
	}
	s.e.memo.put(memoEntry{key: memoKeyFor(s.scope, rehydrated), out: upstream, stats: Stats{ByType: map[entity.Type]int{}}})
}
