package storage

import (
	"encoding/json"
	"errors"

	"github.com/keybase/go-keychain"
	"golang.org/x/oauth2"
)

type SecureStore struct {
	account string
	service string
}

func NewStorage() *SecureStore {
	return &SecureStore{
		account: "cloud.soeren.sc-agent",
		service: "sc-agent",
	}
}

func (s *SecureStore) SaveToken(token *oauth2.Token) error {
	data, err := json.Marshal(token) //#nosec:G117
	if err != nil {
		return err
	}

	item := keychain.NewItem()
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetService(s.service)
	item.SetAccount(s.account)
	item.SetLabel("OAuth Token")
	item.SetData(data)
	item.SetAccessible(keychain.AccessibleAfterFirstUnlockThisDeviceOnly)

	_ = keychain.DeleteGenericPasswordItem(s.service, s.account)

	return keychain.AddItem(item)
}

func (s *SecureStore) LoadToken() (*oauth2.Token, error) {
	query := keychain.NewItem()
	query.SetSecClass(keychain.SecClassGenericPassword)
	query.SetService(s.service)
	query.SetAccount(s.account)
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnData(true)
	query.SetAccessible(keychain.AccessibleAfterFirstUnlockThisDeviceOnly)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, errors.New("token not found in keychain")
	}

	var token oauth2.Token
	err = json.Unmarshal(results[0].Data, &token)
	return &token, err
}

// DeleteToken removes a token from Keychain
func (s *SecureStore) DeleteToken() error {
	return keychain.DeleteGenericPasswordItem(s.service, s.account)
}
