package vault

import (
	"cmp"
	"context"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	vault "github.com/hashicorp/vault/api"
	approle "github.com/hashicorp/vault/api/auth/approle"
	"github.com/rs/zerolog/log"
	backend "github.com/soerenschneider/sc/internal/pki"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"
)

const (
	vaultAddrEnvVarKey  = "VAULT_ADDR"
	vaultTokenEnvVarKey = "VAULT_TOKEN"
)

type AwsCredentials struct {
	AccessKeyID     string
	SecretAccessKey string
	// SessionToken is only set for assumed_role and federation_token credentials.
	SessionToken string
}

type MfaCreateMethodRequest struct {
	MethodName string `json:"method_name"`
	Issuer     string `json:"issuer"`
	Algorithm  string `json:"algorithm"`
}

type VaultApproleConfig struct {
	RoleID   string
	RoleName string
	SecretID *approle.SecretID
}

type VaultClient struct {
	client *vault.Client
}

func MustBuildClient(command *cobra.Command) *VaultClient {
	vaultAddr, err := GetVaultAddress(command)
	if err != nil {
		log.Fatal().Err(err).Msg("could not get vault address")
	}

	config := vault.DefaultConfig()
	config.Address = vaultAddr

	client, err := vault.NewClient(config)
	if err != nil {
		log.Fatal().Msgf("Error creating Vault client: %v", err)
	}

	return &VaultClient{
		client: client,
	}
}

func MustAuthenticateClient(client *VaultClient) *VaultClient {
	if client == nil { //nolint SA5011
		log.Fatal().Msg("nil client passed")
	}

	token, err := GetVaultToken()
	if err != nil {
		log.Fatal().Err(err).Msg("unable to get Vault token")
	}

	client.client.SetToken(token) //nolint SA5011
	return client
}

func GetVaultAddress(cmd *cobra.Command) (string, error) {
	address, err := cmd.Flags().GetString("addr")
	if err == nil && len(address) > 0 {
		return address, nil
	}

	address, found := os.LookupEnv(vaultAddrEnvVarKey)
	if !found {
		return "", errors.New("no vault address specified")
	}

	log.Info().Msgf("No vault address supplied explicitly, using value of env var %s=%s", vaultAddrEnvVarKey, address)
	return address, nil
}

func GetVaultToken() (string, error) {
	token, ok := os.LookupEnv(vaultTokenEnvVarKey)
	if ok {
		return token, nil
	}

	tokenFilePath, err := GetVaultTokenFilePath()
	if err != nil {
		return "", fmt.Errorf("could not get default Vault token file: %w", err)
	}

	tokenData, err := os.ReadFile(tokenFilePath)
	if err != nil {
		return "", fmt.Errorf("could not read Vault token from file: %w", err)
	}

	return string(tokenData), nil
}

// getVaultTokenFilePath returns the default Vault token file path
func GetVaultTokenFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("could not get user home directory")
	}
	return filepath.Join(homeDir, ".vault-token"), nil
}

func (c *VaultClient) AwsListRoles(ctx context.Context, mount string) ([]string, error) {
	path := fmt.Sprintf("%s/roles", mount)
	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS credentials: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	var keys []string
	for _, v := range secret.Data["keys"].([]any) {
		if s, ok := v.(string); ok {
			keys = append(keys, s)
		}
	}

	return keys, nil
}

func (c *VaultClient) LookupToken(ctx context.Context) (*vault.Secret, error) {
	return c.client.Auth().Token().LookupSelfWithContext(ctx)
}

func (c *VaultClient) RevokeSelfToken(ctx context.Context) error {
	return c.client.Auth().Token().RevokeSelfWithContext(ctx, "")
}

// RenewSelfToken renews the current token. An increment of 0 lets Vault apply the default TTL.
func (c *VaultClient) RenewSelfToken(ctx context.Context, increment time.Duration) (*vault.Secret, error) {
	return c.client.Auth().Token().RenewSelfWithContext(ctx, int(increment.Seconds()))
}

