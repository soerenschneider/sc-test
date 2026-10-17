package vault

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type SshSignatureRequest struct {
	PublicKey  string
	Ttl        string
	Principals []string
	Extensions map[string]string

	VaultRole string
}

// SshRole holds the parts of a role of the SSH secret engine that are relevant for signing user keys.
type SshRole struct {
	Name                  string
	KeyType               string
	AllowUserCertificates bool
	DefaultUser           string
	AllowedUsers          []string
	Ttl                   time.Duration
	MaxTtl                time.Duration
}

// CanSignUserKeys returns whether the role can be used to sign user keys.
func (r *SshRole) CanSignUserKeys() bool {
	return r.KeyType == "ca" && r.AllowUserCertificates
}

// PkiRole holds the parts of a role of the PKI secret engine that are relevant for issuing certificates.
type PkiRole struct {
	Name             string
	AllowedDomains   []string
	AllowBareDomains bool
	AllowSubdomains  bool
	AllowLocalhost   bool
	AllowAnyName     bool
	AllowGlobDomains bool
	Ttl              time.Duration
	MaxTtl           time.Duration
}

type PkiSignatureArgs struct {
	CommonName string
	Ttl        string
	IpSans     []string
	AltNames   []string
}

type PkiIssueArgs struct {
	CommonName string
	Ttl        string
	IpSans     []string
	AltNames   []string
	Role       string
}

type PkiCertData struct {
	PrivateKey  []byte //#nosec:G117
	Certificate []byte
	CaData      []byte
	Csr         []byte
}

func (cert *PkiCertData) AsContainer() string {
	var buffer strings.Builder

	if cert.HasCaData() {
		buffer.Write(cert.CaData)
		buffer.Write([]byte("\n"))
	}

	buffer.Write(cert.Certificate)
	buffer.Write([]byte("\n"))

	if cert.HasPrivateKey() {
		buffer.Write(cert.PrivateKey)
		buffer.Write([]byte("\n"))
	}

	return buffer.String()
}

func (cert *PkiCertData) HasPrivateKey() bool {
	return len(cert.PrivateKey) > 0
}

func (cert *PkiCertData) HasCertificate() bool {
	return len(cert.Certificate) > 0
}

func (cert *PkiCertData) HasCaData() bool {
	return len(cert.CaData) > 0
}

type PkiSignature struct {
	Certificate []byte
	CaData      []byte
	Serial      string
}

func (cert *PkiSignature) HasCaData() bool {
	return len(cert.CaData) > 0
}

type Kv2SyncConfig struct {
	SecretPath    string                 `yaml:"secret_path" validate:"required"`
	DestUri       string                 `yaml:"dest_uri" validate:"required,filepath"`
	Formatter     string                 `yaml:"formatter" validate:"oneof=env template json yaml"`
	FormatterArgs map[string]interface{} `yaml:"formatter_args"`
}

type StorageImplementation interface {
	Read() ([]byte, error)
	CanRead() error
	Write([]byte) error
}

// Secret is the storage-neutral view of a versioned KV secret.
type Secret struct {
	Data      map[string]string
	Version   int
	UpdatedAt time.Time
}

// SecretMetadata is the storage-neutral view of a secret's version history.
type SecretMetadata struct {
	CurrentVersion int
	OldestVersion  int
	Versions       []VersionInfo // newest first
}

// VersionInfo describes one historical version. DeletedAt is zero if the
// version is still readable; Destroyed is true if the version's data has
// been permanently removed.
type VersionInfo struct {
	Version   int
	CreatedAt time.Time
	DeletedAt time.Time
	Destroyed bool
}

// Readable reports whether the version's data can still be fetched.
func (v VersionInfo) Readable() bool {
	return !v.Destroyed && v.DeletedAt.IsZero()
}

// Entry is one item returned by SecretStore.List.
type Entry struct {
	Name  string
	IsDir bool
}

// ErrorKind classifies VaultError so the UI can decide how to react.
type ErrorKind int

const (
	ErrKindUnknown ErrorKind = iota
	ErrKindNotFound
	ErrKindPermissionDenied
	ErrKindCASConflict
	ErrKindRateLimited
	ErrKindServerError
	ErrKindBadRequest
	ErrKindTimeout
	ErrKindNetwork
)

func (k ErrorKind) String() string {
	switch k {
	case ErrKindNotFound:
		return "not-found"
	case ErrKindPermissionDenied:
		return "permission-denied"
	case ErrKindCASConflict:
		return "cas-conflict"
	case ErrKindRateLimited:
		return "rate-limited"
	case ErrKindServerError:
		return "server-error"
	case ErrKindBadRequest:
		return "bad-request"
	case ErrKindTimeout:
		return "timeout"
	case ErrKindNetwork:
		return "network"
	default:
		return "unknown"
	}
}

// VaultError is the typed error every SecretStore implementation must return.
type VaultError struct {
	Kind    ErrorKind
	Op      string // "get", "get-version", "get-metadata", "put", "delete", "list"
	Message string // human-readable detail (safe to display; must not contain secret values)
	Cause   error  // underlying error, if any
}

func (e *VaultError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s: %s", e.Op, e.Kind, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Op, e.Kind)
}

func (e *VaultError) Unwrap() error { return e.Cause }

// IsKind reports whether err is (or wraps) a *VaultError with the given Kind.
func IsKind(err error, k ErrorKind) bool {
	var se *VaultError
	return errors.As(err, &se) && se.Kind == k
}

// Convenience predicates for the two kinds the UI cares about most.
func IsNotFound(err error) bool    { return IsKind(err, ErrKindNotFound) }
func IsCASConflict(err error) bool { return IsKind(err, ErrKindCASConflict) }
