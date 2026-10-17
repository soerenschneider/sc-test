package cmd

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/soerenschneider/sc/internal/vault/prompt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/pki"
	"github.com/soerenschneider/sc/internal/pki/format"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"

	"github.com/spf13/cobra"
)

type pkiStorage interface {
	WriteCert(certData *vault.PkiCertData) error
	ReadCert() (*x509.Certificate, error)
}

type vaultPkiIssueOpts struct {
	MinDuration time.Duration
	MinLifetime float32
	Mount       string
}

var vaultPkiIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Issues a x509 certificate",
	Long: `Issues a x509 certificate using the Vault PKI secret engine.

Anything that is not passed as a flag is deduced if possible:
  - mount: the only PKI mount, otherwise a selection of all PKI mounts is offered
  - role: the only role, otherwise a selection is offered
  - common name, alt names and ip sans: taken from the existing certificate when renewing,
    otherwise the common name is prompted for with suggestions based on the role
  - ttl: 24h, capped to the max_ttl of the role`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		issueArgs := vault.PkiIssueArgs{
			CommonName: pkg.GetString(cmd, vaultCommonName),
			Ttl:        pkg.GetString(cmd, vaultTtl),
			Role:       pkg.GetString(cmd, vaultRoleName),
			IpSans:     pkg.GetStringArray(cmd, vaultIpSans),
			AltNames:   pkg.GetStringArray(cmd, vaultAltNames),
		}

		minDurationVal := pkg.GetString(cmd, vaultCertMinDuration)
		minDuration, err := time.ParseDuration(minDurationVal)
		if err != nil {
			log.Fatal().Err(err).Msgf("can not parse %q", vaultCertMinDuration)
		}

		opts := vaultPkiIssueOpts{
			MinDuration: minDuration,
			MinLifetime: pkg.GetFloat32(cmd, vaultCertMinLifetime),
			Mount:       pkg.GetString(cmd, vaultMountPath),
		}

		privateKeyFile := pkg.GetExpandedFile(pkg.GetString(cmd, vaultPrivateKeyFile))
		if privateKeyFile == "" {
			privateKeyFile = promptPkiPrivateKeyFile(issueArgs.CommonName)
		}
		certFile := pkg.GetExpandedFile(pkg.GetString(cmd, vaultCertificateFile))
		caFile := pkg.GetExpandedFile(pkg.GetString(cmd, vaultCaFile))

		var caStorage, certStorage, privateKeyStorage format.StorageImplementation
		if len(caFile) > 0 {
			caStorage, err = pki.NewFilesystemStorageFromUri(caFile)
			pkg.DieOnErr(err, "could not build ca storage")
		}

		if len(certFile) > 0 {
			certStorage, err = pki.NewFilesystemStorageFromUri(certFile)
			pkg.DieOnErr(err, "could not build cert storage")
		}

		privateKeyStorage, err = pki.NewFilesystemStorageFromUri(privateKeyFile)
		pkg.DieOnErr(err, "could not build private key storage")

		storageImpl, err := format.NewKeyPairStorage(certStorage, privateKeyStorage, caStorage)
		pkg.DieOnErr(err, "could not build storage impl")

		existingCert, err := storageImpl.ReadCert()
		if err != nil && !errors.Is(err, pki.ErrNotFound) {
			log.Fatal().Err(err).Msg("could not read certificate")
		}

		if opts.Mount == "" {
			opts.Mount = prompt.Mount(cmd.Context(), client, "pki")
		}

		if !isIssueNewCert(cmd.Context(), client, opts, existingCert) {
			return
		}

		var role *vault.PkiRole
		if issueArgs.Role == "" {
			issueArgs.Role, role = promptPkiRole(cmd.Context(), client, opts.Mount)
		}
		if role == nil {
			role = loadPkiRole(cmd.Context(), client, opts.Mount, issueArgs.Role)
		}
		prompt.RequireCapability(cmd.Context(), client, vault.Path(opts.Mount, "issue", issueArgs.Role), "update")

		if issueArgs.CommonName == "" {
			if existingCert != nil && existingCert.Subject.CommonName != "" {
				reuseCertIdentity(cmd, &issueArgs, existingCert)
			} else {
				issueArgs.CommonName = promptPkiCommonName(role)
			}
		}

		if !cmd.Flags().Changed(vaultTtl) && role != nil {
			issueArgs.Ttl = vault.CapTtl(issueArgs.Ttl, role.MaxTtl, role.Name)
		}

		issuePkiCert(cmd.Context(), client, storageImpl, opts, issueArgs)
	},
}

// pkiRoleClient is the subset of the vault client needed to discover PKI roles.
type pkiRoleClient interface {
	prompt.CapabilityChecker
	PkiListRoles(ctx context.Context, mount string) ([]string, error)
	PkiReadRole(ctx context.Context, mount, role string) (*vault.PkiRole, error)
}

