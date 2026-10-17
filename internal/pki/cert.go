package pki

import (
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strings"
	"time"
)

var (
	ErrValidationFailed = errors.New("cert validation failed")
	ErrNotFound         = errors.New("no existing item found")
)

func ParseCertPem(data []byte) (*x509.Certificate, error) {
	if data == nil {
		return nil, errors.New("empty data provided")
	}

	var der *pem.Block
	rest := data
	for {
		der, rest = pem.Decode(rest)
		if der == nil {
			return nil, errors.New("invalid pem provided")
		}

		if strings.Contains(der.Type, "PRIVATE KEY") {
			continue
		}

		cert, err := x509.ParseCertificate(der.Bytes)
		if err != nil {
			return nil, err
		}

		if !cert.IsCA {
			return cert, nil
		}
	}
}

func GetFormattedSerial(content []byte) (string, error) {
	cert, err := ParseCertPem(content)
	if err != nil {
		return "", fmt.Errorf("could not parse certificate: %v", err)
	}

	return FormatSerial(cert.SerialNumber), nil
}

func FormatSerial(i *big.Int) string {
	hex := fmt.Sprintf("%x", i)
	if len(hex)%2 == 1 {
		hex = "0" + hex
	}
	re := regexp.MustCompile("..")
	return strings.TrimRight(re.ReplaceAllString(hex, "$0:"), ":")
}

func IsCertExpired(cert x509.Certificate) bool {
	return time.Now().After(cert.NotAfter)
}

func VerifyCertAgainstCa(cert, ca *x509.Certificate) error {
	if cert == nil || ca == nil {
		return errors.New("empty cert(s) supplied")
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(ca)

	verifyOptions := x509.VerifyOptions{
		Roots: certPool,
	}
	_, err := cert.Verify(verifyOptions)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrValidationFailed, err)
	}

	return nil
}

func MustRenewCert(cert *x509.Certificate, minDurationLeft time.Duration, minPercentLeft float32) (bool, error) {
	if cert == nil {
		return true, errors.New("empty certificate provided")
	}

	percentage := GetCertRemainingLifetimePercent(*cert)
	return percentage <= minPercentLeft || time.Until(cert.NotAfter) < minDurationLeft, nil
}

func GetCertRemainingLifetimePercent(cert x509.Certificate) float32 {
	from := cert.NotBefore
	expiry := cert.NotAfter

	secondsTotal := expiry.Sub(from).Seconds()
	durationUntilExpiration := time.Until(expiry)

	return float32(math.Max(0, durationUntilExpiration.Seconds()*100./secondsTotal))
}
