package outworld

import (
	"container/list"
	"crypto/sha256"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/davidparkercodes/outworld/pkg/dateshift"
)

const (
	maxModelDates       = 2048
	maxModelDatesTotal  = 1 << 18
	maxRealDates        = 8192
	maxRealDatesTotal   = 1 << 19
	maxDateScopes       = 1 << 14
	maxDateTombstones   = 1 << 18
	weakDateCueWords    = `on by due until till before after from since through starting ending effective dated deadline`
	knownDateSource     = "vault:date"
	knownWeakDateSource = "vault:date-weak"
)

var weakDateCues = func() map[string]bool {
	m := map[string]bool{}
	for _, w := range strings.Fields(weakDateCueWords) {
		m[w] = true
	}
	return m
}()

type dateTier struct {
	ll    *list.List
	exact map[string]*list.Element
	lower map[string]int
}

func newDateTier() *dateTier {
	return &dateTier{ll: list.New(), exact: map[string]*list.Element{}, lower: map[string]int{}}
}

func (t *dateTier) add(date string) {
	if el, ok := t.exact[date]; ok {
		t.ll.MoveToFront(el)
		return
	}
	t.exact[date] = t.ll.PushFront(date)
	t.lower[strings.ToLower(date)]++
}

func (t *dateTier) evictOldest() {
	el := t.ll.Back()
	if el == nil {
		return
	}
	date := el.Value.(string)
	t.ll.Remove(el)
	delete(t.exact, date)
	k := strings.ToLower(date)
	if t.lower[k]--; t.lower[k] <= 0 {
		delete(t.lower, k)
	}
}

func (t *dateTier) has(date string, exact bool) bool {
	if el, ok := t.exact[date]; ok {
		t.ll.MoveToFront(el)
		return true
	}
	return !exact && t.lower[strings.ToLower(date)] > 0
}

type dateScope struct {
	key        [sha256.Size]byte
	el         *list.Element
	real       *dateTier
	rehydrated *dateTier
	model      *dateTier
	saturated  bool
	wordy      bool
	gen        uint64
	days       map[string]bool
}

func (sc *dateScope) realCount() int { return sc.real.ll.Len() + sc.rehydrated.ll.Len() }

type dateBook struct {
	mu         sync.Mutex
	scopes     map[[sha256.Size]byte]*dateScope
	lru        *list.List
	realTotal  int
	modelTotal int
	tombs      map[[sha256.Size]byte]*list.Element
	tombList   *list.List
	clock      uint64
}

func newDateBook() *dateBook {
	return &dateBook{scopes: map[[sha256.Size]byte]*dateScope{}, lru: list.New(), tombs: map[[sha256.Size]byte]*list.Element{}, tombList: list.New()}
}

func scopeKey(scope string) [sha256.Size]byte { return sha256.Sum256([]byte(scope)) }

func (b *dateBook) scope(scope string, create bool) (*dateScope, bool) {
	k := scopeKey(scope)
	if sc, ok := b.scopes[k]; ok {
		b.lru.MoveToFront(sc.el)
		return sc, sc.saturated
	}
	_, tomb := b.tombs[k]
	if !create {
		return nil, tomb
	}
	sc := &dateScope{key: k, real: newDateTier(), rehydrated: newDateTier(), model: newDateTier(), saturated: tomb}
	if el, ok := b.tombs[k]; ok {
		b.tombList.Remove(el)
		delete(b.tombs, k)
	}
	sc.el = b.lru.PushFront(sc)
	b.scopes[k] = sc
	if tomb {
		b.touch(sc)
	}
	for len(b.scopes) > maxDateScopes {
		b.evictOldestScope()
	}
	return sc, sc.saturated
}

func (b *dateBook) evictOldestScope() {
	el := b.lru.Back()
	if el == nil {
		return
	}
	sc := el.Value.(*dateScope)
	b.lru.Remove(el)
	delete(b.scopes, sc.key)
	b.realTotal -= sc.realCount()
	b.modelTotal -= sc.model.ll.Len()
	if sc.realCount() == 0 && !sc.saturated {
		return
	}
	b.tombs[sc.key] = b.tombList.PushFront(sc.key)
	for b.tombList.Len() > maxDateTombstones {
		old := b.tombList.Back()
		b.tombList.Remove(old)
		delete(b.tombs, old.Value.([sha256.Size]byte))
	}
}

