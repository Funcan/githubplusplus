package cmd

import (
	"testing"

	gogithub "github.com/google/go-github/v72/github"
)

// makeRepoWithFullName constructs a minimal Repository with FullName set.
// GetFullName() returns "" unless FullName is explicitly populated, so tests
// must set it directly rather than relying on owner+name derivation.
func makeRepoWithFullName(fullName string) *gogithub.Repository {
	return &gogithub.Repository{FullName: strPtr(fullName)}
}

func TestDeduplicateRepos(t *testing.T) {
	a := makeRepoWithFullName("alice/alpha")
	b := makeRepoWithFullName("alice/beta")
	c := makeRepoWithFullName("bob/gamma")

	tests := []struct {
		name  string
		input []*gogithub.Repository
		want  []string // expected full names in order
	}{
		{
			name:  "nil input",
			input: nil,
			want:  []string{},
		},
		{
			name:  "empty input",
			input: []*gogithub.Repository{},
			want:  []string{},
		},
		{
			name:  "no duplicates",
			input: []*gogithub.Repository{a, b, c},
			want:  []string{"alice/alpha", "alice/beta", "bob/gamma"},
		},
		{
			name:  "adjacent duplicates removed",
			input: []*gogithub.Repository{a, a, b},
			want:  []string{"alice/alpha", "alice/beta"},
		},
		{
			name:  "non-adjacent duplicates removed",
			input: []*gogithub.Repository{a, b, a, c, b},
			want:  []string{"alice/alpha", "alice/beta", "bob/gamma"},
		},
		{
			name:  "first occurrence is preserved",
			input: []*gogithub.Repository{a, b, a},
			want:  []string{"alice/alpha", "alice/beta"},
		},
		{
			name:  "all duplicates of single repo",
			input: []*gogithub.Repository{a, a, a},
			want:  []string{"alice/alpha"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deduplicateRepos(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d repos, want %d: %v", len(got), len(tt.want), fullNames(got))
			}
			for i, r := range got {
				if r.GetFullName() != tt.want[i] {
					t.Errorf("repo[%d] = %q, want %q", i, r.GetFullName(), tt.want[i])
				}
			}
		})
	}
}

func fullNames(repos []*gogithub.Repository) []string {
	names := make([]string, len(repos))
	for i, r := range repos {
		names[i] = r.GetFullName()
	}
	return names
}
