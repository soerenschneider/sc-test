package selfupdate

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

type testSigner struct {
	keyNum [keyNumLen]byte
	priv   ed25519.PrivateKey
	pub    string
}

func newTestSigner(t *testing.T) *testSigner {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	s := &testSigner{priv: priv}
	if _, err := rand.Read(s.keyNum[:]); err != nil {
		t.Fatal(err)
	}
	raw := append(append([]byte(signifyAlgorithm), s.keyNum[:]...), pub...)
	s.pub = "untrusted comment: signify public key\n" + base64.StdEncoding.EncodeToString(raw) + "\n"
	return s
}

func (s *testSigner) sign(msg []byte) []byte {
	raw := append(append([]byte(signifyAlgorithm), s.keyNum[:]...), ed25519.Sign(s.priv, msg)...)
	return []byte("untrusted comment: verify with test.pub\n" + base64.StdEncoding.EncodeToString(raw) + "\n")
}

func TestVerifySignify(t *testing.T) {
	signer := newTestSigner(t)
	other := newTestSigner(t)
	msg := []byte("404b1ae1  builds/sc-linux-amd64\n")
	sig := signer.sign(msg)

	if err := verifySignify([]string{other.pub, signer.pub}, msg, sig); err != nil {
		t.Errorf("valid signature rejected: %v", err)
	}
	if err := verifySignify([]string{signer.pub}, append(msg, 'x'), sig); err == nil {
		t.Error("tampered message accepted")
	}
	if err := verifySignify([]string{other.pub}, msg, sig); err == nil || !strings.Contains(err.Error(), "unknown key") {
		t.Errorf("signature by untrusted key: got %v", err)
	}
	if err := verifySignify(nil, msg, sig); err == nil {
		t.Error("accepted signature without trusted keys")
	}
	if err := verifySignify([]string{signer.pub}, msg, []byte("garbage")); err == nil {
		t.Error("accepted malformed signature")
	}
}

func TestParseVersion(t *testing.T) {
	got, err := parseVersion(strings.NewReader("# version: v1.12.0\nabc  builds/sc-linux-amd64\n"))
	if err != nil || got != "v1.12.0" {
		t.Errorf("parseVersion() = %q, %v", got, err)
	}
	if _, err := parseVersion(strings.NewReader("abc  builds/sc-linux-amd64\n")); err == nil {
		t.Error("expected error for missing version")
	}
}

type fakeClient map[string][]byte

func (c fakeClient) Do(r *http.Request) (*http.Response, error) {
	body, ok := c[r.URL.String()]
	if !ok {
		return &http.Response{StatusCode: http.StatusNotFound, Status: "404 Not Found", Body: io.NopCloser(bytes.NewReader(nil))}, nil
	}
	return &http.Response{StatusCode: http.StatusOK, Status: "200 OK", Body: io.NopCloser(bytes.NewReader(body))}, nil
}

