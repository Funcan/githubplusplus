package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Clone runs `git clone <url> <dest>`.
func Clone(url, dest string) error {
	cmd := exec.Command("git", "clone", url, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone %s: %w\n%s", url, err, out)
	}
	return nil
}

// GetRemoteURL returns the URL configured for the named remote in dir.
func GetRemoteURL(dir, remote string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", "-C", dir, "remote", "get-url", remote)
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git remote get-url %s: %w\n%s", remote, err, stderr.String())
	}
	return strings.TrimSpace(string(out)), nil
}

// GetCurrentBranch returns the name of the currently checked-out branch in dir.
func GetCurrentBranch(dir string) (string, error) {
	var stderr bytes.Buffer
	cmd := exec.Command("git", "-C", dir, "symbolic-ref", "--short", "HEAD")
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git symbolic-ref HEAD: %w\n%s", err, stderr.String())
	}
	return strings.TrimSpace(string(out)), nil
}

// Fetch runs `git fetch <remote>` in dir.
func Fetch(dir, remote string) error {
	cmd := exec.Command("git", "-C", dir, "fetch", remote)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git fetch %s: %w\n%s", remote, err, out)
	}
	return nil
}

// MergeFFOnly runs `git merge --ff-only <ref>` in dir.
func MergeFFOnly(dir, ref string) error {
	cmd := exec.Command("git", "-C", dir, "merge", "--ff-only", ref)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git merge --ff-only %s: %w\n%s", ref, err, out)
	}
	return nil
}
