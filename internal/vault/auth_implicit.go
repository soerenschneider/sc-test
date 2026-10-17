package vault

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/hashicorp/vault/api"
	"github.com/rs/zerolog/log"
)

const (
	defaultEnvVar    = "VAULT_TOKEN"
	defaultTokenFile = ".vault-token"
)

type TokenImplicitAuth struct {
	envVar    string
	tokenFile string
}

func NewTokenImplicitAuth() *TokenImplicitAuth {
	return &TokenImplicitAuth{
		envVar:    defaultEnvVar,
		tokenFile: defaultTokenFile,
	}
}

func (t *TokenImplicitAuth) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	token := os.Getenv(t.envVar)
	if len(token) > 0 {
		log.Info().Msgf("Using vault token from env var %s", t.envVar)
		return &api.Secret{
			Auth: &api.SecretAuth{
				ClientToken: token,
			},
		}, nil
	}

	dirname, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("can't get user home dir: %v", err)
	}

	tokenPath := path.Join(dirname, t.tokenFile)
	if _, err := os.Stat(tokenPath); errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("file '%s' to read vault token from does not exist", t.tokenFile)
	}

	read, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("error reading file '%s': %v", defaultTokenFile, err)
	}

	log.Info().Msgf("Using vault token from file '%s'", t.tokenFile)
	return &api.Secret{
		Auth: &api.SecretAuth{
			ClientToken: string(read),
		},
	}, nil
}
