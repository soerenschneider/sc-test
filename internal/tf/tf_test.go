package tf

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSplitArgs(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    []string
		wantErr bool
	}{
		{name: "empty", in: "", want: nil},
		{name: "plain words", in: "state  list", want: []string{"state", "list"}},
		{name: "single quotes", in: `import 'module.buckets["x"]' x`, want: []string{"import", `module.buckets["x"]`, "x"}},
		{name: "double quotes with escapes", in: `"a \"b\" \$c" d`, want: []string{`a "b" $c`, "d"}},
		{name: "backslash escapes a space", in: `a\ b`, want: []string{"a b"}},
		{name: "empty quoted argument", in: `'' x`, want: []string{"", "x"}},
		{name: "unterminated single quote", in: `'abc`, wantErr: true},
		{name: "unterminated double quote", in: `"abc`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := splitArgs(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("splitArgs(%q) error = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitArgs(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestShellQuoteRoundTrip(t *testing.T) {
	for _, s := range []string{"plain", "-target=module.x", `module.buckets["x"]`, "it's", "a b", ""} {
		quoted := shellQuote(s)
		got, err := splitArgs(quoted)
		if err != nil || len(got) != 1 || got[0] != s {
			t.Errorf("shellQuote(%q) = %s, which splits into %q (%v)", s, quoted, got, err)
		}
	}
}

func TestStripHCLComments(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want bool // whether an encryption block remains
	}{
		{name: "active block", src: "terraform {\n  encryption {\n  }\n}\n", want: true},
		{name: "hash comment", src: "# encryption {\n", want: false},
		{name: "slash comment", src: "// encryption {\n", want: false},
		{name: "block comment", src: "/*\n  encryption {\n  }\n*/\n", want: false},
		{name: "comment marker inside a string", src: "x = \"#\"\nencryption {\n", want: true},
		{name: "block in a string only", src: "x = \"encryption {\"\n", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encryptionRE.Match(stripHCLComments([]byte(tt.src))); got != tt.want {
				t.Errorf("encryption block found = %v, want %v in:\n%s", got, tt.want, tt.src)
			}
		})
	}
}

func TestHCLString(t *testing.T) {
	tests := map[string]string{
		"abc":        `"abc"`,
		`a"b`:        `"a\"b"`,
		"a\\b":       `"a\\b"`,
		"${x}":       `"$${x}"`,
		"%{x}":       `"%%{x}"`,
		"<&>":        `"<&>"`,
		"line\nnext": `"line\nnext"`,
	}
	for in, want := range tests {
		if got := hclString(in); got != want {
			t.Errorf("hclString(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestCksum(t *testing.T) {
	// Reference values from POSIX cksum(1).
	tests := map[string]uint32{
		"":          4294967295,
		"123456789": 930766865,
	}
	for in, want := range tests {
		if got := cksum([]byte(in)); got != want {
			t.Errorf("cksum(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestBuildTFEncryption(t *testing.T) {
	r := &runner{keyID: "key_new", previousKeyID: "key_old", secrets: map[string]any{
		"passphrase":          "new",
		"previous_passphrase": "old",
	}}

	plain := r.buildTFEncryption(false)
	if strings.Contains(plain, "key_old") || strings.Contains(plain, "fallback") {
		t.Errorf("without the previous passphrase, got:\n%s", plain)
	}
	if !strings.Contains(plain, `key_provider "pbkdf2" "key_new"`) || !strings.Contains(plain, `method "aes_gcm" "default"`) {
		t.Errorf("missing the current key provider or the default method:\n%s", plain)
	}

	rotating := r.buildTFEncryption(true)
	for _, want := range []string{`key_provider "pbkdf2" "key_old"`, `method "aes_gcm" "previous"`, "state {", "plan {", "fallback {"} {
		if !strings.Contains(rotating, want) {
			t.Errorf("during a rotation, missing %q in:\n%s", want, rotating)
		}
	}
}

func TestHintAndCommandLine(t *testing.T) {
	r := &runner{env: "prod", stack: "aws-s3", operation: "output", outputName: "user credentials", passthrough: []string{"-no-color"}}
	if got, want := r.hint("apply"), "sc tf apply -e prod -s aws-s3"; got != want {
		t.Errorf("hint = %q, want %q", got, want)
	}
	if got, want := r.commandLine(), "sc tf output -e prod -s aws-s3 -n 'user credentials' -- -no-color"; got != want {
		t.Errorf("commandLine = %q, want %q", got, want)
	}
}

func TestReadContract(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, contractName)

	if _, err := readContract(path); err == nil {
		t.Error("a missing file should be an error")
	}

	if err := os.WriteFile(path, []byte("1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if v, err := readContract(path); err != nil || v != 1 {
		t.Errorf("readContract = %d, %v, want 1", v, err)
	}

	if err := os.WriteFile(path, []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readContract(path); err == nil {
		t.Error("a non-integer should be an error")
	}
}

func TestChangesInfrastructure(t *testing.T) {
	for op, want := range map[string]bool{
		"apply": true, "destroy": true, "refresh": true, "raw": true,
		"rotate-passphrase-start": true, "rotate-passphrase-finish": true,
		"plan": false, "plan-destroy": false, "output": false, "show": false,
		"validate": false, "init": false, "fmt": false, "test": false,
	} {
		if got := changesInfrastructure(op); got != want {
			t.Errorf("changesInfrastructure(%q) = %v, want %v", op, got, want)
		}
	}
}

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:gosec // G204: test helper
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGitFreshness(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")

	root := t.TempDir()
	remote := filepath.Join(root, "remote.git")
	a := filepath.Join(root, "a")
	b := filepath.Join(root, "b")

	git(t, root, "init", "--quiet", "--bare", "--initial-branch=main", remote)
	git(t, root, "clone", "--quiet", remote, a)
	git(t, a, "commit", "--quiet", "--allow-empty", "-m", "first")
	git(t, a, "push", "--quiet", "origin", "HEAD:main")
	git(t, root, "clone", "--quiet", remote, b)

	if _, behind, ahead, err := gitFreshness(b); err != nil || behind != 0 || ahead != 0 {
		t.Fatalf("fresh clone: behind=%d ahead=%d err=%v, want 0 0 nil", behind, ahead, err)
	}

	git(t, a, "commit", "--quiet", "--allow-empty", "-m", "second")
	git(t, a, "push", "--quiet", "origin", "HEAD:main")
	upstream, behind, ahead, err := gitFreshness(b)
	if err != nil || behind != 1 || ahead != 0 || upstream != "origin/main" {
		t.Fatalf("after a push elsewhere: upstream=%s behind=%d ahead=%d err=%v, want origin/main 1 0 nil", upstream, behind, ahead, err)
	}

	git(t, b, "pull", "--quiet", "--ff-only")
	git(t, b, "commit", "--quiet", "--allow-empty", "-m", "local")
	if _, behind, ahead, err := gitFreshness(b); err != nil || behind != 0 || ahead != 1 {
		t.Fatalf("with a local commit: behind=%d ahead=%d err=%v, want 0 1 nil", behind, ahead, err)
	}

	git(t, b, "checkout", "--quiet", "-b", "no-upstream")
	if _, _, _, err := gitFreshness(b); err == nil {
		t.Error("a branch without upstream should be an error")
	}

	git(t, b, "checkout", "--quiet", "main")
	if err := os.RemoveAll(remote); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := gitFreshness(b); err == nil || !strings.Contains(err.Error(), "could not fetch") {
		t.Errorf("an unreachable remote should fail the fetch, got %v", err)
	}
}
