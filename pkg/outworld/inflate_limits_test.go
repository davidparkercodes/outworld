package outworld

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/detect"
)

func cleanGzipLog(n int) string {
	line := []byte("INFO request ok path=/health status=200\n")
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	w.Write(bytes.Repeat(line, n/len(line)))
	w.Close()
	return base64.StdEncoding.EncodeToString(out.Bytes())
}

func TestInflateLimitsFromPolicy(t *testing.T) {
	two, big := cleanGzipLog(2<<20), cleanGzipLog(9<<19)
	for _, tc := range []struct {
		name   string
		limits detect.InflateLimits
		v      string
		kept   bool
	}{
		{"default strips 2 MiB", detect.InflateLimits{}, two, false},
		{"4 MiB cap passes 2 MiB", detect.InflateLimits{Blob: 4 << 20, Request: 8 << 20}, two, true},
		{"1 MiB cap strips 2 MiB", detect.InflateLimits{Blob: 1 << 20, Request: 8 << 20}, two, false},
		{"4 MiB cap strips 4.5 MiB", detect.InflateLimits{Blob: 4 << 20, Request: 8 << 20}, big, false},
		{"6 MiB cap passes 4.5 MiB", detect.InflateLimits{Blob: 6 << 20, Request: 16 << 20}, big, true},
		{"request cap below blob strips", detect.InflateLimits{Blob: 4 << 20, Request: 1 << 20}, two, false},
	} {
		e := newEngine(t, func(o *Options) { o.Policy.Inflate = tc.limits })
		out, err := e.NewSession("caps").Mask(ctx, "see "+tc.v)
		if err != nil {
			t.Fatal(err)
		}
		if kept := strings.Contains(out, tc.v); kept != tc.kept {
			t.Errorf("%s: kept=%v", tc.name, kept)
		}
	}
}
