package outworld

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/davidparkercodes/outworld/pkg/dateshift"
	"github.com/davidparkercodes/outworld/pkg/detect"
	"github.com/davidparkercodes/outworld/pkg/entity"
	"github.com/davidparkercodes/outworld/pkg/surrogate"
	"github.com/davidparkercodes/outworld/pkg/vault"
)

type Policy struct {
	Actions      map[entity.Type]entity.Action
	DateShift    bool
	DateOrder    dateshift.Order
	ShiftMinDays int
	ShiftMaxDays int
}

func DefaultPolicy() Policy {
	return Policy{
		Actions: map[entity.Type]entity.Action{
			entity.Person:    entity.Mask,
			entity.Email:     entity.Mask,
			entity.Phone:     entity.Mask,
			entity.Address:   entity.Mask,
			entity.Date:      entity.Shift,
			entity.Org:       entity.Keep,
			entity.Location:  entity.Keep,
			entity.URL:       entity.Keep,
			entity.IP:        entity.Mask,
			entity.Secret:    entity.Strip,
			entity.GovID:     entity.Strip,
			entity.Financial: entity.Strip,
		},
		DateShift:    true,
		ShiftMinDays: 30,
		ShiftMaxDays: 400,
	}
}

func (p Policy) Action(t entity.Type) entity.Action {
	a, ok := p.Actions[t]
	if !ok {
		a = DefaultPolicy().Actions[t]
	}
	if a == "" {
		return entity.Keep
	}
	if t == entity.Date && (a == entity.Shift || a == entity.Mask) && !p.DateShift {
		return entity.Keep
	}
	if t == entity.Date && a == entity.Mask {
		return entity.Shift
	}
	if t != entity.Date && a == entity.Shift {
		return entity.Mask
	}
	return a
}

type Options struct {
	Vault       *vault.Vault
	Policy      Policy
	Dictionary  *detect.Dictionary
	NER         detect.Detector
	NEROptional bool
	Extra       []detect.Detector
}

type Engine struct {
	vault       *vault.Vault
	policy      Policy
	gen         *surrogate.Generator
	detectors   []detect.Detector
	checkers    []detect.Detector
	ner         detect.Detector
	nerOptional bool
}

var ErrNoVault = errors.New("outworld: a vault is required")

func New(o Options) (*Engine, error) {
	if o.Vault == nil {
		return nil, ErrNoVault
	}
	if o.Policy.Actions == nil {
		o.Policy = DefaultPolicy()
	}
	if o.Policy.ShiftMaxDays <= 0 {
		o.Policy.ShiftMinDays, o.Policy.ShiftMaxDays = 30, 400
	}
	if o.Policy.ShiftMinDays > o.Policy.ShiftMaxDays {
		o.Policy.ShiftMinDays = o.Policy.ShiftMaxDays
	}
	e := &Engine{vault: o.Vault, policy: o.Policy, gen: surrogate.NewGenerator(o.Vault.Secret()), ner: o.NER, nerOptional: o.NEROptional}
	base := []detect.Detector{detect.NewRegexDetector(), detect.DateDetector{Order: o.Policy.DateOrder}}
	if o.Dictionary != nil {
		dd := o.Dictionary.Detector()
		base = append(base, dd)
		o.Vault.Reserve(dd.Phrases()...)
	}
	base = append(base, o.Extra...)
	e.checkers = base
	e.detectors = append([]detect.Detector(nil), base...)
	return e, nil
}

func (e *Engine) Policy() Policy { return e.policy }

func (e *Engine) Vault() *vault.Vault { return e.vault }

func (e *Engine) DateOffset(scope string) int {
	if !e.policy.DateShift {
		return 0
	}
	m := hmac.New(sha256.New, e.vault.Secret())
	fmt.Fprintf(m, "outworld-date-shift\x00%s", scope)
	v := binary.BigEndian.Uint64(m.Sum(nil))
	span := uint64(e.policy.ShiftMaxDays - e.policy.ShiftMinDays + 1)
	days := e.policy.ShiftMinDays + int(v%span)
	if (v>>63)&1 == 1 {
		days = -days
	}
	return days
}

func (e *Engine) Detect(ctx context.Context, text string) ([]entity.Span, error) {
	var all []entity.Span
	for _, d := range e.detectors {
		spans, err := d.Detect(ctx, text)
		if err != nil {
			return nil, err
		}
		all = append(all, spans...)
	}
	if e.ner != nil {
		spans, err := e.ner.Detect(ctx, text)
		if err != nil && !e.nerOptional {
			return nil, fmt.Errorf("outworld: ner detector failed (fail closed): %w", err)
		}
		all = append(all, spans...)
	}
	return all, nil
}

func (e *Engine) plan(spans []entity.Span) []entity.Span {
	kept := e.keptOrgs(spans)
	var act []entity.Span
	for _, s := range spans {
		s = trimPossessive(s)
		if s.Len() <= 0 {
			continue
		}
		if e.policy.Action(s.Type) == entity.Keep || insideKept(kept, s.Type, s.Start, s.End) {
			continue
		}
		act = append(act, s)
	}
	return detect.Resolve(act)
}

func (e *Engine) keptOrgs(spans []entity.Span) []entity.Span {
	if e.policy.Action(entity.Org) != entity.Keep {
		return nil
	}
	var out []entity.Span
	for _, s := range spans {
		if s.Type == entity.Org && s.Source != "ner" {
			out = append(out, s)
		}
	}
	return out
}

func insideKept(kept []entity.Span, t entity.Type, start, end int) bool {
	if t != entity.Person {
		return false
	}
	for _, k := range kept {
		if k.Start <= start && end <= k.End && k.End-k.Start > end-start {
			return true
		}
	}
	return false
}

func trimPossessive(s entity.Span) entity.Span {
	if s.Type != entity.Person && s.Type != entity.Org {
		return s
	}
	for _, suf := range []string{"'s", "’s", "'S"} {
		if strings.HasSuffix(s.Text, suf) && len(s.Text) > len(suf) {
			s.Text = s.Text[:len(s.Text)-len(suf)]
			s.End -= len(suf)
		}
	}
	return s
}

type insertion struct {
	start, end int
	typ        entity.Type
}

func covered(ins []insertion, start, end int) bool {
	pos := start
	for _, in := range ins {
		if in.end <= pos || in.start > pos {
			continue
		}
		pos = in.end
		if pos >= end {
			return true
		}
	}
	return pos >= end
}

func sortInsertions(ins []insertion) {
	sort.Slice(ins, func(i, j int) bool { return ins[i].start < ins[j].start })
}
