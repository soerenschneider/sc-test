package vault

import (
	"slices"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// DefaultTimeout is the default timeout for requests to Vault.
const DefaultTimeout = 7 * time.Second

// Path joins the given parts to a Vault API path, ignoring superfluous slashes, e.g. of mounts
// passed as "transit/".
func Path(parts ...string) string {
	trimmed := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.Trim(part, "/"); part != "" {
			trimmed = append(trimmed, part)
		}
	}
	return strings.Join(trimmed, "/")
}

// HasCapability returns whether capabilities grants the wanted capability.
func HasCapability(capabilities []string, wanted string) bool {
	return slices.Contains(capabilities, wanted) || slices.Contains(capabilities, "root")
}

// CapTtl caps ttl to the max_ttl of the given role, as Vault refuses to issue credentials with a larger ttl.
// ttl is returned unchanged if it can not be parsed or if the role does not define a max_ttl.
func CapTtl(ttl string, maxTtl time.Duration, roleName string) string {
	if maxTtl <= 0 {
		return ttl
	}

	requested, err := time.ParseDuration(ttl)
	if err != nil || requested <= maxTtl {
		return ttl
	}

	log.Info().Msgf("Requested ttl %v exceeds max_ttl of role %q, using %v instead", requested, roleName, maxTtl)
	return maxTtl.String()
}
