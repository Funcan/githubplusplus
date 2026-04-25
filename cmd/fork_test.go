package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/v72/github"
)

func makeParentRepo(owner, name string) *gogithub.Repository {
	return &gogithub.Repository{
		Name:  strPtr(name),
		Owner: &gogithub.User{Login: strPtr(owner)},
	}
}

func TestValidateFork(t *testing.T) {
	parent := makeParentRepo("upstream-owner", "upstream-repo")

	tests := []struct {
		name       string
		repo       *gogithub.Repository
		owner      string
		repoName   string
		wantErr    string
		wantParent bool
	}{
		{
			name:     "not a fork",
			repo:     &gogithub.Repository{Fork: boolPtr(false)},
			owner:    "alice",
			repoName: "myrepo",
			wantErr:  "alice/myrepo is not a fork",
		},
		{
			name:     "fork with nil parent",
			repo:     &gogithub.Repository{Fork: boolPtr(true)},
			owner:    "alice",
			repoName: "myrepo",
			wantErr:  "alice/myrepo: upstream parent metadata unavailable",
		},
		{
			name:       "fork with parent returns parent",
			repo:       &gogithub.Repository{Fork: boolPtr(true), Parent: parent},
			owner:      "alice",
			repoName:   "myrepo",
			wantParent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateFork(tt.repo, tt.owner, tt.repoName)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("validateFork() error = %v, want %q", err, tt.wantErr)
				}
				if got != nil {
					t.Errorf("validateFork() returned non-nil parent on error")
				}
				return
			}
			if err != nil {
				t.Fatalf("validateFork() unexpected error: %v", err)
			}
			if got != parent {
				t.Errorf("validateFork() returned wrong parent")
			}
		})
	}
}

func TestForEachRepoArg(t *testing.T) {
	t.Run("all succeed", func(t *testing.T) {
		var called []string
		err := forEachRepoArg([]string{"a", "b", "c"}, "something failed", func(arg string) error {
			called = append(called, arg)
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
		if strings.Join(called, ",") != "a,b,c" {
			t.Errorf("called = %v, want [a b c]", called)
		}
	})

	t.Run("empty args", func(t *testing.T) {
		err := forEachRepoArg(nil, "something failed", func(arg string) error {
			t.Fatal("fn must not be called with empty args")
			return nil
		})
		if err != nil {
			t.Fatalf("expected nil error, got %v", err)
		}
	})

	t.Run("one failure does not short-circuit remaining args", func(t *testing.T) {
		var called []string
		err := forEachRepoArg([]string{"a", "b", "c"}, "one or more failed", func(arg string) error {
			called = append(called, arg)
			if arg == "b" {
				return errors.New("boom")
			}
			return nil
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "one or more failed" {
			t.Errorf("error = %q, want %q", err.Error(), "one or more failed")
		}
		if strings.Join(called, ",") != "a,b,c" {
			t.Errorf("called = %v, want [a b c] — remaining args must still be processed", called)
		}
	})

	t.Run("all fail returns summary error", func(t *testing.T) {
		err := forEachRepoArg([]string{"a", "b"}, "all failed", func(arg string) error {
			return fmt.Errorf("error for %s", arg)
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "all failed" {
			t.Errorf("error = %q, want %q", err.Error(), "all failed")
		}
	})

	t.Run("errors printed to stderr", func(t *testing.T) {
		old := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		_ = forEachRepoArg([]string{"owner/repo"}, "failed", func(arg string) error {
			return errors.New("something went wrong")
		})

		w.Close()
		os.Stderr = old

		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])
		if !strings.Contains(output, "error: owner/repo: something went wrong") {
			t.Errorf("stderr = %q, want it to contain %q", output, "error: owner/repo: something went wrong")
		}
	})
}
