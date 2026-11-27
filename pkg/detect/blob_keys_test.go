package detect

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/base64"
	"math"
	"math/big"
	mrand "math/rand"
	"strings"
	"testing"
	"time"
)

func padTo(b []byte, n int) []byte {
	if len(b) >= n {
		return b
	}
	tail := make([]byte, n-len(b))
	mrand.New(mrand.NewSource(int64(len(b)))).Read(tail)
	return append(append([]byte(nil), b...), tail...)
}

func mustDER(b []byte, err error) []byte {
	if err != nil {
		panic(err)
	}
	return b
}

func randomDER(seed int64, n int) []byte {
	b := make([]byte, n)
	mrand.New(mrand.NewSource(seed)).Read(b)
	return b
}

func encryptedKeyDER(oid asn1.ObjectIdentifier, n int) []byte {
	params := mustDER(asn1.Marshal(struct {
		Salt []byte
		Iter int
	}{randomDER(3, 8), 2048}))
	return mustDER(asn1.Marshal(struct {
		Alg struct {
			Algorithm  asn1.ObjectIdentifier
			Parameters asn1.RawValue
		}
		Data []byte
	}{struct {
		Algorithm  asn1.ObjectIdentifier
		Parameters asn1.RawValue
	}{oid, asn1.RawValue{FullBytes: params}}, randomDER(4, n)}))
}

func dsaShapedDER() []byte {
	n := func(seed int64, size int) *big.Int {
		b := randomDER(seed, size)
		b[0] |= 0x80
		return new(big.Int).SetBytes(b)
	}
	return mustDER(asn1.Marshal(struct {
		V             int
		P, Q, G, Y, X *big.Int
	}{0, n(11, 384), n(12, 32), n(13, 384), n(14, 384), n(15, 32)}))
}

type keyFixtures struct {
	keys, public map[string][]byte
}

func buildKeyFixtures(t *testing.T) keyFixtures {
	t.Helper()
	rsa2048, _ := rsa.GenerateKey(rand.Reader, 2048)
	rsa4096, _ := rsa.GenerateKey(rand.Reader, 4096)
	ec, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	edPub, ed, _ := ed25519.GenerateKey(rand.Reader)
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(7), Subject: pkix.Name{CommonName: "build.example.test"}, NotBefore: time.Unix(1700000000, 0), NotAfter: time.Unix(1900000000, 0)}
	cert2048 := mustDER(x509.CreateCertificate(rand.Reader, tmpl, tmpl, &rsa2048.PublicKey, rsa2048))
	cert4096 := mustDER(x509.CreateCertificate(rand.Reader, tmpl, tmpl, &rsa4096.PublicKey, rsa4096))
	csr := mustDER(x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: "build.example.test"}}, rsa2048))
	signed := mustDER(asn1.Marshal(struct {
		Version int
		Digests asn1.RawValue
		Content struct{ Type asn1.ObjectIdentifier }
		Certs   asn1.RawValue `asn1:"tag:0"`
		Signers asn1.RawValue
	}{1, asn1.RawValue{Tag: asn1.TagSet, IsCompound: true}, struct{ Type asn1.ObjectIdentifier }{oidPKCS7Data}, asn1.RawValue{Class: asn1.ClassContextSpecific, Tag: 0, IsCompound: true, Bytes: cert2048}, asn1.RawValue{Tag: asn1.TagSet, IsCompound: true}}))
	pkcs7 := mustDER(asn1.Marshal(struct {
		Type    asn1.ObjectIdentifier
		Content asn1.RawValue `asn1:"explicit,tag:0"`
	}{oidSignedData, asn1.RawValue{FullBytes: signed}}))
	ssh := padTo(append([]byte(nil), opensshKeyMagic...), 1600)
	return keyFixtures{
		keys: map[string][]byte{
			"rsa 2048 pkcs1":  x509.MarshalPKCS1PrivateKey(rsa2048),
			"rsa 4096 pkcs1":  x509.MarshalPKCS1PrivateKey(rsa4096),
			"rsa 2048 pkcs8":  mustDER(x509.MarshalPKCS8PrivateKey(rsa2048)),
			"ec p-256 sec1":   mustDER(x509.MarshalECPrivateKey(ec)),
			"ec p-256 pkcs8":  mustDER(x509.MarshalPKCS8PrivateKey(ec)),
			"ed25519 pkcs8":   mustDER(x509.MarshalPKCS8PrivateKey(ed)),
			"pkcs8 pbes2":     encryptedKeyDER(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 13}, 1300),
			"pkcs8 pbes1 md5": encryptedKeyDER(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 3}, 1300),
			"pkcs8 pbes1 sha": encryptedKeyDER(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 5, 10}, 1300),
			"pkcs8 pkcs12":    encryptedKeyDER(asn1.ObjectIdentifier{1, 2, 840, 113549, 1, 12, 1, 3}, 1300),
			"dsa 3072":        dsaShapedDER(),
			"openssh-key-v1":  ssh,
		},
		public: map[string][]byte{
			"cert rsa 2048": cert2048,
			"cert rsa 4096": cert4096,
			"csr rsa 2048":  csr,
			"pkcs7 signed":  pkcs7,
			"spki rsa 2048": mustDER(x509.MarshalPKIXPublicKey(&rsa2048.PublicKey)),
			"spki rsa 4096": mustDER(x509.MarshalPKIXPublicKey(&rsa4096.PublicKey)),
			"spki ec p-256": mustDER(x509.MarshalPKIXPublicKey(&ec.PublicKey)),
			"spki ed25519":  mustDER(x509.MarshalPKIXPublicKey(edPub)),
		},
	}
}