func (c *VaultClient) AwsGenCreds(ctx context.Context, mount string, role string, ttl string) (*AwsCredentials, error) {
	path := fmt.Sprintf("%s/creds/%s", mount, role)

	options := map[string][]string{
		"ttl": {cmp.Or(ttl, "3600s")},
	}

	secret, err := c.client.Logical().ReadWithDataWithContext(ctx, path, options)
	if err != nil {
		return nil, fmt.Errorf("failed to read AWS credentials: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	accessKey, ok1 := secret.Data["access_key"].(string)
	secretKey, ok2 := secret.Data["secret_key"].(string)
	if !ok1 || !ok2 {
		return nil, fmt.Errorf("unexpected response structure: %#v", secret.Data)
	}

	sessionToken, _ := secret.Data["security_token"].(string)

	return &AwsCredentials{
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		SessionToken:    sessionToken,
	}, nil
}

func (c *VaultClient) SshListRoles(ctx context.Context, mount string) ([]string, error) {
	path := fmt.Sprintf("%s/roles", mount)
	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	rawKeys, ok := secret.Data["keys"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response structure: %#v", secret.Data)
	}

	var keys []string
	for _, v := range rawKeys {
		if s, ok := v.(string); ok {
			keys = append(keys, s)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// SshReadRole reads the configuration of the given role of the SSH secret engine mounted at mount.
func (c *VaultClient) SshReadRole(ctx context.Context, mount, role string) (*SshRole, error) {
	path := fmt.Sprintf("%s/roles/%s", mount, role)
	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read role: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("role %q not found", role)
	}

	ret := &SshRole{Name: role}
	ret.KeyType, _ = secret.Data["key_type"].(string)
	ret.AllowUserCertificates, _ = secret.Data["allow_user_certificates"].(bool)
	ret.DefaultUser, _ = secret.Data["default_user"].(string)
	if allowedUsers, ok := secret.Data["allowed_users"].(string); ok {
		for _, user := range strings.Split(allowedUsers, ",") {
			if user = strings.TrimSpace(user); user != "" {
				ret.AllowedUsers = append(ret.AllowedUsers, user)
			}
		}
	}
	ret.Ttl = parseVaultDuration(secret.Data["ttl"])
	ret.MaxTtl = parseVaultDuration(secret.Data["max_ttl"])

	return ret, nil
}

// parseVaultDuration parses a duration as returned by Vault, either as number of seconds or as duration string.
// Returns 0 if the value can not be parsed.
func parseVaultDuration(val any) time.Duration {
	switch v := val.(type) {
	case json.Number:
		if seconds, err := v.Int64(); err == nil {
			return time.Duration(seconds) * time.Second
		}
	case float64:
		return time.Duration(v) * time.Second
	case int64:
		return time.Duration(v) * time.Second
	case int:
		return time.Duration(v) * time.Second
	case string:
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
		if seconds, err := strconv.ParseInt(v, 10, 64); err == nil {
			return time.Duration(seconds) * time.Second
		}
	}
	return 0
}

func (c *VaultClient) UserpassAuth(ctx context.Context, mount, username, password string) (*vault.Secret, error) {
	data := map[string]any{"password": password}
	path := fmt.Sprintf("auth/%s/login/%s", mount, username)
	return c.client.Logical().WriteWithContext(ctx, path, data)
}

func (c *VaultClient) ApproleAuth(ctx context.Context, mount string, conf *VaultApproleConfig) (*vault.Secret, error) {
	opts := []approle.LoginOption{
		approle.WithMountPath(mount),
	}
	data, err := approle.NewAppRoleAuth(conf.RoleID, conf.SecretID, opts...)
	if err != nil {
		return nil, err
	}
	return c.client.Auth().Login(ctx, data)
}

func (c *VaultClient) GetApproleID(ctx context.Context, mount, roleName string) (string, error) {
	if roleName == "" {
		return "", fmt.Errorf("roleName must not be empty")
	}

	path := fmt.Sprintf("auth/%s/role/%s/role-id", mount, roleName)

	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return "", fmt.Errorf("failed to read role-id for role %q: %w", roleName, err)
	}
	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no data returned for role %q", roleName)
	}

	roleIDRaw, ok := secret.Data["role_id"]
	if !ok {
		return "", fmt.Errorf("role_id not found in response for role %q", roleName)
	}

	roleID, ok := roleIDRaw.(string)
	if !ok || roleID == "" {
		return "", fmt.Errorf("invalid role_id format for role %q", roleName)
	}

	return roleID, nil
}

// ListAppRoles returns all AppRole names
func (c *VaultClient) ListAppRoles(ctx context.Context, mount string) ([]string, error) {
	secret, err := c.client.Logical().ListWithContext(ctx, fmt.Sprintf("auth/%s/role", mount))
	if err != nil {
		return nil, fmt.Errorf("failed to list approles: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return []string{}, nil
	}

	rawKeys, ok := secret.Data["keys"]
	if !ok {
		return []string{}, nil
	}

	keysInterface, ok := rawKeys.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected keys format")
	}

	roles := make([]string, 0, len(keysInterface))
	for _, k := range keysInterface {
		if roleName, ok := k.(string); ok {
			roles = append(roles, roleName)
		}
	}

	return roles, nil
}

// GetAppRoleAccessors returns all secret_id_accessors for a given AppRole
func (c *VaultClient) GetAppRoleAccessors(ctx context.Context, mount, roleName string) ([]string, error) {
	// Path to list secret IDs (accessors)
	path := fmt.Sprintf("auth/%s/role/%s/secret-id", mount, roleName)

	// Perform LIST operation
	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list secret IDs: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned for role %s", roleName)
	}

	// Extract keys (accessors)
	keysRaw, ok := secret.Data["keys"]
	if !ok {
		return nil, fmt.Errorf("no keys field in response")
	}

	keysInterface, ok := keysRaw.([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected keys format")
	}

	accessors := make([]string, 0, len(keysInterface))
	for _, k := range keysInterface {
		if s, ok := k.(string); ok {
			accessors = append(accessors, s)
		}
	}

	return accessors, nil
}

func (c *VaultClient) GetAppRoleAccessorMetadata(ctx context.Context, mount, roleName, accessor string) error {
	lookupPath := fmt.Sprintf("auth/approle/role/%s/secret-id-accessor/lookup", roleName)

	resp, err := c.client.Logical().WriteWithContext(ctx, lookupPath, map[string]interface{}{
		"secret_id_accessor": accessor,
	})
	if err != nil {
		log.Printf("failed to lookup accessor %s: %v", accessor, err)
		return err
	}

	if resp == nil || resp.Data == nil {
		log.Printf("no data for accessor %s", accessor)
		return err
	}

	// Step 3: Extract expiration info
	expirationStr, ok := resp.Data["expiration_time"].(string)
	if !ok || expirationStr == "" {
		log.Printf("accessor %s has no expiration_time (might be non-expiring)", accessor)
		return err
	}

	expirationTime, err := time.Parse(time.RFC3339, expirationStr)
	if err != nil {
		log.Printf("failed to parse expiration_time for accessor %s: %v", accessor, err)
		return err
	}

	now := time.Now().UTC()

	if now.After(expirationTime) {
		fmt.Printf("EXPIRED: accessor=%s expired at %s\n", accessor, expirationTime)
	} else {
		fmt.Printf("VALID: accessor=%s expires at %s\n", accessor, expirationTime)
	}

	return nil
}

// DeleteAppRoleAccessor deletes a secret_id_accessor for a given AppRole
func (c *VaultClient) DeleteAppRoleAccessor(ctx context.Context, mount, roleName, accessor string) error {
	path := fmt.Sprintf("auth/%s/role/%s/secret-id-accessor/destroy", mount, roleName)

	data := map[string]interface{}{
		"secret_id_accessor": accessor,
	}

	_, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return fmt.Errorf("failed to delete accessor %s for role %s: %w", accessor, roleName, err)
	}

	return nil
}

func (c *VaultClient) DeleteAllAppRoleAccessors(ctx context.Context, mount, roleName string) error {
	accessors, err := c.GetAppRoleAccessors(ctx, mount, roleName)
	if err != nil {
		return fmt.Errorf("failed to list accessors: %w", err)
	}

	if len(accessors) == 0 {
		return nil
	}

	sem := make(chan struct{}, 4)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errs error

	for _, accessor := range accessors {
		wg.Add(1)
		sem <- struct{}{}

		go func(acc string) {
			defer wg.Done()
			defer func() { <-sem }()

			if err := c.DeleteAppRoleAccessor(ctx, mount, roleName, acc); err != nil {
				mu.Lock()
				errs = multierr.Append(errs, err)
				mu.Unlock()
			}
		}(accessor)
	}

	wg.Wait()

	return errs
}

// CreateAppRoleSecretID creates a new SecretID for the given AppRole.
// Optionally accepts metadata, CIDR restrictions, and TTL.
func (c *VaultClient) CreateAppRoleSecretID(ctx context.Context, mount string, roleName string, metadata map[string]string, cidrList []string, tokenBoundCIDRs []string, ttl string) (secretID string, accessor string, err error) {
	path := fmt.Sprintf("auth/%s/role/%s/secret-id", mount, roleName)

	data := map[string]any{}
	if metadata != nil {
		data["metadata"] = metadata
	}
	if len(cidrList) > 0 {
		data["cidr_list"] = cidrList
	}
	if len(tokenBoundCIDRs) > 0 {
		data["token_bound_cidrs"] = tokenBoundCIDRs
	}
	if ttl != "" {
		data["ttl"] = ttl
	}

	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", "", fmt.Errorf("failed to create secret ID: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", "", fmt.Errorf("no data returned from Vault")
	}

	secretIDRaw, ok := secret.Data["secret_id"]
	if !ok {
		return "", "", fmt.Errorf("missing secret_id in response")
	}

	accessorRaw, ok := secret.Data["secret_id_accessor"]
	if !ok {
		return "", "", fmt.Errorf("missing secret_id_accessor in response")
	}

	secretID, ok = secretIDRaw.(string)
	if !ok {
		return "", "", fmt.Errorf("invalid secret_id type")
	}

	accessor, ok = accessorRaw.(string)
	if !ok {
		return "", "", fmt.Errorf("invalid secret_id_accessor type")
	}

	return secretID, accessor, nil
}

