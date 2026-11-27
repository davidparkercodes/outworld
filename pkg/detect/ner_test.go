package detect

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

func TestNERDetectorNilClientAndSortedLabels(t *testing.T) {
	var bodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req nerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		if !sort.StringsAreSorted(req.Labels) {
			t.Errorf("labels not sorted: %v", req.Labels)
		}
		b, _ := json.Marshal(req)
		bodies = append(bodies, string(b))
		w.Write([]byte(`{"entities":[{"start":0,"end":4,"label":"person","score":0.9}]}`))
	}))
	defer srv.Close()
	d := &NERDetector{URL: srv.URL, Labels: DefaultNERLabels(), Threshold: 0.5}
	for i := 0; i < 5; i++ {
		spans, err := d.Detect(context.Background(), "Mary went home")
		if err != nil {
			t.Fatal(err)
		}
		if len(spans) != 1 || spans[0].Text != "Mary" || spans[0].Type != entity.Person {
			t.Fatalf("spans %+v", spans)
		}
	}
	for _, b := range bodies[1:] {
		if b != bodies[0] {
			t.Fatalf("request body not stable: %s vs %s", b, bodies[0])
		}
	}
}

func TestNERDetectorErrorNeverCarriesBody(t *testing.T) {
	input := "Wilhelmina Quarrington lives at 12 Elm Road"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req nerRequest
		json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusUnprocessableEntity)
		b, _ := json.Marshal(map[string]any{"detail": []map[string]any{{"msg": "bad", "input": req.Text}}})
		w.Write(b)
	}))
	defer srv.Close()
	d := NewNERDetector(srv.URL, 0)
	_, err := d.Detect(context.Background(), input)
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if msg != "ner sidecar: status 422" {
		t.Fatalf("error %q", msg)
	}
	for _, w := range []string{"Wilhelmina", "Quarrington", "Elm", "detail", "input"} {
		if strings.Contains(msg, w) {
			t.Fatalf("error carries body text %q: %q", w, msg)
		}
	}
}

func TestNERDetectorBadResponseHidesBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"entities":[{"start":0,"end":4,"label":"Wilhelmina"Q}]}`))
	}))
	defer srv.Close()
	_, err := NewNERDetector(srv.URL, 0).Detect(context.Background(), "Wilhelmina went home")
	if err == nil || err.Error() != "ner sidecar: bad response" {
		t.Fatalf("error %v", err)
	}
}

func TestNERDetectorNilLabelsUseDefaults(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req nerRequest
		json.NewDecoder(r.Body).Decode(&req)
		want := []string{"address", "location", "organization", "person"}
		if strings.Join(req.Labels, ",") != strings.Join(want, ",") {
			t.Errorf("labels %v", req.Labels)
		}
		w.Write([]byte(`{"entities":[{"start":0,"end":4,"label":"person","score":0.9}]}`))
	}))
	defer srv.Close()
	d := &NERDetector{URL: srv.URL, Threshold: 0.5}
	spans, err := d.Detect(context.Background(), "Mary went home")
	if err != nil || len(spans) != 1 || spans[0].Type != entity.Person {
		t.Fatalf("%+v %v", spans, err)
	}
}
