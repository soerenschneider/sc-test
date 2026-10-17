package storage

import (
	"encoding/json"

	"github.com/zalando/go-keyring"
	"golang.org/x/oauth2"
)

// SecureStore implements TokenStore using Secret Service via go-keyring
type SecureStore struct {
	user    string
	service string
}

func NewStorage() *SecureStore {
	panic("not implemented")
}

// SaveToken stores the token in the system keyring
func (s *SecureStore) SaveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token) //#nosec:G117
	if err != nil {
		return err
	}

	return keyring.Set(s.service, s.user, string(data))
}

// LoadToken retrieves the token from the system keyring
func (s *SecureStore) LoadToken() (*oauth2.Token, error) {
	secret, err := keyring.Get(s.service, s.user)
	if err != nil {
		return nil, err
	}

	var token oauth2.Token
	err = json.Unmarshal([]byte(secret), &token)
	return &token, err
}

// DeleteToken removes the token from the keyring
func (s *SecureStore) DeleteToken() error {
	return keyring.Delete(s.service, s.user)
}
