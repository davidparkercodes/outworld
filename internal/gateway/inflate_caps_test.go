package gateway

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"
)

func capLog(n int) []byte {
	return infGz(bytes.Repeat([]byte("INFO request ok path=/health status=200\n"), n/40))
}

func TestInflateCapsFromConfig(t *testing.T) {
	small, mid, two, big := capLog(900<<10), capLog(3<<19), capLog(2<<20), capLog(9<<19)
	wide := "  max_inflated_bytes: 6291456\n  max_inflated_request_bytes: 16777216\n"
	for _, tc := range []struct {
		name   string
		policy string
		blobs  [][]byte
		clean  bool
	}{
		{"default passes 900 KiB", "", [][]byte{small}, true},
		{"default refuses 1.5 MiB", "", [][]byte{mid}, false},
		{"default request cap refuses three 900 KiB", "", [][]byte{small, small, small}, false},
		{"4 MiB cap passes 2 MiB", testInflateCaps, [][]byte{two}, true},
		{"1 MiB cap refuses 2 MiB", "  max_inflated_bytes: 1048576\n  max_inflated_request_bytes: 8388608\n", [][]byte{two}, false},
		{"6 MiB cap passes 4.5 MiB", wide, [][]byte{big}, true},
		{"4 MiB cap refuses 4.5 MiB", testInflateCaps, [][]byte{big}, false},
		{"8 MiB request passes two 1.5 MiB", testInflateCaps, [][]byte{mid, mid}, true},
		{"2 MiB request cap refuses two 1.5 MiB", "  max_inflated_bytes: 4194304\n  max_inflated_request_bytes: 2097152\n", [][]byte{mid, mid}, false},
	} {
		tr := &fnTransport{reply: echoReply}
		gw := newTestServer(t, tc.policy, tr)
		body, vs := filesBody(tc.blobs...)
		resp, out := postJSON(t, gw.URL+"/v1/messages", body, nil)
		verbatim := resp.StatusCode == http.StatusOK
		for _, v := range vs {
			verbatim = verbatim && strings.Contains(lastForwarded(tr), v)
		}
		if verbatim != tc.clean {
			t.Errorf("%s: status %d verbatim=%v %.200s", tc.name, resp.StatusCode, verbatim, out)
		}
		if !tc.clean && (resp.StatusCode != http.StatusUnprocessableEntity || !strings.Contains(string(out), "gateway:compressed_unreadable")) {
			t.Errorf("%s: refusal without the compressed reason: %.300s", tc.name, out)
		}
	}
}

var vaultGiven = strings.Fields("Maria Elena John Priya Ahmed Sofia Lucas Hannah Omar Grace")
var vaultSurnames = strings.Fields("Gonzalez Petrova Smith Raman Haddad")

func populatedVault(t *testing.T) *Server {
	t.Helper()
	srv := newTestGateway(t, "masked", "", &fnTransport{reply: echoReply})
	sess := srv.engine.NewSession("")
	var b strings.Builder
	for _, g := range vaultGiven {
		for _, s := range vaultSurnames {
			fmt.Fprintf(&b, "Yesterday %s %s called the office about the invoice.\n", g, s)
		}
	}
	if _, err := sess.Mask(t.Context(), b.String()); err != nil {
		t.Fatal(err)
	}
	held := 0
	for _, g := range vaultGiven {
		for _, s := range vaultSurnames {
			if len(srv.engine.Vault().FindReals(g+" "+s)) > 0 {
				held++
			}
		}
	}
	if held < 45 {
		t.Fatalf("only %d of 50 people in the vault", held)
	}
	return srv
}

func blobWith(r *rand.Rand, s string) string {
	raw := make([]byte, 4096)
	r.Read(raw)
	raw[2047] = 0
	copy(raw[2048:], s)
	raw[2048+len(s)] = 0
	return base64.StdEncoding.EncodeToString(raw)
}

func TestVaultNamesInBinaryRuns(t *testing.T) {
	srv := populatedVault(t)
	r := rand.New(rand.NewSource(31))
	for _, name := range []string{"Priya Raman", "Omar Haddad", "Grace Smith"} {
		found := corpusFindings(t, srv, []string{blobWith(r, " "+name+" ")}, []int{0})
		if found["vault"] == 0 {
			t.Errorf("vault-held %q in a binary run not counted: %v", name, found)
		}
	}
	for _, single := range []string{"Raman", "Priya"} {
		if found := corpusFindings(t, srv, []string{blobWith(r, " "+single+" ")}, []int{0}); found["vault"] != 0 {
			t.Errorf("single vault-held word %q counted: %v", single, found)
		}
	}
}

func TestVaultNamesRandomBlobsClean(t *testing.T) {
	srv := populatedVault(t)
	r := rand.New(rand.NewSource(32))
	total := 0
	for done := 0; done < 2000; done += 50 {
		var blobs []string
		var pos []int
		for i := 0; i < 50; i++ {
			raw := make([]byte, 4096)
			r.Read(raw)
			blobs = append(blobs, base64.StdEncoding.EncodeToString(raw))
			pos = append(pos, done+i)
		}
		total += corpusFindings(t, srv, blobs, pos)["vault"]
	}
	if total != 0 {
		t.Errorf("vault findings on 2000 random blobs: %d", total)
	}
}
