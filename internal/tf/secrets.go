package tf

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// cksum is POSIX cksum(1), used to name the lock directory.
func cksum(data []byte) uint32 {
	var table [256]uint32
	for i := range table {
		c := uint32(i) << 24
		for range 8 {
			if c&0x80000000 != 0 {
				c = c<<1 ^ 0x04c11db7
			} else {
				c <<= 1
			}
		}
		table[i] = c
	}
	var crc uint32
	step := func(b byte) { crc = crc<<8 ^ table[byte(crc>>24)^b] }
	for _, b := range data {
		step(b)
	}
	for n := len(data); n > 0; n >>= 8 {
		step(byte(n))
	}
	return ^crc
}

// Only one run per working directory at a time. The remote state has its own
// lock, but that one is taken late and does not cover the sops decryption or a
// plan file being overwritten halfway through. The name is the one the
// repository's own run.go uses, so the two keep out of each other's way.
func (r *runner) acquireLock() {
	scope := r.env + "-" + r.stack
	tmp := os.Getenv("TMPDIR")
	if tmp == "" {
		tmp = "/tmp"
	}
	// The scope is made of names checked against nameRE, so the lock stays
	// directly below TMPDIR.
	candidate := filepath.Join(tmp, fmt.Sprintf("tf-run-%d-%s.lock", cksum([]byte(r.repoRoot)), scope))

	if err := os.Mkdir(candidate, 0o700); err != nil { //nolint:gosec // G703: see above
		if !errors.Is(err, fs.ErrExist) {
			die("could not acquire lock %s: %v", candidate, err)
		}
		data, _ := os.ReadFile(filepath.Join(candidate, "pid")) //nolint:gosec // G703: see above
		if owner, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && owner > 0 {
			if err := syscall.Kill(owner, 0); err == nil || errors.Is(err, syscall.EPERM) {
				die("another run for '%s' is in progress (pid %d)", scope, owner)
			}
		}
		warnf("removing stale lock %s", candidate)
		_ = os.RemoveAll(candidate)                        //nolint:gosec // G703: see above
		if err := os.Mkdir(candidate, 0o700); err != nil { //nolint:gosec // G703: see above
			die("could not acquire lock %s", candidate)
		}
	}

	cleanup.Lock()
	cleanup.lock = candidate
	cleanup.Unlock()
	_ = os.WriteFile(filepath.Join(candidate, "pid"), []byte(fmt.Sprintf("%d\n", os.Getpid())), 0o600) //nolint:gosec // G703: see above
}

func (r *runner) secretString(key string) string {
	v, ok := r.secrets[key]
	if !ok || v == nil {
		return ""
	}
	s, ok := v.(string)
	if !ok {
		die("%s in %s is not a string", key, r.rel(r.secretsFile))
	}
	return s
}

func (r *runner) readSecrets() {
	cmd := exec.Command("sops", "--decrypt", "--output-type", "json", r.secretsFile) //nolint:gosec // G204: a path below the repository
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	out, err := cmd.Output()
	if err != nil {
		die("could not decrypt %s", r.rel(r.secretsFile))
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.UseNumber()
	if err := dec.Decode(&r.secrets); err != nil || r.secrets == nil {
		die("%s does not decrypt to an object", r.rel(r.secretsFile))
	}

	if r.secretString("passphrase") == "" {
		die("passphrase in %s is empty", r.rel(r.secretsFile))
	}

	r.keyID = r.secretString("key_id")
	if r.keyID == "" {
		r.keyID = defaultKeyID
	}
	r.previousKeyID = r.secretString("previous_key_id")
	hasPrevious := r.secretString("previous_passphrase") != ""

	// Both become HCL identifiers.
	for _, id := range []string{r.keyID, r.previousKeyID} {
		if id != "" && !keyIDRE.MatchString(id) {
			die("invalid key id '%s' in %s", id, r.rel(r.secretsFile))
		}
	}
	if (r.previousKeyID != "") != hasPrevious {
		die("previous_key_id and previous_passphrase in %s go together", r.rel(r.secretsFile))
	}
	if r.keyID == r.previousKeyID {
		die("key_id and previous_key_id in %s must differ", r.rel(r.secretsFile))
	}
}

// hclString turns s into an HCL string literal. A JSON string is one already,
// which takes care of quotes, backslashes and newlines; HCL additionally reads
// '${' and '%{' as the start of a template, so those are escaped on top.
func hclString(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	lit := strings.TrimSuffix(buf.String(), "\n")
	lit = strings.ReplaceAll(lit, "${", "$${")
	return strings.ReplaceAll(lit, "%{", "%%{")
}

// buildTFEncryption declares the key provider and the 'default' method the
// stacks' 'encryption' blocks refer to. OpenTofu records the key provider's
// name in the state, so a state written with the previous passphrase can only
// be read through a key provider of the same name: while a rotation is in
// progress, the state and plan blocks are overridden with a fallback to it.
func (r *runner) buildTFEncryption(withPrevious bool) string {
	provider := func(id, passphrase string) string {
		return fmt.Sprintf("key_provider \"pbkdf2\" %q {\n  passphrase = %s\n}\n", id, hclString(passphrase))
	}
	method := func(name, id string) string {
		return fmt.Sprintf("method \"aes_gcm\" %q {\n  keys = key_provider.pbkdf2.%s\n}\n", name, id)
	}
	withFallback := func(block string) string {
		return block + " {\n  enforced = true\n  method   = method.aes_gcm.default\n\n" +
			"  fallback {\n    method = method.aes_gcm.previous\n  }\n}\n"
	}

	out := provider(r.keyID, r.secretString("passphrase")) + method("default", r.keyID)
	if previous := r.secretString("previous_passphrase"); withPrevious && previous != "" {
		out += provider(r.previousKeyID, previous) + method("previous", r.previousKeyID) +
			withFallback("state") + withFallback("plan")
	}
	return out
}

// writeSecrets encrypts secrets into the secrets file. The temporary file next
// to it only ever holds ciphertext.
func (r *runner) writeSecrets(secrets map[string]any) {
	sopsConfig := filepath.Join(r.stacksDir, ".sops.yaml")
	if !fileExists(sopsConfig) {
		die("%s does not exist", r.rel(sopsConfig))
	}
	data, err := json.Marshal(secrets)
	if err != nil {
		die("could not encode the secrets: %v", err)
	}

	tmp := fmt.Sprintf("%s.tmp.%d", r.secretsFile, os.Getpid())
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		die("could not create %s: %v", r.rel(tmp), err)
	}
	cleanup.Lock()
	cleanup.secrets = tmp
	cleanup.Unlock()

	cmd := exec.Command("sops", "--config", sopsConfig, "--encrypt", "--filename-override", r.secretsFile,
		"--input-type", "json", "--output-type", "yaml", "/dev/stdin")
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = f
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	_ = f.Close()
	if err != nil {
		die("could not encrypt %s", r.rel(r.secretsFile))
	}
	if err := os.Rename(tmp, r.secretsFile); err != nil {
		die("could not replace %s: %v", r.rel(r.secretsFile), err)
	}

	cleanup.Lock()
	cleanup.secrets = ""
	cleanup.Unlock()
}