// ResetAppRoleSecretIDs deletes ALL secret_id_accessors for a role
// and then generates a fresh SecretID.
func (c *VaultClient) ResetAppRoleSecretIDs(ctx context.Context, mount, roleName string, metadata map[string]string, ttl string) (newSecretID string, newAccessor string, err error) {
	// Step 1: list all accessors
	accessors, err := c.GetAppRoleAccessors(ctx, mount, roleName)
	if err != nil {
		return "", "", fmt.Errorf("failed to list accessors: %w", err)
	}

	// Step 2: delete each accessor
	for _, accessor := range accessors {
		if err := c.DeleteAppRoleAccessor(ctx, mount, roleName, accessor); err != nil {
			return "", "", fmt.Errorf("failed deleting accessor %s: %w", accessor, err)
		}
	}

	// Step 3: create a new SecretID
	secretID, accessor, err := c.CreateAppRoleSecretID(
		ctx,
		mount,
		roleName,
		metadata,
		nil,
		nil,
		ttl,
	)
	if err != nil {
		return "", "", fmt.Errorf("failed to create new secret ID: %w", err)
	}

	return secretID, accessor, nil
}

func (c *VaultClient) UpdatePassword(ctx context.Context, mount, username, password string) error {
	userPath := fmt.Sprintf("auth/%s/users/%s/password", mount, username)
	data := map[string]interface{}{
		"password": password,
	}

	_, err := c.client.Logical().WriteWithContext(ctx, userPath, data)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}
	return nil
}

func GetVaultMfaRequirement(secret *vault.Secret) (*vault.MFARequirement, string, error) {
	if secret == nil {
		return nil, "", errors.New("empty secret supplied")
	}

	mfaRequirement := secret.Auth.MFARequirement
	defaultMfa, found := mfaRequirement.MFAConstraints["default"]
	if !found {
		return nil, "", errors.New("no default mfa requirement available")
	}

	if len(defaultMfa.Any) == 0 {
		return nil, "", errors.New("no mfa requirements provided")
	}

	method := defaultMfa.Any[0]
	mfaType := method.Type
	if mfaType == "totp" {
		return mfaRequirement, method.ID, nil
	}

	return nil, "", errors.New("no totp available")
}