func keyForms(name string, der []byte, seed int64) map[string][]byte {
	pad := func(b []byte) []byte {
		out := append(randomDER(seed*7919+int64(len(b)), 2048), b...)
		return append(out, randomDER(seed*7919+int64(len(b))+1, 2048)...)
	}
	cut := der[:len(der)*7/10]
	flip := append([]byte(nil), der...)
	at := 1
	if strings.HasPrefix(name, "openssh") {
		at = 3
	}
	flip[at] ^= 0x5a
	return map[string][]byte{"standalone": der, "in 4 KB": pad(der), "cut 70%": cut, "cut 70% in 4 KB": pad(cut), "header byte flipped": flip}
}

func blobStripped(t *testing.T, raw []byte) bool {
	t.Helper()
	v := base64.StdEncoding.EncodeToString(raw)
	for _, sp := range secretSpans(t, "see "+v) {
		if sp.Text == v {
			return true
		}
	}
	return false
}

func TestKeyMaterialMatrix(t *testing.T) {
	f := buildKeyFixtures(t)
	for _, seed := range []int64{1, 2, 3} {
		for name, der := range f.keys {
			for form, raw := range keyForms(name, der, seed) {
				if !keyMaterial(raw) {
					t.Errorf("seed %d %s %s: not recognised as key material", seed, name, form)
				}
				if !blobStripped(t, raw) {
					t.Errorf("seed %d %s %s (%d bytes): not stripped", seed, name, form, len(raw))
				}
			}
		}
		for name, der := range f.public {
			forms := keyForms(name, der, seed)
			delete(forms, "header byte flipped")
			for form, raw := range forms {
				if keyMaterial(raw) {
					t.Errorf("seed %d %s %s: counted as key material", seed, name, form)
				}
				if blobStripped(t, raw) {
					t.Errorf("seed %d %s %s (%d bytes): clean public structure stripped", seed, name, form, len(raw))
				}
			}
		}
	}
}

func TestDamagedRSAKeyTails(t *testing.T) {
	k, _ := rsa.GenerateKey(rand.Reader, 2048)
	der := x509.MarshalPKCS1PrivateKey(k)
	flip := append([]byte(nil), der...)
	flip[len(flip)-1] ^= 0xff
	for name, raw := range map[string][]byte{"drop last byte": der[:len(der)-1], "drop last 64": der[:len(der)-64], "flip last byte": flip} {
		if !keyMaterial(raw) || !blobStripped(t, raw) {
			t.Errorf("rsa 2048 pkcs1 %s: not stripped", name)
		}
	}
}

func TestLooseDERPrefixNotKeyMaterial(t *testing.T) {
	for name, raw := range map[string][]byte{
		"30 82 02 01 00 02 82 01 01 then random": padTo([]byte{0x30, 0x82, 0x06, 0xa0, 0x02, 0x01, 0x00, 0x02, 0x82, 0x01, 0x01}, 2000),
		"pkcs12 version then random":             padTo([]byte{0x30, 0x82, 0x06, 0xa0, 0x02, 0x01, 0x03}, 2000),
		"02 01 00 30 then random":                append(randomDER(21, 1000), append([]byte{0x02, 0x01, 0x00, 0x30, 0x0d}, randomDER(22, 1000)...)...),
	} {
		if keyMaterial(raw) {
			t.Errorf("%s counted as key material", name)
		}
	}
}

func TestKeyAnchorChance(t *testing.T) {
	const positions = 4 << 20 * 4 / 3
	for _, a := range keyAnchors {
		p := math.Pow(256, -float64(a.fixed))
		if p*positions > 1e-4 {
			t.Errorf("%s: %.2e expected chance hits per 4 MiB blob", a.name, p*positions)
		}
	}
	hits := 0
	for seed := int64(0); seed < 400; seed++ {
		if keyMaterial(randomDER(seed+9000, 48<<10)) {
			hits++
		}
	}
	if hits != 0 {
		t.Errorf("key anchors matched %d of 400 random 48 KB blobs", hits)
	}
}