// newKey makes up a key id and a random passphrase for it.
func newKey() (keyID, passphrase string) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		die("could not generate a passphrase: %v", err)
	}
	return "key_" + time.Now().UTC().Format("20060102T150405Z"), base64.StdEncoding.EncodeToString(raw)
}

// writesState reports whether the operation may write state, which is then
// encrypted with the unit's passphrase.
func writesState(op string) bool {
	switch op {
	case "apply", "destroy", "refresh", "raw":
		return true
	}
	return false
}

// generateSecrets creates the passphrase file of a unit that encrypts its state
// but has none yet, and leaves the new passphrase in r.secrets.
//
// A missing file is only ever a new unit if the unit also has no state. If it
// has state, the file was lost or this is the wrong checkout, and a fresh
// passphrase would hide the real problem behind a state nobody can decrypt.
// Asking the backend with the fresh passphrase tells the two apart without
// knowing anything about the backend: no state reads fine with any
// passphrase, encrypted state fails to decrypt. Plaintext state reads fine too,
// and that is the one case where a new passphrase is right, as OpenTofu
// encrypts it on the next write.
func (r *runner) generateSecrets() {
	keyID, passphrase := newKey()
	r.secrets = map[string]any{"key_id": keyID, "passphrase": passphrase}
	r.keyID = keyID

	logf("%s does not exist, checking that '%s/%s' has no state yet ...", r.rel(r.secretsFile), r.stack, r.env)
	_ = os.Setenv("TF_ENCRYPTION", r.buildTFEncryption(false))
	if r.runTerragrunt(io.Discard, "state", "pull") != 0 {
		die("could not read the state of '%s/%s' with a new passphrase, see above. Either it has state encrypted "+
			"with a passphrase whose %s is missing: restore the file from git instead of generating a new one. "+
			"Or the backend could not be reached: fix that and run again.", r.stack, r.env, secretsName)
	}

	r.writeSecrets(r.secrets)
	logf("Generated %s with key '%s'. Commit it before applying: without it the state cannot be read.",
		r.rel(r.secretsFile), keyID)
}

// requireCommitted stops an operation that writes state while the passphrase
// exists only in the working tree, where losing it loses the state with it.
// A file that changed since the last commit counts as not committed, which
// also covers a rotate-passphrase-start that has not been committed yet.
func (r *runner) requireCommitted() {
	rel, err := filepath.Rel(r.repoRoot, r.secretsFile)
	if err != nil {
		die("%s is outside the repository", r.secretsFile)
	}
	tracked := exec.Command("git", "-C", r.repoRoot, "ls-files", "--error-unmatch", "--", rel).Run() == nil //nolint:gosec // G204: a path below the repository
	status, err := exec.Command("git", "-C", r.repoRoot, "status", "--porcelain", "--", rel).Output()       //nolint:gosec // G204: a path below the repository
	if err != nil {
		die("could not ask git about %s: %v", rel, err)
	}
	if !tracked || len(bytes.TrimSpace(status)) > 0 {
		die("%s is not committed. '%s' writes state encrypted with it, and without the file that state cannot "+
			"be read: commit it, then run again.", rel, r.operation)
	}
}
