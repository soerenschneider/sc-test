package picker

import (
	"context"
	"time"

	"github.com/soerenschneider/sc/internal/vault"
)

type vaultKv2Browser interface {
	// List returns the immediate children of prefix. Prefix is "" for the
	// mount root; otherwise it ends with "/". Children that end in "/" are
	// represented with IsDir == true.
	List(ctx context.Context, mount, prefix string) ([]vault.Entry, error)
}

// VaultTreeProvider wraps a vaultKv2Browser.List behind the TreeProvider
// interface so the generic browser can walk a Vault KV v2 mount.
type VaultTreeProvider struct {
	store   vaultKv2Browser
	mount   string
	timeout time.Duration
}

func NewVaultTreeProvider(store vaultKv2Browser, mount string, timeout time.Duration) *VaultTreeProvider {
	return &VaultTreeProvider{store: store, mount: mount, timeout: timeout}
}

func (v *VaultTreeProvider) Children(ctx context.Context, prefix string) ([]TreeNode, error) {
	entries, err := v.store.List(ctx, v.mount, prefix)
	if err != nil {
		return nil, err
	}
	out := make([]TreeNode, len(entries))
	for i, e := range entries {
		out[i] = TreeNode{Name: e.Name, IsDir: e.IsDir}
	}
	return out, nil
}