func (c *VaultClient) SendOtp(ctx context.Context, mfaReq *vault.MFARequirement, otp string) (*vault.Secret, error) {
	if mfaReq == nil {
		return nil, errors.New("nil mfarequirement passed")
	}

	defaultMfa, found := mfaReq.MFAConstraints["default"]
	if !found {
		return nil, errors.New("no default mfa requirement available")
	}

	if len(defaultMfa.Any) == 0 {
		return nil, errors.New("no mfa requirements provided")
	}

	method := defaultMfa.Any[0]
	mfaId := method.ID

	mfaPayload := map[string]any{
		"mfa_request_id": mfaReq.MFARequestID,
		"mfa_payload": map[string]any{
			mfaId: []string{otp},
		},
	}

	return c.client.Logical().WriteWithContext(ctx, "sys/mfa/validate", mfaPayload)
}

func (c *VaultClient) UserpassAuthMfa(ctx context.Context, username, password, mfaId, otp string) (*vault.Secret, error) {
	var vaultMfaHeaderValue = mfaId
	if otp != "" {
		vaultMfaHeaderValue = fmt.Sprintf("%s: %s", mfaId, otp)
	}
	c.client.SetHeaders(map[string][]string{
		"X-Vault-MFA": {
			vaultMfaHeaderValue,
		},
	})
	defer c.client.SetHeaders(nil)

	data := map[string]any{"password": password}
	path := fmt.Sprintf("auth/userpass/login/%s", username)
	return c.client.Logical().WriteWithContext(ctx, path, data)
}

func (c *VaultClient) TotpListMethods(ctx context.Context) ([]string, error) {
	path := "identity/mfa/method/totp"
	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	var keys []string
	for _, v := range secret.Data["keys"].([]any) {
		if s, ok := v.(string); ok {
			keys = append(keys, s)
		}
	}

	return keys, nil
}

func (c *VaultClient) WrapValue(data map[string]interface{}, ttl string) (string, error) {
	if ttl == "" {
		return "", fmt.Errorf("ttl must not be empty")
	}

	// Enable wrapping
	c.client.SetWrappingLookupFunc(func(operation, path string) string {
		return ttl
	})

	secret, err := c.client.Logical().Write("sys/wrapping/wrap", data)
	if err != nil {
		return "", fmt.Errorf("failed to wrap value: %w", err)
	}

	if secret == nil || secret.WrapInfo == nil {
		return "", fmt.Errorf("no wrap info returned")
	}

	return secret.WrapInfo.Token, nil
}

func structToMap(v any) (map[string]any, error) {
	var result map[string]any
	bytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(bytes, &result)
	return result, err
}

func (c *VaultClient) TotpGenerateMethod(ctx context.Context, req MfaCreateMethodRequest) (string, error) {
	path := "identity/mfa/method/totp"
	data, err := structToMap(req)
	if err != nil {
		return "", err
	}
	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", err
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no data returned from Vault")
	}

	return secret.Data["method_id"].(string), nil
}

func (c *VaultClient) TotpDestroySecretAdmin(ctx context.Context, methodId, entityId string) error {
	path := "identity/mfa/method/totp/admin-destroy"
	data := map[string]any{
		"method_id": methodId,
		"entity_id": entityId,
	}
	_, err := c.client.Logical().WriteWithContext(ctx, path, data)
	return err
}

var ErrTotpAlreadyDefined = errors.New("TOTP already defined")
var ErrTotpDeleteFailed = errors.New("TOTP deletion failed")

func (c *VaultClient) TotpGenerateSecretAdmin(ctx context.Context, methodId, entityId string, force bool) (string, error) {
	path := "identity/mfa/method/totp/admin-generate"
	data := map[string]any{
		"method_id": methodId,
		"entity_id": entityId,
	}

	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", err
	}

	if secret == nil || secret.Data == nil || len(secret.Warnings) > 0 {
		log.Warn().Msgf("TOTP already defined for %s", entityId)
		if !force {
			return "", ErrTotpAlreadyDefined
		}

		if err := c.TotpDestroySecretAdmin(ctx, methodId, entityId); err != nil {
			return "", ErrTotpDeleteFailed
		}
		return c.TotpGenerateSecretAdmin(ctx, methodId, entityId, false)
	}

	return fmt.Sprintf("%s", secret.Data["url"]), nil
}

// IdentityGetEntityMfaMethodIds returns the ids of all MFA methods the entity has a secret for.
func (c *VaultClient) IdentityGetEntityMfaMethodIds(ctx context.Context, entityId string) ([]string, error) {
	resp, err := c.client.Logical().ReadWithContext(ctx, fmt.Sprintf("identity/entity/id/%s", entityId))
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, nil
	}

	raw, ok := resp.Data["mfa_secrets"].(map[string]any)
	if !ok {
		return nil, nil
	}

	methodIds := make([]string, 0, len(raw))
	for methodId := range raw {
		methodIds = append(methodIds, methodId)
	}
	return methodIds, nil
}

func (c *VaultClient) IdentityListEntities(ctx context.Context) ([]string, error) {
	path := "identity/entity/name"

	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	var keys []string
	for _, v := range secret.Data["keys"].([]any) {
		if s, ok := v.(string); ok {
			keys = append(keys, s)
		}
	}

	return keys, nil
}

var ErrVaultEmptyDataReturned = errors.New("no data returned from Vault")

