package picker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

// FsTreeProvider exposes the local filesystem through TreeProvider. It does
// not sandbox, so the user can navigate anywhere.
type FsTreeProvider struct {
	// ShowHidden controls whether dotfiles (".bashrc", ".config") are
	// included. Defaults to false to keep the listing tidy; set to true
	// if you need to navigate into hidden config directories.
	ShowHidden bool
}

func (f *FsTreeProvider) Children(_ context.Context, prefix string) ([]TreeNode, error) {
	p := "/" + prefix // empty prefix → "/"; "etc/" → "/etc/"

	entries, err := os.ReadDir(p)
	if err != nil {
		return nil, err
	}
	out := make([]TreeNode, 0, len(entries))
	for _, e := range entries {
		name := e.Name()
		if !f.ShowHidden && strings.HasPrefix(name, ".") {
			continue
		}
		out = append(out, TreeNode{Name: name, IsDir: e.IsDir()})
	}
	return out, nil
}

// FsPrefixFromAbsPath converts an absolute path like "/home/alice/work" into
// the trailing-slash prefix the browser expects: "home/alice/work/".
func FsPrefixFromAbsPath(abs string) string {
	abs = filepath.Clean(abs)
	if abs == "/" {
		return ""
	}
	p := strings.TrimPrefix(abs, "/")
	if !strings.HasSuffix(p, "/") {
		p += "/"
	}
	return p
}

// FsAbsPath turns a (prefix, name) result back into an absolute path.
func FsAbsPath(prefix, name string) string {
	return "/" + prefix + name
}
