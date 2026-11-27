package detect

import (
	"context"
	"encoding/base64"
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"testing"
)

const (
	testUpperNum = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	testAlnum    = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	testStd      = testAlnum + "+/"
	testURL      = testAlnum + "-_"
	testBech32   = "QPZRY9X8GF2TVDW0S3JN54KHCE6MUA7L"
)

func pick(r *rand.Rand, n int, set string) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = set[r.Intn(len(set))]
	}
	return string(b)
}

func reviewerKeys(r *rand.Rand) [][2]string {
	return [][2]string{
		{"akia", "AKIA" + pick(r, 16, testUpperNum)},
		{"aizasy", "AIzaSy" + pick(r, 33, testAlnum)},
		{"aiza", "AIza" + pick(r, 35, testAlnum)},
		{"ghp", "ghp_" + pick(r, 36, testAlnum)},
		{"glpat", "glpat-" + pick(r, 20, testAlnum)},
		{"skant", "sk-ant-api03-" + pick(r, 93, testAlnum)},
		{"skproj", "sk-proj-" + pick(r, 48, testAlnum)},
		{"npm", "npm_" + pick(r, 36, testAlnum)},
		{"xoxb", "xoxb-" + pick(r, 12, "0123456789") + "-" + pick(r, 24, testAlnum)},
		{"age", "AGE-SECRET-KEY-1" + pick(r, 58, testBech32)},
	}
}

var reviewerPrefixes = map[string]string{"akia": "AKIA", "aizasy": "AIzaSy", "aiza": "AIza", "ghp": "ghp_", "glpat": "glpat-", "skant": "sk-ant-api03-", "skproj": "sk-proj-", "npm": "npm_", "xoxb": "xoxb-", "age": "AGE-SECRET-KEY-1"}

func qualifies(name string) bool {
	for _, k := range runVendorShapes {
		if k.prefix == reviewerPrefixes[name] {
			return true
		}
	}
	return false
}

func TestRunVendorBudgetDerivesList(t *testing.T) {
	var got []string
	for _, k := range runVendorShapes {
		got = append(got, k.name)
	}
	sort.Strings(got)
	want := []string{"age AGE-SECRET-KEY-1", "anthropic sk-ant-api03-", "aws ABIA", "aws ACCA", "aws AKIA", "aws ASIA", "github_pat_", "gitlab glpat-", "google AIzaSy", "openai sk-proj-", "slack xoxa-", "slack xoxb-", "slack xoxo-", "slack xoxp-", "slack xoxr-", "slack xoxs-", "stripe sk-live-", "stripe sk-test-"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("qualifying shapes %v", got)
	}
	for _, k := range vendorKeyShapes {
		if (k.runChance() <= runVendorBudget) != qualifiesShape(k) {
			t.Errorf("%s: list and budget disagree", k.name)
		}
		if k.runChance()*float64(4<<20*4/3) > 1e-4 && k.runChance() <= runVendorBudget {
			t.Errorf("%s: over 1e-4 expected chance hits per 4 MiB blob", k.name)
		}
	}
}

func qualifiesShape(k keyShape) bool {
	for _, q := range runVendorShapes {
		if q.name == k.name {
			return true
		}
	}
	return false
}

func keyCaught(t *testing.T, text, key string) bool {
	spans, err := NewRegexDetectorWith(testLimits).Detect(context.Background(), text)
	if err != nil {
		t.Fatal(err)
	}
	for _, sp := range spans {
		if strings.Contains(sp.Text, key) && (sp.Source == "regex:vendor_token" || sp.Source == "regex:api_key_prefixed") {
			return true
		}
	}
	return false
}

