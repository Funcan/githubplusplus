package git

import (
	"fmt"
	"os/exec"
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
