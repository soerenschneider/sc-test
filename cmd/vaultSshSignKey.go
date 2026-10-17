package cmd

import (
	"context"
	"errors"
	"fmt"
	"github.com/soerenschneider/sc/internal/vault/prompt"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/soerenschneider/sc/internal/ssh"
	"github.com/soerenschneider/sc/internal/tui"
	"github.com/soerenschneider/sc/internal/vault"
	"github.com/soerenschneider/sc/pkg"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

var sshSignKeyCmd = &cobra.Command{
	Use:   "sign-key",
	Short: "Signs a SSH public key",
	Long: `Signs a SSH public key using the Vault SSH secret engine and writes the certificate next to it.

Anything that is not passed as a flag is deduced if possible:
  - public key: the only public key in ~/.ssh, otherwise a selection is offered
  - mount: the only SSH mount, otherwise a selection of all SSH mounts is offered
  - role: the only role that is able to sign user keys, otherwise a selection is offered
  - principals: the users allowed by the role, a selection is offered if there are multiple
  - ttl: 24h, capped to the max_ttl of the role`,
	Run: func(cmd *cobra.Command, args []string) {
		client := vault.MustAuthenticateClient(vault.MustBuildClient(cmd))

		req := vault.SshSignatureRequest{
			Ttl:        pkg.GetString(cmd, vaultTtl),
			Principals: pkg.GetStringArray(cmd, vaultPrincipals),
			Extensions: nil, // TODO
			VaultRole:  pkg.GetString(cmd, vaultRoleName),
		}

		forceNewSignature, _ := pkg.GetBool(cmd, vaultForce)

		publicKeyFile := pkg.GetExpandedFile(pkg.GetString(cmd, vaultPublicKeyFile))
		if publicKeyFile == "" {
			publicKeyFile = promptSshPublicKeyFile()
		}

		certificateFile := pkg.GetString(cmd, vaultCertificateFile)
		if certificateFile == "" && publicKeyFile != "" {
			certificateFile = strings.Replace(publicKeyFile, ".pub", "", 1)
			certificateFile = pkg.GetExpandedFile(fmt.Sprintf("%s-cert.pub", certificateFile))
		}

		fs := afero.NewOsFs()
		publicKeyData, err := afero.ReadFile(fs, publicKeyFile)
		if err != nil {
			log.Fatal().Err(err).Msg("Could not read public key data")
		}
		req.PublicKey = string(publicKeyData)

		// Check the existing certificate before talking to Vault, so there are no prompts if
		// there is nothing to do.
		if !forceNewSignature {
			requestNewCertificate, err := needsRequestNewCertificate(fs, certificateFile)
			if err != nil {
				log.Fatal().Err(err).Msg("Can not proceed")
			}

			if !requestNewCertificate {
				return
			}
		}

		mount := pkg.GetString(cmd, vaultMountPath)
		if mount == "" {
			mount = prompt.Mount(cmd.Context(), client, "ssh")
		}

		var role *vault.SshRole
		if req.VaultRole == "" {
			req.VaultRole, role = promptSshRole(cmd.Context(), client, mount)
		}
		if role == nil {
			role = loadSshRole(cmd.Context(), client, mount, req.VaultRole)
		}
		prompt.RequireCapability(cmd.Context(), client, vault.Path(mount, "sign", req.VaultRole), "update")

		if len(req.Principals) == 0 {
			req.Principals = promptSshPrincipals(role)
		}

		if !cmd.Flags().Changed(vaultTtl) && role != nil {
			req.Ttl = vault.CapTtl(req.Ttl, role.MaxTtl, role.Name)
		}

		ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
		defer cancel()

		var certificateData string
		if err := tui.RunWithSpinner(ctx, "Sending signature request...", func(ctx context.Context) error {
			var err error
			certificateData, err = client.SshSignKey(ctx, mount, req)
			return err
		}); err != nil {
			log.Fatal().Err(err).Msg("could not sign public key")
		}

		if err := afero.WriteFile(fs, certificateFile, []byte(certificateData), 0640); err != nil {
			log.Error().Err(err).Msg("could not write signature")
		}

		cert, err := ssh.ParseSshCertData([]byte(certificateData))
		if err != nil {
			log.Fatal().Err(err).Msg("Could not parse received cert data")
		}
		log.Info().Msgf("New certificate written to disk, valid until %v (%v)", cert.ValidBefore, time.Until(cert.ValidBefore).Round(time.Hour))
	},
}

// sshRoleClient is the subset of the vault client needed to discover SSH roles.
type sshRoleClient interface {
	prompt.CapabilityChecker
	SshListRoles(ctx context.Context, mount string) ([]string, error)
	SshReadRole(ctx context.Context, mount, role string) (*vault.SshRole, error)
}

// promptSshPublicKeyFile offers the public keys found in ~/.ssh and falls back to free-form input
// if none are found.
func promptSshPublicKeyFile() string {
	var publicKeys []string

	sshDir := pkg.GetExpandedFile("~/.ssh")
	matches, err := filepath.Glob(filepath.Join(sshDir, "*.pub"))
	if err != nil {
		log.Warn().Err(err).Msg("could not search for public keys")
	}
	for _, match := range matches {
		if !strings.HasSuffix(match, "-cert.pub") {
			publicKeys = append(publicKeys, match)
		}
	}

	return pkg.GetExpandedFile(tui.SelectOrReadInput("public key", publicKeys, nil))
}

// promptSshRole offers the roles of the given mount that are able to sign user keys and falls back to
// free-form input if the roles can not be listed. Returns the name of the role and, if it could be read,
// the role's configuration.
func promptSshRole(ctx context.Context, client sshRoleClient, mount string) (string, *vault.SshRole) {
	roles := map[string]*vault.SshRole{}

	title := fmt.Sprintf("Loading roles of SSH mount %q...", mount)
	names, err := prompt.Load(ctx, title, func(ctx context.Context) ([]string, error) {
		names, err := client.SshListRoles(ctx, mount)
		if err != nil {
			return nil, err
		}

		for _, name := range names {
			role, err := client.SshReadRole(ctx, mount, name)
			if err != nil {
				log.Debug().Err(err).Msgf("could not read SSH role %q", name)
				continue
			}
			roles[name] = role
		}
		return names, nil
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not list SSH roles, falling back to manual input")
	}

	// Only offer roles that can sign user keys. Roles that could not be read are kept, as their
	// capabilities are unknown.
	var usableRoles []string
	for _, name := range names {
		if role, found := roles[name]; !found || role.CanSignUserKeys() {
			usableRoles = append(usableRoles, name)
		} else {
			log.Debug().Msgf("Skipping SSH role %q as it can not sign user keys", name)
		}
	}

	usableRoles = prompt.FilterByCapability(ctx, client, "SSH role", usableRoles, func(role string) string {
		return vault.Path(mount, "sign", role)
	}, "update")

	name := tui.SelectOrReadInput("SSH role", usableRoles, nil)
	return name, roles[name]
}

// loadSshRole reads the configuration of the given role. As the configuration is only used to deduce
// defaults, nil is returned if it can not be read.
func loadSshRole(ctx context.Context, client sshRoleClient, mount, name string) *vault.SshRole {
	title := fmt.Sprintf("Reading SSH role %q...", name)
	role, err := prompt.Load(ctx, title, func(ctx context.Context) (*vault.SshRole, error) {
		return client.SshReadRole(ctx, mount, name)
	})
	if err != nil {
		log.Debug().Err(err).Msgf("could not read SSH role %q, can not deduce defaults", name)
		return nil
	}
	return role
}

// promptSshPrincipals deduces the principals from the users allowed by role. The only allowed user is used
// without asking, a selection is offered for multiple allowed users. If the role allows arbitrary users or
// can not be read, it falls back to free-form input.
func promptSshPrincipals(role *vault.SshRole) []string {
	var allowedUsers []string
	allowsAnyUser := role == nil

	if role != nil {
		// Templated users depend on the identity of the token and can not be resolved locally.
		for _, user := range append([]string{role.DefaultUser}, role.AllowedUsers...) {
			switch {
			case user == "*":
				allowsAnyUser = true
			case user == "" || strings.Contains(user, "{{") || slices.Contains(allowedUsers, user):
				continue
			default:
				allowedUsers = append(allowedUsers, user)
			}
		}
	}

	if !allowsAnyUser {
		switch len(allowedUsers) {
		case 0:
		case 1:
			log.Info().Msgf("Using only allowed principal %q", allowedUsers[0])
			return allowedUsers
		default:
			return tui.MultiSelectInput("Select principals", allowedUsers)
		}
	}

	suggestions := allowedUsers
	if currentUser, err := user.Current(); err == nil && !slices.Contains(suggestions, currentUser.Username) {
		suggestions = append(suggestions, currentUser.Username)
	}
	return []string{tui.ReadInput("Enter principal", suggestions)}
}

func needsRequestNewCertificate(fs afero.Fs, certificateFile string) (bool, error) {
	_, err := os.Stat(certificateFile)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		log.Info().Msg("No existing certificate, requesting new certificate")
		return true, nil
	}

	// TODO: maybe need to check more potential errors
	certData, err := afero.ReadFile(fs, certificateFile)
	if err != nil {
		return false, errors.New("could not read certificate file")
	}

	cert, err := ssh.ParseSshCertData(certData)
	if err != nil {
		log.Warn().Err(err).Msg("could not parse existing cert data, requesting new certificate")
		return true, nil
	}

	requestNewCertificate := time.Until(cert.ValidBefore) < time.Minute*3 || cert.GetPercentage() < 5
	if time.Now().After(cert.ValidBefore) {
		log.Info().Msgf("Certificate exists but already expired %v ago, requesting new one", time.Since(cert.ValidBefore).Round(time.Minute))
	} else {
		action := "Not requesting new certificate"
		if requestNewCertificate {
			action = "Requesting new certificate"
		}
		log.Info().Msgf("%s, certificate at %.0f%% (expires in %v)", action, cert.GetPercentage(), time.Until(cert.ValidBefore).Round(time.Minute))
	}

	return requestNewCertificate, nil
}

func init() {
	vaultSshCmd.AddCommand(sshSignKeyCmd)

	sshSignKeyCmd.Flags().StringP(vaultPublicKeyFile, "p", "", "Location of the public key to sign. If not specified, public keys in ~/.ssh are discovered.")
	sshSignKeyCmd.Flags().StringP(vaultRoleName, "r", "", "Vault role. If not specified, available roles are discovered.")

	sshSignKeyCmd.Flags().StringP(vaultCertificateFile, "c", "", "Where to save the certificate to")
	sshSignKeyCmd.Flags().StringP(vaultTtl, "t", "24h", "TTL of the certificate, capped to the max_ttl of the role if not specified")
	sshSignKeyCmd.Flags().BoolP(vaultForce, "f", false, "Request new certificate regardless of expiry date")

	sshSignKeyCmd.Flags().StringArray(vaultPrincipals, nil, "Principals. If not specified, deduced from the users allowed by the role.")
}

/*
func logIssueResults(result *signature.IssueResult) {
	if result == nil || signature.Unknown == result.Status {
		log.Warn().Msg("empty/unknown signature result returned")
		return
	}

	switch result.Status {
	case signature.Noop:
		if result.ExistingCert != nil {
			percentage := result.ExistingCert.GetPercentage()
			secondsUntilExpiry := int64(math.Max(0, time.Until(result.ExistingCert.ValidBefore).Seconds()))
			log.Info().Int64("ttl", secondsUntilExpiry).Int("lifetime", int(percentage)).Msgf("Existing certificate at %2.f%% still valid until %v", percentage, result.ExistingCert.ValidBefore)
		}
	case signature.Issued:
		if result.IssuedCert != nil {
			secondsUntilExpiry := int64(time.Until(result.IssuedCert.ValidBefore).Seconds())
			log.Info().Int64("ttl", secondsUntilExpiry).Msgf("Issued new certificate, valid until %s", result.IssuedCert.ValidBefore)
		}
	}
}
*/