func TestUpdate(t *testing.T) {
	name, err := AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Skip(err)
	}

	signer := newTestSigner(t)
	verifiers := []Verifier{Signify{Keys: []string{signer.pub}}}

	newBinary := []byte("new binary")
	sum := sha256.Sum256(newBinary)
	release := func(signedVersion, tag string) (*Release, fakeClient) {
		sums := []byte(fmt.Sprintf("%s%s\n%s  builds/%s\n", versionPrefix, signedVersion, hex.EncodeToString(sum[:]), name))
		client := fakeClient{
			"https://example.com/bin": newBinary,
			"https://example.com/sum": sums,
			"https://example.com/sig": signer.sign(sums),
		}
		return &Release{TagName: tag, Assets: []Asset{
			{Name: name, BrowserDownloadURL: "https://example.com/bin"},
			{Name: checksumAsset, BrowserDownloadURL: "https://example.com/sum"},
			{Name: Signify{}.SignatureAsset(), BrowserDownloadURL: "https://example.com/sig"},
		}}, client
	}

	setup := func(t *testing.T) string {
		exe := filepath.Join(t.TempDir(), "sc")
		if err := os.WriteFile(exe, []byte("old binary"), 0o600); err != nil {
			t.Fatal(err)
		}
		return exe
	}

	t.Run("valid", func(t *testing.T) {
		exe := setup(t)
		r, client := release("v1.12.0", "v1.12.0")
		if err := Update(context.Background(), client, r, exe, verifiers); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(exe); !bytes.Equal(got, newBinary) {
			t.Errorf("binary not replaced, got %q", got)
		}
	})

	t.Run("replayed under newer tag", func(t *testing.T) {
		exe := setup(t)
		r, client := release("v1.11.0", "v1.12.0")
		if err := Update(context.Background(), client, r, exe, verifiers); err == nil || !strings.Contains(err.Error(), "not v1.12.0") {
			t.Fatalf("expected version mismatch, got %v", err)
		}
	})

	t.Run("tampered checksums", func(t *testing.T) {
		exe := setup(t)
		r, client := release("v1.12.0", "v1.12.0")
		client["https://example.com/sum"] = append(client["https://example.com/sum"], '\n')
		if err := Update(context.Background(), client, r, exe, verifiers); err == nil || !strings.Contains(err.Error(), "could not verify") {
			t.Fatalf("expected verification failure, got %v", err)
		}
		if got, _ := os.ReadFile(exe); string(got) != "old binary" {
			t.Errorf("binary was replaced despite failed verification")
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		exe := setup(t)
		r, client := release("v1.12.0", "v1.12.0")
		r.Assets = r.Assets[:2]
		if err := Update(context.Background(), client, r, exe, verifiers); err == nil || !strings.Contains(err.Error(), "no signature") {
			t.Fatalf("expected error for missing signature, got %v", err)
		}
	})

	t.Run("no verifiers", func(t *testing.T) {
		exe := setup(t)
		r, client := release("v1.12.0", "v1.12.0")
		if err := Update(context.Background(), client, r, exe, nil); err == nil {
			t.Fatal("expected error without verifiers")
		}
	})

	t.Run("falls back to next verifier", func(t *testing.T) {
		exe := setup(t)
		r, client := release("v1.12.0", "v1.12.0")
		failing := fakeVerifier{asset: "checksum.sha256.other", err: errors.New("bad signature")}
		r.Assets = append(r.Assets, Asset{Name: failing.asset, BrowserDownloadURL: "https://example.com/other"})
		client["https://example.com/other"] = []byte("sig")
		skipped := fakeVerifier{asset: "checksum.sha256.missing"}
		if err := Update(context.Background(), client, r, exe, []Verifier{skipped, failing, verifiers[0]}); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(exe); !bytes.Equal(got, newBinary) {
			t.Errorf("binary not replaced, got %q", got)
		}
	})

	t.Run("all verifiers fail", func(t *testing.T) {
		exe := setup(t)
		r, client := release("v1.12.0", "v1.12.0")
		failing := fakeVerifier{asset: Signify{}.SignatureAsset(), err: errors.New("bad signature")}
		err := Update(context.Background(), client, r, exe, []Verifier{failing})
		if err == nil || !strings.Contains(err.Error(), "fake: bad signature") {
			t.Fatalf("expected verification failure, got %v", err)
		}
	})
}

type fakeVerifier struct {
	asset string
	err   error
}

func (fakeVerifier) Name() string { return "fake" }

func (f fakeVerifier) SignatureAsset() string { return f.asset }

func (f fakeVerifier) Verify(context.Context, []byte, []byte) error { return f.err }

func TestDefaultVerifiers(t *testing.T) {
	keyless := DefaultVerifiers()[len(DefaultVerifiers())-1].(CosignKeyless)
	re := regexp.MustCompile(keyless.SubjectRegexp)
	for subject, want := range map[string]bool{
		"https://github.com/soerenschneider/sc/.github/workflows/release-container.yaml@refs/heads/main":    true,
		"https://github.com/soerenschneider/sc/.github/workflows/release-container.yaml@refs/heads/mainx":   false,
		"https://github.com/soerenschneider/sc/.github/workflows/release-container.yaml@refs/heads/feature": false,
		"https://github.com/soerenschneider/sc/.github/workflows/release-container.yaml@refs/pull/1/merge":  false,
		"https://github.com/soerenschneider/sc/.github/workflows/test.yaml@refs/heads/main":                 false,
		"https://github.com/soerenschneider/scx/.github/workflows/release-container.yaml@refs/heads/main":   false,
		"https://github.com/evil/sc/.github/workflows/release-container.yaml@refs/heads/main":               false,
	} {
		if got := re.MatchString(subject); got != want {
			t.Errorf("match(%s) = %v, want %v", subject, got, want)
		}
	}
}
