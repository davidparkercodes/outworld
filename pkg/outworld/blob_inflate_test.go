package outworld

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"math/rand"
	"strings"
	"testing"
)

func maskRows(r *rand.Rand, n int, pii bool) []byte {
	var b bytes.Buffer
	for i := 0; i < n; i++ {
		if pii && i == n/2 {
			b.WriteString("Contact: call 804-555-1234 or mail jane.doe@example.com\n")
			continue
		}
		fmt.Fprintf(&b, "build %d finished in %d ms, artefact %08x\n", i, r.Intn(9999), r.Uint32())
	}
	return b.Bytes()
}

func maskCompress(kind string, b []byte) []byte {
	var out bytes.Buffer
	switch kind {
	case "gzip":
		w := gzip.NewWriter(&out)
		w.Write(b)
		w.Close()
	case "zlib":
		w := zlib.NewWriter(&out)
		w.Write(b)
		w.Close()
	default:
		w := zip.NewWriter(&out)
		f, _ := w.Create("notes.txt")
		f.Write(b)
		w.Close()
	}
	return out.Bytes()
}

func TestMaskInflatesCompressedBlobs(t *testing.T) {
	e := newEngine(t, nil)
	s := e.NewSession("inflate")
	r := rand.New(rand.NewSource(53))
	for _, kind := range []string{"gzip", "zip", "zlib"} {
		for _, pii := range []bool{false, true} {
			v := base64.StdEncoding.EncodeToString(maskCompress(kind, maskRows(r, 120, pii)))
			for _, in := range []string{"see " + v, `{"data": "` + v + `"}`} {
				out, err := s.Mask(ctx, in)
				if err != nil {
					t.Fatal(err)
				}
				if kept := strings.Contains(out, v); kept == pii {
					t.Errorf("%s pii=%v: base64 kept=%v", kind, pii, kept)
				}
			}
		}
	}
}
