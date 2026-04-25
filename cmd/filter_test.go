package cmd

import (
	"fmt"
	"net/http"
	"testing"

	gogithub "github.com/google/go-github/v72/github"
)

func boolPtr(b bool) *bool    { return &b }
func strPtr(s string) *string { return &s }

func makeRepo(name string, fork, archived bool, language string) *gogithub.Repository {
	return &gogithub.Repository{
		Name:     strPtr(name),
		Fork:     boolPtr(fork),
		Archived: boolPtr(archived),
		Language: strPtr(language),
	}
}

func TestApplyFilters(t *testing.T) {
	goFork := makeRepo("go-fork", true, false, "Go")
	goNonFork := makeRepo("go-nonfork", false, false, "Go")
	rustFork := makeRepo("rust-fork", true, false, "Rust")
	archivedGo := makeRepo("archived-go", false, true, "Go")
	all := []*gogithub.Repository{goFork, goNonFork, rustFork, archivedGo}

	tests := []struct {
		name  string
		flags filterFlags
		want  []string
	}{
		{
			name:  "no filters returns all",
			flags: filterFlags{},
			want:  []string{"go-fork", "go-nonfork", "rust-fork", "archived-go"},
		},
		{
			name:  "forks only",
			flags: filterFlags{forks: true},
			want:  []string{"go-fork", "rust-fork"},
		},
		{
			name:  "no forks",
			flags: filterFlags{noForks: true},
			want:  []string{"go-nonfork", "archived-go"},
		},
		{
			name:  "language Go",
			flags: filterFlags{language: "Go"},
			want:  []string{"go-fork", "go-nonfork", "archived-go"},
		},
		{
			name:  "language Go case-insensitive",
			flags: filterFlags{language: "go"},
			want:  []string{"go-fork", "go-nonfork", "archived-go"},
		},
		{
			name:  "archived only",
			flags: filterFlags{archived: true},
			want:  []string{"archived-go"},
		},
		{
			name:  "no archived",
			flags: filterFlags{noArchived: true},
			want:  []string{"go-fork", "go-nonfork", "rust-fork"},
		},
		{
			name:  "forks and language Go",
			flags: filterFlags{forks: true, language: "Go"},
			want:  []string{"go-fork"},
		},
		{
			name:  "no forks and no archived",
			flags: filterFlags{noForks: true, noArchived: true},
			want:  []string{"go-nonfork"},
		},
		{
			name:  "language with no matches",
			flags: filterFlags{language: "Python"},
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyFilters(all, &tt.flags)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d repos, want %d: %v", len(got), len(tt.want), repoNames(got))
			}
			for i, r := range got {
				if r.GetName() != tt.want[i] {
					t.Errorf("repo[%d] = %q, want %q", i, r.GetName(), tt.want[i])
				}
			}
		})
	}
}

func repoNames(repos []*gogithub.Repository) []string {
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.GetName()
	}
	return names
}

func TestIsNotFound(t *testing.T) {
	notFoundResp := &http.Response{StatusCode: http.StatusNotFound}
	forbiddenResp := &http.Response{StatusCode: http.StatusForbidden}

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "direct 404 error",
			err:  &gogithub.ErrorResponse{Response: notFoundResp},
			want: true,
		},
		{
			name: "wrapped 404 error",
			err:  fmt.Errorf("listing repos for org %q: %w", "acme", &gogithub.ErrorResponse{Response: notFoundResp}),
			want: true,
		},
		{
			name: "403 forbidden is not a 404",
			err:  &gogithub.ErrorResponse{Response: forbiddenResp},
			want: false,
		},
		{
			name: "plain error is not a 404",
			err:  fmt.Errorf("some other error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFound(tt.err); got != tt.want {
				t.Errorf("isNotFound(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