func (c *VaultClient) IdentityGetEntityIdByName(ctx context.Context, entityName string) (string, error) {
	entity, err := c.IdentityGetEntityByName(ctx, entityName)
	if err != nil {
		return "", err
	}

	if entity == nil {
		return "", ErrVaultEmptyDataReturned
	}

	raw, ok := entity["id"]
	if !ok {
		return "", errors.New("no id contained in response")
	}

	return raw.(string), nil
}

func (c *VaultClient) IdentityGetEntityByName(ctx context.Context, entityName string) (map[string]any, error) {
	path := fmt.Sprintf("identity/entity/name/%s", entityName)

	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, ErrVaultEmptyDataReturned
	}

	return secret.Data, nil
}

func (c *VaultClient) IdentityListGroups(ctx context.Context) ([]string, error) {
	path := "identity/group/name"

	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	var keys []string
	for _, v := range secret.Data["keys"].([]any) {
		if s, ok := v.(string); ok {
			keys = append(keys, s)
		}
	}

	return keys, nil
}

func (c *VaultClient) SshSignKey(ctx context.Context, mount string, req SshSignatureRequest) (string, error) {
	data := convertUserKeyRequest(req)

	path := fmt.Sprintf("%s/sign/%s", mount, req.VaultRole)
	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		// TODO
		//var respErr *api.ResponseError
		//if errors.As(err, &respErr) && !shouldRetry(respErr.StatusCode) {
		//	return "", backoff.Permanent(err)
		//}
		return "", err
	}

	return fmt.Sprintf("%s", secret.Data["signed_key"]), nil
}

func convertUserKeyRequest(req SshSignatureRequest) map[string]any {
	data := map[string]interface{}{
		"public_key": req.PublicKey,
		"cert_type":  "user",
	}

	if len(req.Ttl) > 0 {
		data["ttl"] = req.Ttl
	}

	if len(req.Principals) > 0 {
		data["valid_principals"] = strings.Join(req.Principals, ",")
	}

	return data
}

// PkiListRoles returns the names of all roles of the PKI secret engine mounted at mount.
func (c *VaultClient) PkiListRoles(ctx context.Context, mount string) ([]string, error) {
	path := fmt.Sprintf("%s/roles", mount)
	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	rawKeys, ok := secret.Data["keys"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response structure: %#v", secret.Data)
	}

	var keys []string
	for _, v := range rawKeys {
		if s, ok := v.(string); ok {
			keys = append(keys, s)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// PkiReadRole reads the configuration of the given role of the PKI secret engine mounted at mount.
func (c *VaultClient) PkiReadRole(ctx context.Context, mount, role string) (*PkiRole, error) {
	path := fmt.Sprintf("%s/roles/%s", mount, role)
	secret, err := c.client.Logical().ReadWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read role: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("role %q not found", role)
	}

	ret := &PkiRole{Name: role}
	if allowedDomains, ok := secret.Data["allowed_domains"].([]any); ok {
		for _, v := range allowedDomains {
			if domain, ok := v.(string); ok && domain != "" {
				ret.AllowedDomains = append(ret.AllowedDomains, domain)
			}
		}
	}
	ret.AllowBareDomains, _ = secret.Data["allow_bare_domains"].(bool)
	ret.AllowSubdomains, _ = secret.Data["allow_subdomains"].(bool)
	ret.AllowLocalhost, _ = secret.Data["allow_localhost"].(bool)
	ret.AllowAnyName, _ = secret.Data["allow_any_name"].(bool)
	ret.AllowGlobDomains, _ = secret.Data["allow_glob_domains"].(bool)
	ret.Ttl = parseVaultDuration(secret.Data["ttl"])
	ret.MaxTtl = parseVaultDuration(secret.Data["max_ttl"])

	return ret, nil
}

func (c *VaultClient) PkiIssueCert(ctx context.Context, mount string, args PkiIssueArgs) (*vault.Secret, error) {
	path := fmt.Sprintf("%s/issue/%s", mount, args.Role)
	data := buildIssueRequestArgs(args)

	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		var respErr *vault.ResponseError
		if errors.As(err, &respErr) && !shouldRetry(respErr.StatusCode) {
			return nil, err
		}

		return nil, fmt.Errorf("could not issue certificate: %w", err)
	}

	return secret, nil
}

func (c *VaultClient) PkiVerifyCert(ctx context.Context, mount string, cert *x509.Certificate) error {
	caData, err := c.PkiGetCaChain(ctx, mount)
	if err != nil {
		return err
	}

	caBlock, _ := pem.Decode(caData)
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return err
	}

	return backend.VerifyCertAgainstCa(cert, ca)
}

func (c *VaultClient) PkiGetCa(ctx context.Context, mount string, binary bool) ([]byte, error) {
	path := fmt.Sprintf("%s/ca", mount)
	if !binary {
		path = path + "/pem"
	}

	return c.readRaw(ctx, path)
}

func (c *VaultClient) PkiGetCaChain(ctx context.Context, mount string) ([]byte, error) {
	path := fmt.Sprintf("/%s/ca_chain", mount)
	return c.readRaw(ctx, path)
}

func (c *VaultClient) PkiGetCrl(ctx context.Context, mount string, binary bool) ([]byte, error) {
	path := fmt.Sprintf("%s/crl", mount)
	if !binary {
		path += "/pem"
	}

	return c.readRaw(ctx, path)
}

