package detect

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/davidparkercodes/outworld/pkg/entity"
)

var sourceWords = strings.Fields(`config request handler parse value result cache index buffer token record batch window offset limit count total retry status payload header session client server worker queue event stream reader writer`)

func pySource(r *rand.Rand, i int) []byte {
	var b strings.Builder
	w := func() string { return sourceWords[r.Intn(len(sourceWords))] }
	fmt.Fprintf(&b, "import os\nimport json\nfrom typing import Any\n\n\nclass %s%sHandler:\n", strings.Title(w()), strings.Title(w()))
	for f := 0; f < 2; f++ {
		fmt.Fprintf(&b, "    def %s_%s(self, %s: int = %d) -> dict[str, Any]:\n", w(), w(), w(), r.Intn(500))
		fmt.Fprintf(&b, "        %s = self.%s.get(%q, %d)\n", w(), w(), w(), r.Intn(10000))
		fmt.Fprintf(&b, "        if %s > %d:\n            return {%q: %s, %q: %d}\n", w(), r.Intn(100), w(), w(), w(), r.Intn(65536))
		fmt.Fprintf(&b, "        return json.loads(os.environ.get(%q, \"{}\"))\n\n", strings.ToUpper(w()+"_"+w()))
	}
	fmt.Fprintf(&b, "# module %d\n", i)
	return []byte(b.String())
}

func tinyFiles(n int) []zipFile {
	fs := make([]zipFile, n)
	for i := range fs {
		fs[i] = zipFile{fmt.Sprintf("f%d", i), []byte(fmt.Sprintf("ok %d\n", i))}
	}
	return fs
}

func sourceFiles(r *rand.Rand, n int) []zipFile {
	fs := make([]zipFile, n)
	for i := range fs {
		fs[i] = zipFile{fmt.Sprintf("pkg/mod%d/%s.py", i/50, sourceWords[r.Intn(len(sourceWords))]), pySource(r, i)}
	}
	return fs
}

func archiveShapes(seed int64, n int) map[string][]byte {
	r := rand.New(rand.NewSource(seed))
	src := sourceFiles(r, n)
	var logs bytes.Buffer
	for i := 0; i < n*4; i++ {
		fmt.Fprintf(&logs, "2026-11-24T10:%02d:%02dZ INFO worker=%d request ok path=/v1/items status=200 ms=%d\n", i/60%60, i%60, r.Intn(16), r.Intn(900))
	}
	return map[string][]byte{
		"zip tiny files":   zipOf(zip.Deflate, tinyFiles(n)...),
		"zip source files": zipOf(zip.Deflate, src...),
		"tar.gz source":    gz(tarOf(src...)),
		"gzip logs":        gz(logs.Bytes()),
	}
}

func TestCleanArchivesPassUnchanged(t *testing.T) {
	total, stripped := 0, 0
	for _, n := range []int{10, 50, 200, 1000, 4096} {
		seeds := 4
		if n >= 1000 {
			seeds = 2
		}
		for seed := 0; seed < seeds; seed++ {
			for name, raw := range archiveShapes(int64(n*100+seed), n) {
				if len(raw) < cleanBlobMin {
					continue
				}
				total++
				v := base64.StdEncoding.EncodeToString(raw)
				got := secretSpans(t, "see "+v)
				if overCap := inflatedSize(raw) > testLimits.Blob; overCap {
					if len(got) != 1 || got[0].Text != v {
						t.Errorf("%s n=%d seed %d: over the inflate cap but not stripped whole", name, n, seed)
					}
					continue
				}
				if len(got) != 0 {
					stripped++
					t.Errorf("%s n=%d seed %d: clean archive flagged by %s at %d", name, n, seed, got[0].Source, got[0].Start)
				}
			}
		}
	}
	if total < 30 {
		t.Fatalf("only %d archives generated", total)
	}
	t.Logf("clean archives: %d, stripped %d", total, stripped)
}

func TestZip4096TinyFilesFixture(t *testing.T) {
	raw := zipOf(zip.Deflate, tinyFiles(4096)...)
	v := base64.StdEncoding.EncodeToString(raw)
	if len(v) < 300000 {
		t.Fatalf("fixture only %d chars", len(v))
	}
	if got := detectAll(t, "see "+v); len(got) != 0 {
		t.Errorf("zip of 4096 tiny files flagged: %s %q", got[0].Source, got[0].Text[:min(40, len(got[0].Text))])
	}
}

