package fmt

import (
	"fmt"
	"strings"
	"time"

	"github.com/soerenschneider/sc-agent/pkg/api"
)

func PrintManagedCertificateConfigs(configs []api.SshManagedCertificate) {
	for i, config := range configs {
		fmt.Printf("Configuration %d:\n", i+1)
		PrintManagedCertificateConfig(config)
		fmt.Println()
	}
}

// PrintManagedCertificateConfig prints a ManagedCertificateConfig struct in a formatted manner.
func PrintManagedCertificateConfig(m api.SshManagedCertificate) {
	// Handle nil pointer scenarios gracefully
	var certConfigStr, storageConfigStr, certificateStr string

	if m.CertificateConfig != nil {
		certConfigStr = formatCertificateConfig(*m.CertificateConfig)
	} else {
		certConfigStr = "N/A"
	}

	if m.StorageConfig != nil {
		storageConfigStr = formatCertificateStorage(*m.StorageConfig)
	} else {
		storageConfigStr = "N/A"
	}

	if m.Certificate != nil {
		certificateStr = formatCertificate(*m.Certificate)
	} else {
		certificateStr = "N/A"
	}

	// Print the entire configuration
	fmt.Printf(`Managed Certificate Configuration:
-----------------------------
Certificate Config:
%s

Storage Config:
%s

Certificate:
%s
`, certConfigStr, storageConfigStr, certificateStr)
}

// Helper function to format CertificateConfig struct
func formatCertificateConfig(c api.SshCertificateConfig) string {
	principals := strings.Join(c.Principals, ", ")

	return fmt.Sprintf(`  ID             : %s
  Role           : %s
  Principals     : %s
  TTL            : %s
  Certificate Type: %s`,
		c.Id,
		c.Role,
		principals,
		c.Ttl,
		c.CertType,
	)
}

// Helper function to format CertificateStorage struct
func formatCertificateStorage(s api.SshCertificateStorage) string {
	return fmt.Sprintf(`  Public Key File    : %s
  Certificate File   : %s`,
		s.PublicKeyFile,
		s.CertificateFile,
	)
}

// Helper function to format Certificate struct
func formatCertificate(c api.SshCertificateData) string {
	criticalOptions := formatMap(c.CriticalOptions)
	extensions := formatMap(c.Extensions)
	principals := strings.Join(c.Principals, ", ")

	validAfter := c.ValidAfter.Format(time.RFC1123)
	validBefore := c.ValidBefore.Format(time.RFC1123)
	percentage := fmt.Sprintf("%.2f%%", c.Percentage)

	return fmt.Sprintf(`  Type            : %s
  Serial          : %d
  Valid After     : %s
  Valid Before    : %s
  Principals      : %s
  Critical Options: %s
  Extensions      : %s
  Percentage Used : %s`,
		c.Type,
		c.Serial,
		validAfter,
		validBefore,
		principals,
		criticalOptions,
		extensions,
		percentage,
	)
}

// Helper function to format a map[string]string for display
func formatMap(m map[string]string) string {
	if len(m) == 0 {
		return "None"
	}

	var sb strings.Builder
	for key, value := range m {
		_, _ = fmt.Fprintf(&sb, "%s: %s; ", key, value)
	}

	return strings.TrimSuffix(sb.String(), "; ")
}