func (c *VaultClient) readRaw(ctx context.Context, path string) ([]byte, error) {
	secret, err := c.client.Logical().ReadRawWithContext(ctx, path)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(secret.Body)
}

func (c *VaultClient) SshGetCa(ctx context.Context, mount string) ([]byte, error) {
	path := fmt.Sprintf("%s/public_key", mount)
	resp, err := c.client.Logical().ReadRawWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("reading cert failed: %v", err)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read body from response: %v", err)
	}

	return data, nil
}

// notReadable returns a synthetic ErrKindNotFound when the KVSecret describes
// a version Vault won't actually serve data for. Vault's data endpoint
// returns a successful response (no HTTP error) for a soft-deleted or
// destroyed version, with `data: null` and `metadata.deletion_time`/`destroyed`
// set. Without this check, we hand the caller an empty *KVSecret that
// looks superficially like a valid secret with no key/value pairs.
func notReadable(sec *vault.KVSecret, op string) error {
	if sec == nil || sec.VersionMetadata == nil {
		return nil
	}
	vm := sec.VersionMetadata
	switch {
	case vm.Destroyed:
		return &VaultError{
			Kind:    ErrKindNotFound,
			Op:      op,
			Message: fmt.Sprintf("v%d is destroyed", vm.Version),
		}
	case !vm.DeletionTime.IsZero():
		return &VaultError{
			Kind:    ErrKindNotFound,
			Op:      op,
			Message: fmt.Sprintf("v%d is soft-deleted", vm.Version),
		}
	}
	return nil
}

func (c *VaultClient) Get(ctx context.Context, mount, path string) (*Secret, error) {
	sec, err := c.client.KVv2(mount).Get(ctx, path)
	if err != nil {
		return nil, classify(err, "get")
	}
	if err := notReadable(sec, "get"); err != nil {
		return nil, err
	}
	return fromKVSecret(sec), nil
}

func (c *VaultClient) GetVersion(ctx context.Context, mount, path string, version int) (*Secret, error) {
	sec, err := c.client.KVv2(mount).GetVersion(ctx, path, version)
	if err != nil {
		return nil, classify(err, "get-version")
	}
	if err := notReadable(sec, "get-version"); err != nil {
		return nil, err
	}
	return fromKVSecret(sec), nil
}

func (c *VaultClient) GetMetadata(ctx context.Context, mount, path string) (*SecretMetadata, error) {
	md, err := c.client.KVv2(mount).GetMetadata(ctx, path)
	if err != nil {
		return nil, classify(err, "get-metadata")
	}
	if md == nil {
		return nil, &VaultError{Kind: ErrKindNotFound, Op: "get-metadata"}
	}
	out := &SecretMetadata{
		CurrentVersion: md.CurrentVersion,
		OldestVersion:  md.OldestVersion,
		Versions:       make([]VersionInfo, 0, len(md.Versions)),
	}
	for _, vm := range md.Versions {
		out.Versions = append(out.Versions, VersionInfo{
			Version:   vm.Version,
			CreatedAt: vm.CreatedTime,
			DeletedAt: vm.DeletionTime,
			Destroyed: vm.Destroyed,
		})
	}
	sort.Slice(out.Versions, func(i, j int) bool {
		return out.Versions[i].Version > out.Versions[j].Version
	})
	return out, nil
}

func (c *VaultClient) Put(ctx context.Context, mount, path string, data map[string]string, cas int) (*Secret, error) {
	payload := make(map[string]any, len(data))
	for k, val := range data {
		payload[k] = val
	}
	sec, err := c.client.KVv2(mount).Put(ctx, path, payload, vault.WithCheckAndSet(cas))
	if err != nil {
		return nil, classify(err, "put")
	}
	return fromKVSecret(sec), nil
}

func (c *VaultClient) SoftDelete(ctx context.Context, mount, path string) error {
	if err := c.client.KVv2(mount).Delete(ctx, path); err != nil {
		return classify(err, "delete")
	}
	return nil
}

func (c *VaultClient) List(ctx context.Context, mount, prefix string) ([]Entry, error) {
	p := strings.TrimRight(mount, "/") + "/metadata/" + prefix
	sec, err := c.client.Logical().ListWithContext(ctx, p)
	if err != nil {
		return nil, classify(err, "list")
	}
	if sec == nil || sec.Data == nil {
		return nil, nil
	}
	keys, _ := sec.Data["keys"].([]any)
	out := make([]Entry, 0, len(keys))
	for _, k := range keys {
		s, ok := k.(string)
		if !ok {
			continue
		}
		if strings.HasSuffix(s, "/") {
			out = append(out, Entry{Name: strings.TrimSuffix(s, "/"), IsDir: true})
		} else {
			out = append(out, Entry{Name: s, IsDir: false})
		}
	}
	return out, nil
}

// fromKVSecret converts a Vault KVSecret into the neutral Secret type.
func fromKVSecret(sec *vault.KVSecret) *Secret {
	if sec == nil {
		return nil
	}
	out := &Secret{Data: make(map[string]string, len(sec.Data))}
	for k, v := range sec.Data {
		out.Data[k] = fmt.Sprintf("%v", v)
	}
	if sec.VersionMetadata != nil {
		out.Version = sec.VersionMetadata.Version
		out.UpdatedAt = sec.VersionMetadata.CreatedTime
	}
	return out
}

