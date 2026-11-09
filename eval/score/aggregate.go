package score

import (
	"math"
	"regexp"
	"sort"
)

type Rate struct {
	N    int     `json:"n"`
	Hits int     `json:"hits"`
	Rate float64 `json:"rate"`
}

func (r *Rate) add(hit bool) {
	r.N++
	if hit {
		r.Hits++
	}
	r.Rate = float64(r.Hits) / float64(r.N)
}

type Latency struct {
	N          int     `json:"n"`
	GatewayP50 float64 `json:"gateway_p50_ms"`
	GatewayP95 float64 `json:"gateway_p95_ms"`
	DirectP50  float64 `json:"direct_p50_ms"`
	DirectP95  float64 `json:"direct_p95_ms"`
	OverP50    float64 `json:"overhead_p50_ms"`
	OverP95    float64 `json:"overhead_p95_ms"`
}

type Leak struct {
	Item    string `json:"item"`
	Mode    string `json:"mode"`
	Type    string `json:"type"`
	Value   string `json:"value"`
	Variant string `json:"variant,omitempty"`
	Via     string `json:"via"`
}

type Summary struct {
	Requests        int                `json:"requests"`
	Errors          int                `json:"errors"`
	ErrorRate       float64            `json:"error_rate"`
	ErrorSamples    []string           `json:"error_samples,omitempty"`
	ErrorKinds      map[string]int     `json:"error_kinds,omitempty"`
	Leak            Rate               `json:"leak"`
	LeakAnswered    Rate               `json:"leak_answered_only"`
	LeakByType      map[string]*Rate   `json:"leak_by_type"`
	LeakByMode      map[string]*Rate   `json:"leak_by_mode"`
	LeakByVia       map[string]int     `json:"leak_by_via"`
	PersonByVariant map[string]*Rate   `json:"person_leak_by_variant"`
	PersonByDict    map[string]*Rate   `json:"person_leak_by_dictionary"`
	Restore         Rate               `json:"restore"`
	RestoreByType   map[string]*Rate   `json:"restore_by_type"`
	DateReformatted int                `json:"date_restored_reformatted"`
	OverMask        Rate               `json:"over_mask"`
	OverMaskByType  map[string]*Rate   `json:"over_mask_by_type"`
	QuasiAltered    Rate               `json:"quasi_altered"`
	UnknownReported int                `json:"unknown_surrogates_reported"`
	ResidueItems    int                `json:"items_with_surrogate_residue"`
	ResidueTokens   int                `json:"surrogate_residue_tokens"`
	ResidueSamples  []string           `json:"surrogate_residue_samples,omitempty"`
	Placeholders    []string           `json:"placeholder_tokens,omitempty"`
	UpstreamCalls   int                `json:"upstream_calls"`
	Latency         map[string]Latency `json:"latency"`
	Leaks           []Leak             `json:"leaks"`
}

