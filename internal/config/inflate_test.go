package config

import (
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
)

func TestInflateCapDefaults(t *testing.T) {
	c, err := Parse([]byte(specExample))
	if err != nil {
		t.Fatal(err)
	}
	if c.Policy.MaxInflated != detect.InflateBlobMax || c.Policy.MaxInflatedRequest != detect.InflateRequestMax {
		t.Errorf("defaults %d %d", c.Policy.MaxInflated, c.Policy.MaxInflatedRequest)
	}
	var zero Config
	if l := zero.InflateLimits(); l != (detect.InflateLimits{}) {
		t.Errorf("zero config limits %+v", l)
	}
	if b := detect.NewInflateBudgetFor(zero.InflateLimits()); b.Left != detect.InflateRequestMax || b.Blob != detect.InflateBlobMax {
		t.Errorf("zero config budget %+v", b)
	}
	p, err := c.EnginePolicy()
	if err != nil {
		t.Fatal(err)
	}
	if p.Inflate.Blob != detect.InflateBlobMax || p.Inflate.Request != detect.InflateRequestMax {
		t.Errorf("engine policy limits %+v", p.Inflate)
	}
}

func TestInflateCapsParse(t *testing.T) {
	c, err := Parse([]byte(specExample + "  max_inflated_bytes: 1048576\n  max_inflated_request_bytes: 3145728\n"))
	if err != nil {
		t.Fatal(err)
	}
	if l := c.InflateLimits(); l.Blob != 1<<20 || l.Request != 3<<20 {
		t.Errorf("parsed limits %+v", l)
	}
	p, _ := c.EnginePolicy()
	if p.Inflate.Blob != 1<<20 || p.Inflate.Request != 3<<20 {
		t.Errorf("engine policy limits %+v", p.Inflate)
	}
}
