package selfupdate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/testing/ca"
)

// The cosign-key fixture was created with cosign v3.1.3:
//
//	cosign generate-key-pair
//	cosign sign-blob --key cosign.key --bundle checksum.sha256.sigstore.json checksum.sha256
//
// The key pair is a throwaway one, its private key was deleted.
func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func publicGoodRoot(t *testing.T) root.TrustedMaterial {
	t.Helper()
	trusted, err := root.NewTrustedRootFromJSON(readFixture(t, "trusted_root.json"))
	if err != nil {
		t.Fatal(err)
	}
	return trusted
}

func TestCosignKey(t *testing.T) {
	msg := readFixture(t, "cosign-key/checksum.sha256")
	sig := readFixture(t, "cosign-key/checksum.sha256.sigstore.json")
	pub := string(readFixture(t, "cosign-key/cosign.pub"))
	const otherKey = `-----BEGIN PUBLIC KEY-----
MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAE2G2Y+2tabdTV5BcGiBIx0a9fAFwr
kBbmLSGtks4L3qX6yYY0zufBnhC8Ur/iy55GhWP/9A/bY2LhC30M9+RYtw==
-----END PUBLIC KEY-----
`
	ctx := context.Background()
	v := CosignKey{Keys: []string{otherKey, pub}, TrustedMaterial: publicGoodRoot(t)}

	if err := v.Verify(ctx, msg, sig); err != nil {
		t.Errorf("valid signature rejected: %v", err)
	}
	if err := v.Verify(ctx, append(msg, 'x'), sig); err == nil {
		t.Error("tampered message accepted")
	}
	v.Keys = []string{otherKey}
	if err := v.Verify(ctx, msg, sig); err == nil {
		t.Error("signature by untrusted key accepted")
	}
	v.Keys = nil
	if err := v.Verify(ctx, msg, sig); err == nil {
		t.Error("accepted signature without trusted keys")
	}
	v.Keys = []string{pub}
	if err := v.Verify(ctx, msg, []byte("garbage")); err == nil {
		t.Error("accepted malformed bundle")
	}
	v.Keys = []string{"not a key"}
	if err := v.Verify(ctx, msg, sig); err == nil {
		t.Error("accepted invalid trusted key")
	}
}

func TestCosignKeyless(t *testing.T) {
	sigstore, err := ca.NewVirtualSigstore()
	if err != nil {
		t.Fatal(err)
	}
	const (
		subject = "https://github.com/soerenschneider/sc/.github/workflows/release-container.yaml@refs/heads/main"
		branch  = "https://github.com/soerenschneider/sc/.github/workflows/release-container.yaml@refs/heads/feature"
	)
	msg := []byte("# version: v1.2.3\nabc  builds/sc-linux-amd64\n")
	v := DefaultVerifiers()[len(DefaultVerifiers())-1].(CosignKeyless)
	// The virtual sigstore's certificates carry no SCTs.
	v.withoutSCTs = true

	signed, err := sigstore.Sign(subject, githubActionsIssuer, msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.verifyEntity(signed, msg, sigstore); err != nil {
		t.Errorf("valid signature rejected: %v", err)
	}
	if err := v.verifyEntity(signed, append(msg, 'x'), sigstore); err == nil {
		t.Error("tampered message accepted")
	}

	fromBranch, err := sigstore.Sign(branch, githubActionsIssuer, msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.verifyEntity(fromBranch, msg, sigstore); err == nil {
		t.Error("signature from branch accepted")
	}

	otherIssuer, err := sigstore.Sign(subject, "https://accounts.google.com", msg)
	if err != nil {
		t.Fatal(err)
	}
	if err := v.verifyEntity(otherIssuer, msg, sigstore); err == nil {
		t.Error("signature from other issuer accepted")
	}

	untrusted, err := ca.NewVirtualSigstore()
	if err != nil {
		t.Fatal(err)
	}
	if err := v.verifyEntity(signed, msg, untrusted); err == nil {
		t.Error("signature from untrusted sigstore instance accepted")
	}

	if err := v.Verify(context.Background(), msg, []byte("garbage")); err == nil {
		t.Error("accepted malformed bundle")
	}
	v.withoutSCTs = false
	if err := v.verifyEntity(signed, msg, sigstore); err == nil {
		t.Error("accepted certificate without SCT")
	}
	if err := (CosignKeyless{withoutSCTs: true}).verifyEntity(signed, msg, sigstore); err == nil {
		t.Error("accepted signature without trusted identity")
	}
}
