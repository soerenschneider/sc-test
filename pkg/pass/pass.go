package pass

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

func GetPassEntries(otp bool, otpPrefix string, passwordDir string) ([]string, error) {
	// Get the password store directory
	storeDir := getPasswordStoreDir()

	// Resolve symlinks to get the actual directory path
	resolvedStoreDir, err := filepath.EvalSymlinks(storeDir)
	if err != nil {
		// If symlink resolution fails, try with the original path
		resolvedStoreDir = storeDir
	}

	// Check if the directory exists
	if _, err := os.Stat(resolvedStoreDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("password store directory does not exist: %s", resolvedStoreDir)
	}

	var entries []string

	// Walk through all files in the password store directory
	err = filepath.Walk(resolvedStoreDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-.gpg files
		if info.IsDir() || !strings.HasSuffix(path, ".gpg") {
			return nil
		}

		// Calculate relative path from store directory
		relPath, err := filepath.Rel(resolvedStoreDir, path)
		if err != nil {
			return err
		}

		// Remove .gpg extension
		entryPath := strings.TrimSuffix(relPath, ".gpg")
		if !otp || strings.HasPrefix(entryPath, otpPrefix) {
			entries = append(entries, entryPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan password store directory: %w", err)
	}

	slices.Sort(entries)

	return entries, nil
}

// getPasswordStoreDir returns the password store directory path
func getPasswordStoreDir() string {
	// Check for PASSWORD_STORE_DIR environment variable
	if storeDir := os.Getenv("PASSWORD_STORE_DIR"); storeDir != "" {
		return storeDir
	}

	// Use default directory: ~/.password-store
	homeDir, err := os.UserHomeDir()
	if err != nil {
		// Fallback to $HOME if os.UserHomeDir() fails
		homeDir = os.Getenv("HOME")
	}

	return filepath.Join(homeDir, ".password-store")
}

// CheckPassInstalled verifies that pass is installed and accessible
func CheckPassInstalled() error {
	cmd := exec.Command("pass", "version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("pass command not found or not accessible: %w", err)
	}
	return nil
}

func GetPassEntry(name string, reveal bool, otpPrefix string) (string, error) {
	var args []string

	if strings.HasPrefix(name, otpPrefix) {
		args = append(args, "otp")
	}
	if reveal {
		args = append(args, "show")
	} else {
		args = append(args, "-c")
	}

	args = append(args, name)

	cmd := exec.Command("pass", args...)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to retrieve password for %s: %w", name, err)
	}

	return strings.TrimSpace(string(output)), nil
}