// classify converts a Vault-flavoured error into a *VaultError so the
// rest of the package (and the UI) doesn't need to know about Vault types.
func classify(err error, op string) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, vault.ErrSecretNotFound) {
		return &VaultError{Kind: ErrKindNotFound, Op: op, Cause: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &VaultError{Kind: ErrKindTimeout, Op: op, Message: "request timed out", Cause: err}
	}
	if nerr, ok := errors.AsType[net.Error](err); ok {
		return &VaultError{Kind: ErrKindNetwork, Op: op, Message: nerr.Error(), Cause: err}
	}
	if rerr, ok := errors.AsType[*vault.ResponseError](err); ok {
		if rerr.StatusCode == 400 {
			for _, e := range rerr.Errors {
				le := strings.ToLower(e)
				if strings.Contains(le, "check-and-set") || strings.Contains(le, "did not match the current version") {
					return &VaultError{Kind: ErrKindCASConflict, Op: op, Message: e, Cause: err}
				}
			}
		}
		kind := ErrKindUnknown
		switch rerr.StatusCode {
		case 400:
			kind = ErrKindBadRequest
		case 403:
			kind = ErrKindPermissionDenied
		case 404:
			kind = ErrKindNotFound
		case 429:
			kind = ErrKindRateLimited
		case 500, 502, 503, 504:
			kind = ErrKindServerError
		}
		return &VaultError{Kind: kind, Op: op, Message: strings.Join(rerr.Errors, "; "), Cause: err}
	}
	return &VaultError{Kind: ErrKindUnknown, Op: op, Message: err.Error(), Cause: err}
}

func buildIssueRequestArgs(args PkiIssueArgs) map[string]any {
	data := map[string]any{
		"common_name": args.CommonName,
		"ttl":         args.Ttl,
		"format":      "pem",
		"ip_sans":     strings.Join(args.IpSans, ","),
		"alt_names":   strings.Join(args.AltNames, ","),
	}

	return data
}

func shouldRetry(statusCode int) bool {
	switch statusCode {
	case 400, // Bad Request
		401, // Unauthorized
		403, // Forbidden
		404, // Not Found
		405, // Method Not Allowed
		406, // Not Acceptable
		407, // Proxy Authentication Required
		409, // Conflict
		410, // Gone
		411, // Length Required
		412, // Precondition Failed
		413, // Payload Too Large
		414, // URI Too Long
		415, // Unsupported Media Type
		416, // Range Not Satisfiable
		417, // Expectation Failed
		418, // I'm a Teapot
		421, // Misdirected Request
		422, // Unprocessable Entity
		423, // Locked (WebDAV)
		424, // Failed Dependency (WebDAV)
		425, // Too Early
		426, // Upgrade Required
		428, // Precondition Required
		429, // Too Many Requests
		431, // Request Header Fields Too Large
		451: // Unavailable For Legal Reasons
		return false
	default:
		return true
	}
}

// ---------------------------------------------------------------------------
// error humanization
// ---------------------------------------------------------------------------

func HumanizeError(err error, op string) string {
	if err == nil {
		return ""
	}
	var se *VaultError
	if errors.As(err, &se) {
		switch se.Kind {
		case ErrKindCASConflict:
			return "version conflict — the secret changed on the server since you loaded it. " +
				"Press 'r' to reload (your local edits will be lost)."
		case ErrKindNotFound:
			return fmt.Sprintf("%s: not found on server", op)
		case ErrKindPermissionDenied:
			msg := se.Message
			if msg == "" {
				msg = "your token lacks permission for this operation"
			}
			return fmt.Sprintf("%s denied: %s", op, msg)
		case ErrKindRateLimited:
			return fmt.Sprintf("%s rate-limited — retry in a moment", op)
		case ErrKindServerError:
			return fmt.Sprintf("%s failed: backend error. %s", op, se.Message)
		case ErrKindTimeout:
			return fmt.Sprintf("%s timed out", op)
		case ErrKindNetwork:
			return fmt.Sprintf("%s network error: %s", op, se.Message)
		case ErrKindBadRequest:
			return fmt.Sprintf("%s rejected: %s", op, se.Message)
		case ErrKindUnknown:
			return fmt.Sprintf("%s failed: %s", op, se.Message)
		}
	}
	return fmt.Sprintf("%s failed: %v", op, err)
}

// ListMountsByType returns the paths (without trailing slash) of all mounted secret engines of the given type,
// e.g. "transit" or "ssh". As reading sys/mounts requires privileges that are often not granted, it falls back
// to sys/internal/ui/mounts, which lists the mounts the token has access to.
func (c *VaultClient) ListMountsByType(ctx context.Context, engineType string) ([]string, error) {
	mountTypes, err := c.listMountTypes(ctx)
	if err != nil {
		var uiErr error
		mountTypes, uiErr = c.listUiMountTypes(ctx)
		if uiErr != nil {
			return nil, fmt.Errorf("failed to list mounts: %w", errors.Join(err, uiErr))
		}
		log.Debug().Err(err).Msg("could not read sys/mounts, used sys/internal/ui/mounts instead")
	}

	var matchingMounts []string
	for path, mountType := range mountTypes {
		if mountType == engineType {
			matchingMounts = append(matchingMounts, strings.TrimSuffix(path, "/"))
		}
	}

	sort.Strings(matchingMounts)
	return matchingMounts, nil
}

// listMountTypes returns a map of mount path to secret engine type, read from sys/mounts.
func (c *VaultClient) listMountTypes(ctx context.Context) (map[string]string, error) {
	mounts, err := c.client.Sys().ListMountsWithContext(ctx)
	if err != nil {
		return nil, err
	}

	ret := make(map[string]string, len(mounts))
	for path, mount := range mounts {
		if mount != nil {
			ret[path] = mount.Type
		}
	}
	return ret, nil
}