func Aggregate(results []ItemResult) Summary {
	s := Summary{
		LeakByType: map[string]*Rate{}, LeakByMode: map[string]*Rate{}, LeakByVia: map[string]int{},
		PersonByVariant: map[string]*Rate{}, PersonByDict: map[string]*Rate{},
		RestoreByType: map[string]*Rate{}, OverMaskByType: map[string]*Rate{}, Latency: map[string]Latency{},
		ErrorKinds: map[string]int{},
	}
	get := func(m map[string]*Rate, k string) *Rate {
		if m[k] == nil {
			m[k] = &Rate{}
		}
		return m[k]
	}
	addLeak := func(r ItemResult, e EntityResult) {
		s.Leak.add(e.Leaked)
		get(s.LeakByType, e.Type).add(e.Leaked)
		get(s.LeakByMode, r.Mode).add(e.Leaked)
		if e.Leaked {
			s.LeakByVia[e.LeakVia]++
			s.Leaks = append(s.Leaks, Leak{Item: r.ID, Mode: r.Mode, Type: e.Type, Value: e.Value, Variant: e.Variant, Via: e.LeakVia})
		}
	}
	lat := map[string][3][]float64{}
	residue := map[string]bool{}
	df := map[string]int{}
	for _, r := range results {
		s.Requests++
		s.UpstreamCalls += r.UpstreamCalls
		if r.Err != "" {
			s.Errors++
			kind := ErrorKind(r.Err)
			s.ErrorKinds[kind]++
			if s.ErrorKinds[kind] <= 2 && len(s.ErrorSamples) < 8 {
				s.ErrorSamples = append(s.ErrorSamples, r.ID+" ["+r.Mode+"]: "+r.Err)
			}
			for _, e := range r.Entities {
				if e.Action == ActionMask || e.Action == ActionShift || e.Action == ActionStrip {
					addLeak(r, e)
				}
			}
			continue
		}
		s.UnknownReported += r.Unknown
		for _, t := range r.Residue {
			df[t]++
		}
		l := lat[r.Mode]
		l[0] = append(l[0], r.LatencyMS)
		l[1] = append(l[1], r.DirectMS)
		l[2] = append(l[2], r.LatencyMS-r.DirectMS)
		lat[r.Mode] = l
		for _, e := range r.Entities {
			switch e.Action {
			case ActionMask, ActionShift, ActionStrip:
				s.LeakAnswered.add(e.Leaked)
				addLeak(r, e)
				if e.Type == "person" {
					v := e.Variant
					if v == "" {
						v = "other"
					}
					get(s.PersonByVariant, v).add(e.Leaked)
					d := "not_in_dictionary"
					if e.InDict {
						d = "in_dictionary"
					}
					get(s.PersonByDict, d).add(e.Leaked)
				}
				if e.Restorable {
					s.Restore.add(e.Restored)
					get(s.RestoreByType, e.Type).add(e.Restored)
					if e.Reformat {
						s.DateReformatted++
					}
				}
			case ActionKeep:
				s.OverMask.add(e.Altered)
				get(s.OverMaskByType, e.Type).add(e.Altered)
			case "report":
				s.QuasiAltered.add(e.Altered)
			}
		}
	}
	if s.Requests > 0 {
		s.ErrorRate = float64(s.Errors) / float64(s.Requests)
	}
	placeholder := map[string]bool{}
	for t, n := range df {
		if n >= 5 && float64(n) > 0.1*float64(s.Requests-s.Errors) {
			placeholder[t] = true
			s.Placeholders = append(s.Placeholders, t)
		}
	}
	sort.Strings(s.Placeholders)
	for _, r := range results {
		k := 0
		for _, t := range r.Residue {
			if !placeholder[t] {
				k++
				residue[t] = true
			}
		}
		if k > 0 {
			s.ResidueItems++
			s.ResidueTokens += k
		}
	}
	for t := range residue {
		s.ResidueSamples = append(s.ResidueSamples, t)
	}
	sort.Strings(s.ResidueSamples)
	if len(s.ResidueSamples) > 25 {
		s.ResidueSamples = s.ResidueSamples[:25]
	}
	for mode, l := range lat {
		s.Latency[mode] = Latency{
			N:          len(l[0]),
			GatewayP50: Percentile(l[0], 50), GatewayP95: Percentile(l[0], 95),
			DirectP50: Percentile(l[1], 50), DirectP95: Percentile(l[1], 95),
			OverP50: Percentile(l[2], 50), OverP95: Percentile(l[2], 95),
		}
	}
	return s
}

func Percentile(xs []float64, p float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	c := append([]float64(nil), xs...)
	sort.Float64s(c)
	k := int(math.Ceil(p/100*float64(len(c)))) - 1
	if k < 0 {
		k = 0
	}
	if k >= len(c) {
		k = len(c) - 1
	}
	return math.Round(c[k]*100) / 100
}

var reErrCode = regexp.MustCompile(`"code"\s*:\s*"([^"]+)"`)
var reHTTP = regexp.MustCompile(`^HTTP (\d+)`)

func ErrorKind(err string) string {
	if m := reErrCode.FindStringSubmatch(err); m != nil {
		return m[1]
	}
	if m := reHTTP.FindStringSubmatch(err); m != nil {
		return "http_" + m[1]
	}
	return "transport_or_protocol"
}
