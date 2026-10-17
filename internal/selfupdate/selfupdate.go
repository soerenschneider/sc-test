package selfupdate

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"golang.org/x/mod/semver"
	"golang.org/x/sys/unix"
)

const (
	Repository = "github.com/soerenschneider/sc-test"

	binaryName    = "sc"
	checksumAsset = "checksum.sha256"

	versionPrefix = "# version: "
)

type Client interface {
	Do(r *http.Request) (*http.Response, error)
}

type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type Release struct {
	TagName string  `json:"tag_name"`
	Assets  []Asset `json:"assets"`
}

func (r *Release) asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

// LatestRelease fetches the latest release of Repository from GitHub.
func LatestRelease(ctx context.Context, client Client) (*Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", strings.TrimPrefix(Repository, "github.com/"))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch latest release: %s", resp.Status)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	return &release, nil
}

// IsNewer returns true if remote is a newer semantic version than local.
func IsNewer(local, remote string) (bool, error) {
	local, remote = canonical(local), canonical(remote)
	if !semver.IsValid(local) {
		return false, fmt.Errorf("local version %q is not a valid semantic version", local)
	}
	if !semver.IsValid(remote) {
		return false, fmt.Errorf("remote version %q is not a valid semantic version", remote)
	}
	return semver.Compare(local, remote) < 0, nil
}

func canonical(v string) string {
	if v != "" && !strings.HasPrefix(v, "v") {
		return "v" + v
	}
	return v
}

// AssetName returns the name of the release asset built for the given platform.
func AssetName(goos, goarch string) (string, error) {
	switch goos + "/" + goarch {
	case "linux/amd64":
		return binaryName + "-linux-amd64", nil
	case "linux/arm64":
		return binaryName + "-linux-aarch64", nil
	case "linux/arm":
		return binaryName + "-linux-armv6", nil
	case "darwin/arm64":
		return binaryName + "-darwin-arm64", nil
	}
	return "", fmt.Errorf("no release binaries are published for %s/%s", goos, goarch)
}

// Executable returns the resolved path of the running binary.
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// CheckWritable checks whether the binary at exe can be replaced. The binary
// is swapped by renaming a new file over it, so it is the directory that needs
// to be writable, not the file itself.
func CheckWritable(exe string) error {
	dir := filepath.Dir(exe)
	if err := unix.Access(dir, unix.W_OK); err != nil {
		return fmt.Errorf("directory %s is not writable: %w", dir, err)
	}
	return nil
}

// Update downloads the release binary for the current platform, verifies it
// against the release's checksum file, whose signature must be accepted by one
// of verifiers, and atomically replaces exe with it.
func Update(ctx context.Context, client Client, release *Release, exe string, verifiers []Verifier) error {
	name, err := AssetName(runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return err
	}
	binAsset, ok := release.asset(name)
	if !ok {
		return fmt.Errorf("release %s has no asset %s", release.TagName, name)
	}

	sums, err := fetchVerified(ctx, client, release, verifiers)
	if err != nil {
		return err
	}
	// The tag is not covered by the signature, so make sure the signed file
	// belongs to it. Otherwise an old signed release could be re-published
	// under a newer tag.
	version, err := parseVersion(bytes.NewReader(sums))
	if err != nil {
		return err
	}
	if canonical(version) != canonical(release.TagName) {
		return fmt.Errorf("signed checksums are for version %s, not %s", version, release.TagName)
	}
	wantSum, err := parseChecksum(bytes.NewReader(sums), name)
	if err != nil {
		return err
	}

	info, err := os.Stat(exe)
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(exe), "."+filepath.Base(exe)+".update-*")
	if err != nil {
		return err
	}
	// No-op once the rename succeeded.
	defer os.Remove(tmp.Name())

	gotSum, err := download(ctx, client, binAsset.BrowserDownloadURL, tmp)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}

	if gotSum != wantSum {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", name, wantSum, gotSum)
	}

	if err := os.Chmod(tmp.Name(), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), exe)
}

func get(ctx context.Context, client Client, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("failed to download %s: %s", url, resp.Status)
	}
	return resp.Body, nil
}

func download(ctx context.Context, client Client, url string, w io.Writer) (string, error) {
	body, err := get(ctx, client, url)
	if err != nil {
		return "", err
	}
	defer body.Close()

	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(w, h), body); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Large enough for checksum files and signatures, including sigstore bundles.
const maxChecksumSize = 64 * 1024

func fetchSmall(ctx context.Context, client Client, url string) ([]byte, error) {
	body, err := get(ctx, client, url)
	if err != nil {
		return nil, err
	}
	defer body.Close()

	data, err := io.ReadAll(io.LimitReader(body, maxChecksumSize+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxChecksumSize {
		return nil, fmt.Errorf("%s exceeds %d bytes", url, maxChecksumSize)
	}
	return data, nil
}

// fetchVerified downloads the checksum file of release and returns its content
// if one of verifiers accepts a signature published along with it. Verifiers
// whose signature asset is missing from the release are skipped.
func fetchVerified(ctx context.Context, client Client, release *Release, verifiers []Verifier) ([]byte, error) {
	if len(verifiers) == 0 {
		return nil, errors.New("no trusted signers configured")
	}
	sumAsset, ok := release.asset(checksumAsset)
	if !ok {
		return nil, fmt.Errorf("release %s has no asset %s", release.TagName, checksumAsset)
	}
	sums, err := fetchSmall(ctx, client, sumAsset.BrowserDownloadURL)
	if err != nil {
		return nil, err
	}

	var errs []error
	// Several verifiers may share a signature asset, e.g. both cosign ones.
	sigs := map[string][]byte{}
	for _, v := range verifiers {
		sigAsset, ok := release.asset(v.SignatureAsset())
		if !ok {
			continue
		}
		sig, fetched := sigs[sigAsset.Name]
		if !fetched {
			sig, err = fetchSmall(ctx, client, sigAsset.BrowserDownloadURL)
			if err == nil {
				sigs[sigAsset.Name] = sig
			}
		}
		if err == nil {
			err = v.Verify(ctx, sums, sig)
		}
		if err == nil {
			return sums, nil
		}
		errs = append(errs, fmt.Errorf("%s: %w", v.Name(), err))
	}
	if len(errs) == 0 {
		return nil, fmt.Errorf("release %s has no signature of %s for any of: %s", release.TagName, checksumAsset, verifierNames(verifiers))
	}
	return nil, fmt.Errorf("could not verify %s: %w", checksumAsset, errors.Join(errs...))
}

// parseVersion returns the release version recorded in the checksum file.
func parseVersion(r io.Reader) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if v, ok := strings.CutPrefix(scanner.Text(), versionPrefix); ok {
			return strings.TrimSpace(v), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no version found in " + checksumAsset)
}

// parseChecksum finds the checksum for name in sha256sum output. The file
// names in there may carry a directory prefix, e.g. "builds/sc-linux-amd64".
func parseChecksum(r io.Reader, name string) (string, error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) != 2 {
			continue
		}
		if path.Base(strings.TrimPrefix(fields[1], "*")) == name {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("no checksum found for " + name)
}