func (b *dateBook) touch(sc *dateScope) {
	b.clock++
	sc.gen = b.clock
}

func (b *dateBook) state(scope string) (uint64, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sc, ok := b.scopes[scopeKey(scope)]; ok {
		return sc.gen, sc.saturated
	}
	_, tomb := b.tombs[scopeKey(scope)]
	return 0, tomb
}

func (b *dateBook) generation(scope string) uint64 {
	b.mu.Lock()
	defer b.mu.Unlock()
	if sc, ok := b.scopes[scopeKey(scope)]; ok {
		return sc.gen
	}
	return 0
}

func strictMissesIn(text string, start, end int, order dateshift.Order) bool {
	for _, m := range dateshift.Find(text, order) {
		if m.Start <= start && end <= m.End {
			return false
		}
	}
	return true
}

func (b *dateBook) add(scope, date string, real bool) {
	if date == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, _ := b.scope(scope, true)
	if strings.IndexFunc(date, unicode.IsDigit) < 0 {
		sc.wordy = true
	}
	if !real {
		if sc.real.has(date, true) || b.modelTotal >= maxModelDatesTotal {
			return
		}
		n := sc.model.ll.Len()
		if !sc.model.has(date, true) {
			b.touch(sc)
		}
		sc.model.add(date)
		for sc.model.ll.Len() > maxModelDates {
			sc.model.evictOldest()
		}
		b.modelTotal += sc.model.ll.Len() - n
		return
	}
	b.addReal(sc, sc.real, date)
}

func (b *dateBook) addRehydrated(scope, date string) {
	if date == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, _ := b.scope(scope, true)
	if strings.IndexFunc(date, unicode.IsDigit) < 0 {
		sc.wordy = true
	}
	b.addReal(sc, sc.rehydrated, date)
}

func (b *dateBook) addReal(sc *dateScope, t *dateTier, date string) {
	if t.has(date, true) {
		return
	}
	if t.ll.Len() >= maxRealDates || b.realTotal >= maxRealDatesTotal {
		if !sc.saturated {
			b.touch(sc)
		}
		sc.saturated = true
		return
	}
	t.add(date)
	b.realTotal++
	b.touch(sc)
}

type dateLookup struct {
	known      bool
	rehydrated bool
	saturated  bool
}

func (b *dateBook) lookup(scope, date string, exact bool) dateLookup {
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, saturated := b.scope(scope, false)
	if sc == nil {
		return dateLookup{saturated: saturated}
	}
	reh := sc.rehydrated.has(date, true)
	return dateLookup{known: reh || sc.real.has(date, exact) || sc.model.has(date, exact), rehydrated: reh, saturated: saturated}
}

func dayKey(t time.Time) string { return t.Format("2006-01-02") }

func (b *dateBook) addDay(scope string, t time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, _ := b.scope(scope, true)
	if sc.days == nil {
		sc.days = map[string]bool{}
	}
	k := dayKey(t)
	if sc.days[k] {
		return
	}
	if len(sc.days) >= maxRealDates {
		if !sc.saturated {
			b.touch(sc)
		}
		sc.saturated = true
		return
	}
	sc.days[k] = true
	b.touch(sc)
}

func (b *dateBook) hasDay(scope string, t time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, saturated := b.scope(scope, false)
	if sc == nil {
		return saturated
	}
	return sc.saturated || sc.days[dayKey(t)]
}

func (b *dateBook) hasDayExact(scope string, t time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, _ := b.scope(scope, false)
	return sc != nil && sc.days[dayKey(t)]
}

func (b *dateBook) isReal(scope, date string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, _ := b.scope(scope, false)
	return sc != nil && (sc.real.has(date, true) || sc.rehydrated.has(date, true))
}

func (b *dateBook) worthScanning(scope, text string) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	sc, saturated := b.scope(scope, false)
	if sc == nil {
		return saturated
	}
	return sc.wordy || sc.saturated || strings.ContainsAny(text, "0123456789")
}