// listUiMountTypes returns a map of mount path to secret engine type, read from sys/internal/ui/mounts.
func (c *VaultClient) listUiMountTypes(ctx context.Context) (map[string]string, error) {
	secret, err := c.client.Logical().ReadWithContext(ctx, "sys/internal/ui/mounts")
	if err != nil {
		return nil, err
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	mounts, ok := secret.Data["secret"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response structure: %#v", secret.Data)
	}

	ret := make(map[string]string, len(mounts))
	for path, raw := range mounts {
		if mount, ok := raw.(map[string]any); ok {
			if mountType, ok := mount["type"].(string); ok {
				ret[path] = mountType
			}
		}
	}
	return ret, nil
}

// Capabilities returns the capabilities of the current token for each of the given paths, e.g. "update" or "deny".
func (c *VaultClient) Capabilities(ctx context.Context, paths []string) (map[string][]string, error) {
	if len(paths) == 0 {
		return map[string][]string{}, nil
	}

	secret, err := c.client.Logical().WriteWithContext(ctx, "sys/capabilities-self", map[string]any{
		"paths": paths,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to lookup capabilities: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	ret := make(map[string][]string, len(paths))
	for _, path := range paths {
		rawCapabilities, ok := secret.Data[path].([]any)
		if !ok {
			return nil, fmt.Errorf("no capabilities returned for path %q", path)
		}

		for _, v := range rawCapabilities {
			if capability, ok := v.(string); ok {
				ret[path] = append(ret[path], capability)
			}
		}
	}

	return ret, nil
}

// TransitListKeys returns the names of all keys of the transit secret engine mounted at mount.
func (c *VaultClient) TransitListKeys(ctx context.Context, mount string) ([]string, error) {
	path := fmt.Sprintf("%s/keys", mount)
	secret, err := c.client.Logical().ListWithContext(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to list transit keys: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	rawKeys, ok := secret.Data["keys"].([]any)
	if !ok {
		return nil, fmt.Errorf("unexpected response structure: %#v", secret.Data)
	}

	var keys []string
	for _, v := range rawKeys {
		if s, ok := v.(string); ok {
			keys = append(keys, s)
		}
	}

	sort.Strings(keys)
	return keys, nil
}

// TransitEncrypt encrypts plaintext using the transit key keyName of the transit secret engine mounted at mount
// and returns the ciphertext, e.g. "vault:v1:...". If the key has derivation enabled, keyContext must be supplied.
func (c *VaultClient) TransitEncrypt(ctx context.Context, mount, keyName string, plaintext []byte, keyContext []byte) (string, error) {
	data := map[string]any{
		"plaintext": base64.StdEncoding.EncodeToString(plaintext),
	}
	if len(keyContext) > 0 {
		data["context"] = base64.StdEncoding.EncodeToString(keyContext)
	}

	path := fmt.Sprintf("%s/encrypt/%s", mount, keyName)
	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return "", fmt.Errorf("no data returned from Vault")
	}

	ciphertext, ok := secret.Data["ciphertext"].(string)
	if !ok || ciphertext == "" {
		return "", fmt.Errorf("unexpected response structure: %#v", secret.Data)
	}

	return ciphertext, nil
}

// TransitDecrypt decrypts ciphertext using the transit key keyName of the transit secret engine mounted at mount
// and returns the plaintext. If the key has derivation enabled, keyContext must be supplied.
func (c *VaultClient) TransitDecrypt(ctx context.Context, mount, keyName, ciphertext string, keyContext []byte) ([]byte, error) {
	data := map[string]any{
		"ciphertext": ciphertext,
	}
	if len(keyContext) > 0 {
		data["context"] = base64.StdEncoding.EncodeToString(keyContext)
	}

	path := fmt.Sprintf("%s/decrypt/%s", mount, keyName)
	secret, err := c.client.Logical().WriteWithContext(ctx, path, data)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	if secret == nil || secret.Data == nil {
		return nil, fmt.Errorf("no data returned from Vault")
	}

	encoded, ok := secret.Data["plaintext"].(string)
	if !ok {
		return nil, fmt.Errorf("unexpected response structure: %#v", secret.Data)
	}

	plaintext, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("could not decode plaintext: %w", err)
	}

	return plaintext, nil
}

// ErrPolicyNotFound is returned if the requested ACL policy does not exist.
var ErrPolicyNotFound = errors.New("policy not found")

// PolicyList returns the names of all ACL policies, sorted alphabetically.
func (c *VaultClient) PolicyList(ctx context.Context) ([]string, error) {
	policies, err := c.client.Sys().ListPoliciesWithContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list policies: %w", err)
	}

	sort.Strings(policies)
	return policies, nil
}

// rootPolicyRules is displayed for the built-in root policy, which can not be read as it has no rules.
const rootPolicyRules = "# The root policy is built into Vault and grants all capabilities on all paths.\n# It can not be read or modified."

// PolicyRead returns the rules of the ACL policy with the given name.
func (c *VaultClient) PolicyRead(ctx context.Context, name string) (string, error) {
	if name == "root" {
		return rootPolicyRules, nil
	}

	rules, err := c.client.Sys().GetPolicyWithContext(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to read policy %q: %w", name, err)
	}

	// the client returns an empty policy without error if vault responds with 404
	if rules == "" {
		return "", fmt.Errorf("%w: %q", ErrPolicyNotFound, name)
	}

	return rules, nil
}
