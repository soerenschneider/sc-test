package format

import (
	"crypto/x509"
	"errors"

	"github.com/soerenschneider/sc/internal/pki"
	"github.com/soerenschneider/sc/internal/vault"
)

// CsrStorage offers an interface to read/write keypair data (certificate and private key) and optional ca data.
type CsrStorage struct {
	ca   StorageImplementation
	cert StorageImplementation
	csr  StorageImplementation
}

func NewCsrStorage(cert, csr, chain StorageImplementation) (*CsrStorage, error) {
	if nil == cert {
		return nil, errors.New("empty cert storage provided")
	}

	if nil == csr {
		return nil, errors.New("empty private key storage provided")
	}

	return &CsrStorage{cert: cert, csr: csr, ca: chain}, nil
}

func (f *CsrStorage) WriteCert(certData *vault.PkiCertData) error {
	if certData.HasCaData() && f.ca != nil {
		if err := f.ca.Write(append(certData.CaData, "\n"...)); err != nil {
			return err
		}
	}

	if err := f.cert.Write(append(certData.Certificate, "\n"...)); err != nil {
		return err
	}

	return nil
}

func (f *CsrStorage) ReadCert() (*x509.Certificate, error) {
	data, err := f.cert.Read()
	if err != nil {
		return nil, err
	}

	return pki.ParseCertPem(data)
}

func (f *CsrStorage) ReadCsr() ([]byte, error) {
	return f.csr.Read()
}

func (f *CsrStorage) WriteSignature(cert *vault.PkiSignature) error {
	if cert.HasCaData() && f.ca != nil {
		if err := f.ca.Write(append(cert.CaData, "\n"...)); err != nil {
			return err
		}
	}

	if err := f.cert.Write(append(cert.Certificate, "\n"...)); err != nil {
		return err
	}

	return nil
}
