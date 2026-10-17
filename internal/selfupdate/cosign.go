package selfupdate

import (
	"bytes"
	"context"
	"crypto"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/sigstore/sigstore/pkg/signature"
)

// cosignKeys holds the PEM encoded public keys written by
// "cosign generate-key-pair" that release checksums may be signed with.
var cosignKeys = []string{}

const (
	githubActionsIssuer = "https://token.actions.githubusercontent.com"
	sigstoreBundleAsset = checksumAsset + ".sigstore.json"
)

// CosignKey verifies Sigstore bundles created with "cosign sign-blob --key
// --bundle" by one of Keys. The signature must be logged in Rekor.
type CosignKey struct {
	Keys []string
	// TrustedMaterial replaces the Sigstore public good instance's trust
	// root, which is fetched via TUF otherwise. It is needed to verify the
	// Rekor entry.
	TrustedMaterial root.TrustedMaterial
}

func (CosignKey) Name() string { return "cosign" }

func (CosignKey) SignatureAsset() string { return sigstoreBundleAsset }

func (c CosignKey) Verify(ctx context.Context, msg, sig []byte) error {
	if len(c.Keys) == 0 {
		return errors.New("no trusted cosign keys configured")
	}
	b, err := parseBundle(sig)
	if err != nil {
		return err
	}
	trusted, err := trustedMaterial(ctx, c.TrustedMaterial)
	if err != nil {
		return err
	}

	// Bundles only carry a hint of the key, so try the keys one by one
	// instead of depending on how cosign computes the hint.
	var errs []error
	for _, encoded := range c.Keys {
		keyMaterial, err := publicKeyMaterial(encoded)
		if err != nil {
			return err
		}
		err = verifyBundle(b, msg, root.TrustedMaterialCollection{trusted, keyMaterial}, verify.WithKey(),
			verify.WithTransparencyLog(1),
			verify.WithObserverTimestamps(1),
		)
		if err == nil {
			return nil
		}
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// CosignKeyless verifies Sigstore bundles created with "cosign sign-blob
// --bundle" without a key, i.e. signed with a short-lived Fulcio certificate
// for an OIDC identity. The signature must be logged in Rekor and the
// certificate must be issued by Issuer for a subject matching SubjectRegexp.
type CosignKeyless struct {
	Issuer        string
	SubjectRegexp string
	// TrustedMaterial replaces the Sigstore public good instance's trust
	// root, which is fetched via TUF otherwise.
	TrustedMaterial root.TrustedMaterial

	// withoutSCTs disables checking the certificate transparency log, for
	// tests with certificates that do not embed timestamps of one.
	withoutSCTs bool
}

func (CosignKeyless) Name() string { return "cosign keyless" }

func (CosignKeyless) SignatureAsset() string { return sigstoreBundleAsset }

func (c CosignKeyless) Verify(ctx context.Context, msg, sig []byte) error {
	b, err := parseBundle(sig)
	if err != nil {
		return err
	}
	trusted, err := trustedMaterial(ctx, c.TrustedMaterial)
	if err != nil {
		return err
	}
	return c.verifyEntity(b, msg, trusted)
}

func (c CosignKeyless) verifyEntity(entity verify.SignedEntity, msg []byte, trusted root.TrustedMaterial) error {
	if c.Issuer == "" || c.SubjectRegexp == "" {
		return errors.New("no trusted signer identity configured")
	}
	identity, err := verify.NewShortCertificateIdentity(c.Issuer, "", "", c.SubjectRegexp)
	if err != nil {
		return err
	}
	opts := []verify.VerifierOption{verify.WithTransparencyLog(1), verify.WithObserverTimestamps(1)}
	if !c.withoutSCTs {
		opts = append(opts, verify.WithSignedCertificateTimestamps(1))
	}
	return verifyBundle(entity, msg, trusted, verify.WithCertificateIdentity(identity), opts...)
}

func parseBundle(data []byte) (*bundle.Bundle, error) {
	var b bundle.Bundle
	if err := b.UnmarshalJSON(data); err != nil {
		return nil, fmt.Errorf("invalid sigstore bundle: %w", err)
	}
	return &b, nil
}

func verifyBundle(entity verify.SignedEntity, msg []byte, trusted root.TrustedMaterial, identity verify.PolicyOption, opts ...verify.VerifierOption) error {
	verifier, err := verify.NewVerifier(trusted, opts...)
	if err != nil {
		return err
	}
	_, err = verifier.Verify(entity, verify.NewPolicy(verify.WithArtifact(bytes.NewReader(msg)), identity))
	return err
}

func publicKeyMaterial(encoded string) (root.TrustedMaterial, error) {
	block, _ := pem.Decode([]byte(encoded))
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, errors.New("invalid cosign public key: no PEM PUBLIC KEY block")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("invalid cosign public key: %w", err)
	}
	verifier, err := signature.LoadVerifier(pub, crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("invalid cosign public key: %w", err)
	}
	key := root.NewExpiringKey(verifier, time.Time{}, time.Time{})
	return root.NewTrustedPublicKeyMaterial(func(string) (root.TimeConstrainedVerifier, error) {
		return key, nil
	}), nil
}

func trustedMaterial(ctx context.Context, override root.TrustedMaterial) (root.TrustedMaterial, error) {
	if override != nil {
		return override, nil
	}
	opts := tuf.DefaultOptions().WithContext(ctx)
	if dir, err := os.UserCacheDir(); err == nil {
		opts = opts.WithCachePath(filepath.Join(dir, "sc", "sigstore-tuf"))
	}
	trusted, err := root.FetchTrustedRootWithOptions(opts)
	if err != nil {
		return nil, fmt.Errorf("could not fetch sigstore trust root: %w", err)
	}
	return trusted, nil
}
