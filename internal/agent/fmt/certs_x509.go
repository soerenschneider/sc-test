package fmt

import (
	"fmt"
	"strings"
	"time"

	"github.com/soerenschneider/sc-agent/pkg/api"
)

func PrintPkiManagedCertificateConfigs(configs []api.X509ManagedCertificate) {
	for i, config := range configs {
		fmt.Printf("Configuration %d:\n", i+1)
		PrintPkiManagedCertificateConfig(config)
		fmt.Println() // Add a blank line between configurations
	}
}

func PrintPkiManagedCertificateConfig(m api.X509ManagedCertificate) {
	var certStr, certConfigStr, postHooksStr, storageConfigStr string

	if m.CertificateData != nil {
		certStr = formatPkiCertificate(*m.CertificateData)
	} else {
		certStr = "N/A"
	}

	if m.CertificateConfig != nil {
		certConfigStr = formatPkiCertificateConfig(*m.CertificateConfig)
	} else {
		certConfigStr = "N/A"
	}

	if len(m.PostHooks) > 0 {
		postHooksStr = agentFormatPostHooks(m.PostHooks)
	} else {
		postHooksStr = "N/A"
	}

	if len(m.StorageConfig) > 0 {
		storageConfigStr = formatPkiCertificateStorages(m.StorageConfig)
	} else {
		storageConfigStr = "N/A"
	}

	fmt.Printf(`Pki Managed Certificate Configuration:
-----------------------------
Certificate:
%s

Certificate Config:
%s

Post Hooks:
%s

Storage Config:
%s
`, certStr, certConfigStr, postHooksStr, storageConfigStr)
}

func formatPkiCertificate(c api.X509CertificateData) string {
	emailAddresses := strings.Join(c.EmailAddresses, ", ")
	notAfter := c.NotAfter.Format(time.RFC1123)
	notBefore := c.NotBefore.Format(time.RFC1123)
	percentage := fmt.Sprintf("%.2f%%", c.Percentage)

	return fmt.Sprintf(`  Email Addresses : %s
  Issuer          : %s
  Not After       : %s
  Not Before      : %s
  Percentage Used : %s
  Serial          : %s
  Subject         : %s`,
		emailAddresses,
		c.Issuer.CommonName,
		notAfter,
		notBefore,
		percentage,
		c.Serial,
		c.Subject,
	)
}

func formatPkiCertificateConfig(c api.X509CertificateConfig) string {
	altNames := strings.Join(c.AltNames, ", ")
	ipSans := strings.Join(c.IpSans, ", ")

	return fmt.Sprintf(`  Alt Names  : %s
  Common Name: %s
  ID         : %s
  IP SANs    : %s
  Role       : %s
  TTL        : %s`,
		altNames,
		c.CommonName,
		c.Id,
		ipSans,
		c.Role,
		c.Ttl,
	)
}

func formatPkiCertificateStorages(storageConfigs []api.X509CertificateStorage) string {
	var sb strings.Builder
	for _, storage := range storageConfigs {
		_, _ = fmt.Fprintf(&sb, `  CA File    : %s
  Cert File  : %s
  Key File   : %s
`, storage.CaFile, storage.CertFile, storage.KeyFile)
	}
	return sb.String()
}