func TestVendorKeyPlainInBase64Run(t *testing.T) {
	for _, seed := range []int64{18, 118, 218} {
		r := rand.New(rand.NewSource(seed))
		for _, k := range reviewerKeys(r) {
			set := testStd
			if strings.ContainsAny(k[1], "-_") {
				set = testURL
			}
			rows := map[string]string{}
			for _, at := range []int{0, 300, 1500} {
				v := pick(r, at, set) + k[1] + pick(r, 1600, set)
				rows[fmt.Sprint(at)] = v[:len(v)/4*4]
			}
			v := pick(r, 1600, set) + k[1]
			for len(v)%4 != 0 {
				v = pick(r, 1, set) + v
			}
			rows["end"] = v
			for at, v := range rows {
				caught := keyCaught(t, "see "+v, k[1])
				edge := at == "0" || at == "end"
				switch {
				case qualifies(k[0]) || edge:
					if !caught {
						t.Errorf("%s at %s: not caught", k[0], at)
					}
				case caught:
					t.Errorf("%s at %s: residual row now caught, update the residual list", k[0], at)
				}
			}
		}
	}
}

func TestVendorKeyMidRunResidualStaysRaw(t *testing.T) {
	for _, seed := range []int64{19, 119, 219} {
		r := rand.New(rand.NewSource(seed))
		for _, k := range [][2]string{{"aiza", "AIza" + pick(r, 35, testAlnum)}, {"ghp", "ghp_" + pick(r, 36, testAlnum)}, {"npm", "npm_" + pick(r, 36, testAlnum)}} {
			v := pick(r, 1500, testURL) + k[1] + pick(r, 1600, testURL)
			v = v[:len(v)/4*4]
			if keyCaught(t, "see "+v, k[1]) {
				t.Errorf("%s mid-run caught; residual list is out of date", k[0])
			}
		}
	}
}

func TestVendorKeyShortAndLongRuns(t *testing.T) {
	for _, seed := range []int64{20, 120, 220} {
		r := rand.New(rand.NewSource(seed))
		for _, k := range reviewerKeys(r) {
			if !qualifies(k[0]) {
				continue
			}
			for _, n := range []int{1200, 1400} {
				set := testStd
				if strings.ContainsAny(k[1], "-_") {
					set = testURL
				}
				v := pick(r, n/2, set) + k[1] + pick(r, n/2, set)
				v = v[:len(v)/4*4]
				if !keyCaught(t, "see "+v, k[1]) {
					t.Errorf("%s in a %d-char run: not caught", k[0], n)
				}
			}
		}
	}
}

func TestVendorKeyGluedInDecodedBinary(t *testing.T) {
	for _, seed := range []int64{21, 121, 221} {
		r := rand.New(rand.NewSource(seed))
		for _, k := range reviewerKeys(r) {
			raw := make([]byte, 4096)
			r.Read(raw)
			ins := "Qz" + k[1] + "Xy"
			raw = append(append(append([]byte(nil), raw[:2048]...), ins...), raw[2048:]...)
			if !blobStripped(t, raw) {
				t.Errorf("%s glued to letters in decoded binary: not stripped", k[0])
			}
			found := false
			for _, run := range blobRuns(raw) {
				found = found || strings.Contains(run, k[1]) && printableRunVendorKey(run)
			}
			if !found {
				t.Errorf("seed %d %s glued to letters in decoded binary: not found as a vendor key", seed, k[0])
			}
		}
	}
}

func TestVendorWholeRunScanRandomRuns(t *testing.T) {
	hits := 0
	for seed := int64(0); seed < 200; seed++ {
		raw := randomDER(seed+7000, 48<<10)
		for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.URLEncoding} {
			hits += len(scanBlobVendorKeys("see "+enc.EncodeToString(raw), ""))
		}
	}
	if hits != 0 {
		t.Errorf("vendor scan hit %d times on random 48 KB blobs", hits)
	}
}

func TestDegenerateKeyBodyGuard(t *testing.T) {
	if !degenerateBody("AAAAAAAAAAAAAAAA") || !degenerateBody("ABABABABABABABAB") || !degenerateBody("Q7RS9AAAAAATU1VW3XY5") || degenerateBody("Q7RS9TU1VW3XY5AB") {
		t.Errorf("body guard")
	}
	zip := "see " + strings.Repeat("A", 800) + "ABIAAAAAAAAAAAAAAAAA" + strings.Repeat("A", 800)
	if got := scanBlobVendorKeys(zip, ""); len(got) != 0 {
		t.Errorf("degenerate ABIA run matched: %+v", got)
	}
}