// promptPkiPrivateKeyFile prompts for the file to store the private key in, suggesting a name based on
// the common name or the hostname.
func promptPkiPrivateKeyFile(commonName string) string {
	var suggestions []string
	if commonName != "" {
		suggestions = append(suggestions, commonName+".key")
	}
	if hostname, err := os.Hostname(); err == nil && !slices.Contains(suggestions, hostname+".key") {
		suggestions = append(suggestions, hostname+".key")
	}

	return pkg.GetExpandedFile(tui.ReadInput("Enter private key file", suggestions))
}

// promptPkiRole offers the roles of the given mount and falls back to free-form input if the roles can
// not be listed. Returns the name of the role and, if it could be read, the role's configuration.
func promptPkiRole(ctx context.Context, client pkiRoleClient, mount string) (string, *vault.PkiRole) {
	title := fmt.Sprintf("Loading roles of PKI mount %q...", mount)
	names, err := prompt.Load(ctx, title, func(ctx context.Context) ([]string, error) {
		return client.PkiListRoles(ctx, mount)
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not list PKI roles, falling back to manual input")
	}

	names = prompt.FilterByCapability(ctx, client, "PKI role", names, func(role string) string {
		return vault.Path(mount, "issue", role)
	}, "update")

	name := tui.SelectOrReadInput("PKI role", names, nil)
	return name, loadPkiRole(ctx, client, mount, name)
}

// loadPkiRole reads the configuration of the given role. As the configuration is only used to deduce
// defaults, nil is returned if it can not be read.
func loadPkiRole(ctx context.Context, client pkiRoleClient, mount, name string) *vault.PkiRole {
	title := fmt.Sprintf("Reading PKI role %q...", name)
	role, err := prompt.Load(ctx, title, func(ctx context.Context) (*vault.PkiRole, error) {
		return client.PkiReadRole(ctx, mount, name)
	})
	if err != nil {
		log.Debug().Err(err).Msgf("could not read PKI role %q, can not deduce defaults", name)
		return nil
	}
	return role
}

// reuseCertIdentity takes over the common name of the existing certificate as well as its alt names
// and ip sans, unless they have been passed explicitly.
func reuseCertIdentity(cmd *cobra.Command, issueArgs *vault.PkiIssueArgs, cert *x509.Certificate) {
	issueArgs.CommonName = cert.Subject.CommonName
	log.Info().Msgf("Using common name %q of existing certificate", issueArgs.CommonName)

	if !cmd.Flags().Changed(vaultAltNames) {
		for _, name := range cert.DNSNames {
			if name != issueArgs.CommonName {
				issueArgs.AltNames = append(issueArgs.AltNames, name)
			}
		}
		if len(issueArgs.AltNames) > 0 {
			log.Info().Msgf("Using alt names %v of existing certificate", issueArgs.AltNames)
		}
	}

	if !cmd.Flags().Changed(vaultIpSans) {
		for _, ip := range cert.IPAddresses {
			issueArgs.IpSans = append(issueArgs.IpSans, ip.String())
		}
		if len(issueArgs.IpSans) > 0 {
			log.Info().Msgf("Using ip sans %v of existing certificate", issueArgs.IpSans)
		}
	}
}

// promptPkiCommonName deduces the common name from the names allowed by role. If the role only allows a
// single domain name, it is used without asking. Otherwise, the common name is prompted for, suggesting names
// allowed by the role based on the hostname.
func promptPkiCommonName(role *vault.PkiRole) string {
	hostname, _ := os.Hostname()
	shortHostname, _, _ := strings.Cut(hostname, ".")

	var suggestions []string
	add := func(name string) {
		if name != "" && !slices.Contains(suggestions, name) {
			suggestions = append(suggestions, name)
		}
	}

	if role == nil {
		add(hostname)
		return tui.ReadInput("Enter common name", suggestions)
	}

	for _, domain := range role.AllowedDomains {
		// Glob and templated domains can not be resolved locally.
		if strings.ContainsAny(domain, "*{") {
			continue
		}
		if role.AllowSubdomains && shortHostname != "" {
			add(shortHostname + "." + domain)
		}
		if role.AllowBareDomains {
			add(domain)
		}
	}
	// allow_localhost is enabled by default but rarely the desired common name, so it is not taken
	// into account when deciding whether the role only allows a single name.
	onlyBareDomain := role.AllowBareDomains && !role.AllowSubdomains && !role.AllowGlobDomains &&
		!role.AllowAnyName && len(role.AllowedDomains) == 1 && len(suggestions) == 1
	if onlyBareDomain {
		log.Info().Msgf("Using only allowed common name %q", suggestions[0])
		return suggestions[0]
	}

	if role.AllowLocalhost {
		add("localhost")
	}
	if role.AllowAnyName {
		add(hostname)
	}

	return tui.ReadInput("Enter common name", suggestions)
}

func issuePkiCert(ctx context.Context, client *vault.VaultClient, storageImpl pkiStorage, opts vaultPkiIssueOpts, issueArgs vault.PkiIssueArgs) {
	// depending on the hardware running Vault, the available randomness and the key size it's possible that this
	// operating takes quite some time
	ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
	defer cancel()

	var secret *api.Secret
	if err := tui.RunWithSpinner(ctx, "Issuing new x509 certificate...", func(ctx context.Context) error {
		var err error
		secret, err = client.PkiIssueCert(ctx, opts.Mount, issueArgs)
		return err
	}); err != nil {
		log.Fatal().Err(err).Msg("could not issue x509 certificate")
	}

	issuerCertData := &vault.PkiCertData{
		PrivateKey:  []byte(fmt.Sprintf("%s", secret.Data["private_key"])),
		Certificate: []byte(fmt.Sprintf("%s", secret.Data["certificate"])),
		CaData:      []byte(fmt.Sprintf("%s", secret.Data["issuing_ca"])),
	}

	if err := storageImpl.WriteCert(issuerCertData); err != nil {
		log.Fatal().Err(err).Msg("could not write x509 certificate")
	}

	cert, err := pki.ParseCertPem(issuerCertData.Certificate)
	if err != nil {
		log.Fatal().Err(err).Msg("could not parse issued certificate data")
	}
	log.Info().Msgf("New x509 certificate written to disk, valid until %v (%v)", cert.NotAfter, time.Until(cert.NotAfter).Round(time.Hour))
}

func isIssueNewCert(ctx context.Context, client *vault.VaultClient, opts vaultPkiIssueOpts, cert *x509.Certificate) bool {
	if cert == nil {
		return true
	}

	caChainData, err := prompt.Load(ctx, "Fetching ca-chain from Vault", func(ctx context.Context) ([]byte, error) {
		return client.PkiGetCaChain(ctx, opts.Mount)
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not fetch ca-chain")
	}

	caBlock, _ := pem.Decode(caChainData)
	var ca *x509.Certificate
	if caBlock != nil {
		var err error
		ca, err = x509.ParseCertificate(caBlock.Bytes)
		if err != nil {
			log.Warn().Err(err).Msg("could not parse ca-data")
		}
	}

	certValid, err := isCertValid(opts, ca, cert)
	if err == nil && certValid {
		percentage := pki.GetCertRemainingLifetimePercent(*cert)
		log.Info().Msgf("Existing certificate still valid for %v (%0.1f%%), not issuing new certificate", time.Until(cert.NotAfter).Round(time.Minute), percentage)
		return false
	}

	return true
}

func isCertValid(opts vaultPkiIssueOpts, ca, cert *x509.Certificate) (bool, error) {
	if cert == nil {
		return false, errors.New("cert must not be nil")
	}

	if pki.IsCertExpired(*cert) {
		return false, nil
	}

	if ca != nil {
		err := pki.VerifyCertAgainstCa(cert, ca)
		if err != nil && errors.Is(err, pki.ErrValidationFailed) {
			log.Warn().Err(err).Msg("validation against ca failed, cert has been issued to old ca")
			return false, nil
		}
	}

	expiresSoon, err := pki.MustRenewCert(cert, opts.MinDuration, opts.MinLifetime)
	if expiresSoon {
		wanted := fmt.Sprintf("%s / %0.1f%%", opts.MinDuration, opts.MinLifetime)
		got := fmt.Sprintf("%s / %0.1f%%", time.Until(cert.NotAfter).Round(time.Minute), pki.GetCertRemainingLifetimePercent(*cert))
		log.Info().Str("wanted", wanted).Str("got", got).Msg("Certificate lifetime threshold passed, issuing new certificate")
		return false, err
	}
	return true, err
}

func init() {
	vaultPkiCmd.AddCommand(vaultPkiIssueCmd)

	vaultPkiIssueCmd.Flags().StringP(vaultCommonName, "n", "", "The CN for the certificate. If not specified, taken from the existing certificate or prompted for.")
	vaultPkiIssueCmd.Flags().StringP(vaultPrivateKeyFile, "k", "", "File to save the private key to. If not specified, prompted for.")
	vaultPkiIssueCmd.Flags().StringP(vaultRoleName, "r", "", "Vault role. If not specified, available roles are discovered.")

	vaultPkiIssueCmd.Flags().StringP(vaultCertificateFile, "c", "", "File to save the certificate to")
	vaultPkiIssueCmd.Flags().String(vaultCaFile, "", "File to save the ca to")
	vaultPkiIssueCmd.Flags().StringP(vaultTtl, "t", "24h", "TTL of the certificate, capped to the max_ttl of the role if not specified")
	vaultPkiIssueCmd.Flags().Float32P(vaultCertMinLifetime, "l", 10., "Minimum percentage of cert lifetime left before forcing issuing a new certificate")
	vaultPkiIssueCmd.Flags().StringP(vaultCertMinDuration, "d", "15m", "Minimum duration of the cert before forcing issuing a new certificate")

	vaultPkiIssueCmd.Flags().StringArrayP(vaultIpSans, "s", nil, "IP Sans")
	vaultPkiIssueCmd.Flags().StringArray(vaultAltNames, nil, "Alternative names")
}