func TestNoTextRuleInsideJudgedBlob(t *testing.T) {
	raw := zipOf(zip.Deflate, tinyFiles(4096)...)
	v := base64.StdEncoding.EncodeToString(raw)
	text := "see " + v
	runs := blobRunSpans(text)
	if len(runs) != 1 || runs[0] != [2]int{4, len(text)} {
		t.Fatalf("blob not one run: %v", runs)
	}
	inner := strings.Index(v, "EAA")
	if inner < 0 {
		t.Skip("fixture has no vendor-shaped substring")
	}
	p := &pattern{}
	if got := p.outsideBlobs([]entity.Span{{Start: 4 + inner, End: 4 + inner + 70}}, runs); len(got) != 0 {
		t.Errorf("span inside a judged blob survived")
	}
}

func archiveSeeds(t *testing.T) int {
	if v := os.Getenv("OUTWORLD_ARCHIVE_SEEDS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	if testing.Short() {
		return 40
	}
	return 160
}

func validatorArchive(seed int) ([]byte, string) {
	r := rand.New(rand.NewSource(int64(1000 + seed)))
	n := []int{3, 10, 40, 150, 600, 2000}[r.Intn(6)]
	gens := map[string]func(int) []byte{
		"src": func(i int) []byte {
			return bytes.Repeat([]byte(fmt.Sprintf("def f%d(x):\n    return x * %d + %d\n\n", i, r.Intn(100), r.Intn(1000))), 1+r.Intn(30))
		},
		"tiny": func(i int) []byte { return []byte(fmt.Sprintf("ok %d\n", i)) },
		"json": func(i int) []byte {
			return bytes.Repeat([]byte(fmt.Sprintf(`{"id": %d, "name": "item%d", "tags": ["a","b"], "v": %d}`+"\n", i, i, r.Intn(1<<20))), 1+r.Intn(10))
		},
		"bin": func(int) []byte {
			b := make([]byte, 16+r.Intn(1985))
			r.Read(b)
			return b
		},
	}
	kinds := []string{"src", "tiny", "json", "bin"}
	kind := kinds[r.Intn(len(kinds))]
	cont := []string{"zip", "zipstored", "tgz", "gzip", "txz", "tzst", "7z"}[r.Intn(7)]
	files := make([]zipFile, n)
	for i := range files {
		files[i] = zipFile{fmt.Sprintf("dir%d/f%d.dat", i%7, i), gens[kind](i)}
	}
	var raw []byte
	switch cont {
	case "zip":
		raw = zipOf(zip.Deflate, files...)
	case "zipstored":
		raw = zipOf(zip.Store, files...)
	case "tgz":
		raw = gz(tarOf(files...))
	case "txz":
		raw = append([]byte{0xfd, '7', 'z', 'X', 'Z', 0}, gz(tarOf(files...))...)
	case "tzst":
		raw = append([]byte{0x28, 0xb5, 0x2f, 0xfd}, gz(tarOf(files...))...)
	case "7z":
		raw = append([]byte{'7', 'z', 0xbc, 0xaf, 0x27, 0x1c}, gz(tarOf(files...))...)
	default:
		var all bytes.Buffer
		for _, f := range files {
			all.Write(f.data)
		}
		raw = gz(all.Bytes())
	}
	return raw, cont + "/" + kind
}

func TestValidatorArchiveGenerator(t *testing.T) {
	total, stripped := 0, 0
	kinds := map[string][2]int{}
	for seed := 0; seed < archiveSeeds(t); seed++ {
		raw, kind := validatorArchive(seed)
		if len(raw) < cleanBlobMin || len(raw) > 3_000_000 {
			continue
		}
		total++
		k := kinds[kind]
		k[0]++
		v := base64.StdEncoding.EncodeToString(raw)
		got := secretSpans(t, "see "+v)
		sealed := strings.HasPrefix(kind, "txz") || strings.HasPrefix(kind, "tzst") || strings.HasPrefix(kind, "7z")
		switch {
		case sealed && (len(got) != 1 || got[0].Text != v):
			t.Errorf("seed %d %s: sealed format not refused", seed, kind)
		case sealed:
			k[1]++
		case len(got) != 0:
			stripped++
			k[1]++
			t.Errorf("seed %d %s (%d bytes): clean archive flagged by %s", seed, kind, len(raw), got[0].Source)
		}
		kinds[kind] = k
	}
	t.Logf("validator archives: %d, stripped %d, by kind %v", total, stripped, kinds)
}

func inflatedSize(raw []byte) int {
	if r, err := gzip.NewReader(bytes.NewReader(raw)); err == nil {
		n, _ := io.Copy(io.Discard, r)
		return int(n)
	}
	return 0
}
