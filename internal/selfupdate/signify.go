package selfupdate

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

// signifyKeys holds the signify public keys release checksums may be signed
// with, in the format written by "signify -G" (comment line + base64 key).
var signifyKeys = []string{
	`untrusted comment: signify public key
RWSFxNuvQMx07H1IC6sUxJvlsdtfDlY39EdoHMG/ZpivtOmp8sJ3DMEg`,
}

// Signify verifies signatures created with "signify -S" by one of Keys.
type Signify struct {
	Keys []string
}

func (Signify) Name() string { return "signify" }

func (Signify) SignatureAsset() string { return checksumAsset + ".sig" }

func (s Signify) Verify(_ context.Context, msg, sig []byte) error {
	return verifySignify(s.Keys, msg, sig)
}

const (
	signifyAlgorithm = "Ed"
	keyNumLen        = 8
)

type signifyKey struct {
	keyNum [keyNumLen]byte
	key    ed25519.PublicKey
}

type signifySig struct {
	keyNum [keyNumLen]byte
	sig    []byte
}

// decodeSignify returns the decoded payload of a signify key or signature
// file, which consists of an "untrusted comment:" line and a base64 line.
func decodeSignify(data []byte, wantLen int) ([]byte, error) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	if !scanner.Scan() || !strings.HasPrefix(scanner.Text(), "untrusted comment: ") {
		return nil, errors.New("missing untrusted comment line")
	}
	if !scanner.Scan() {
		return nil, errors.New("missing base64 line")
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimSpace(scanner.Text()))
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}
	if len(raw) != wantLen {
		return nil, fmt.Errorf("unexpected length %d, expected %d", len(raw), wantLen)
	}
	if string(raw[:2]) != signifyAlgorithm {
		return nil, fmt.Errorf("unsupported algorithm %q", raw[:2])
	}
	return raw[2:], nil
}

func parseSignifyKey(data []byte) (*signifyKey, error) {
	raw, err := decodeSignify(data, 2+keyNumLen+ed25519.PublicKeySize)
	if err != nil {
		return nil, fmt.Errorf("invalid signify public key: %w", err)
	}
	k := &signifyKey{key: ed25519.PublicKey(raw[keyNumLen:])}
	copy(k.keyNum[:], raw[:keyNumLen])
	return k, nil
}

func parseSignifySig(data []byte) (*signifySig, error) {
	raw, err := decodeSignify(data, 2+keyNumLen+ed25519.SignatureSize)
	if err != nil {
		return nil, fmt.Errorf("invalid signify signature: %w", err)
	}
	s := &signifySig{sig: raw[keyNumLen:]}
	copy(s.keyNum[:], raw[:keyNumLen])
	return s, nil
}

// verifySignify checks that sig is a valid signify signature of msg made by
// one of keys.
func verifySignify(keys []string, msg, sig []byte) error {
	if len(keys) == 0 {
		return errors.New("no trusted signing keys configured")
	}
	s, err := parseSignifySig(sig)
	if err != nil {
		return err
	}
	for _, encoded := range keys {
		k, err := parseSignifyKey([]byte(encoded))
		if err != nil {
			return err
		}
		if k.keyNum != s.keyNum {
			continue
		}
		if !ed25519.Verify(k.key, msg, s.sig) {
			return errors.New("signature verification failed")
		}
		return nil
	}
	return fmt.Errorf("signed with unknown key %x", s.keyNum)
}
