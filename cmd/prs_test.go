package cmd

import (
	"testing"

	gogithub "github.com/google/go-github/v72/github"
)

func makePR(mergeableState string, draft bool) *gogithub.PullRequest {
	return &gogithub.PullRequest{
		MergeableState: strPtr(mergeableState),
		Draft:          boolPtr(draft),
	}
}

func TestFilterReadyPRs(t *testing.T) {
	clean := makePR("clean", false)
	dirty := makePR("dirty", false)
	blocked := makePR("blocked", false)
	behind := makePR("behind", false)
	unstable := makePR("unstable", false)
	draft := makePR("draft", true)
	unknown := makePR("unknown", false)

	tests := []struct {
		name string
		in   []*gogithub.PullRequest
		want int
	}{
		{
			name: "only clean PRs are returned",
			in:   []*gogithub.PullRequest{clean, dirty, blocked, behind, unstable, draft, unknown},
			want: 1,
		},
		{
			name: "empty input",
			in:   nil,
			want: 0,
		},
		{
			name: "all clean",
			in:   []*gogithub.PullRequest{clean, makePR("clean", false)},
			want: 2,
		},
		{
			name: "no clean PRs",
			in:   []*gogithub.PullRequest{dirty, blocked, draft},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterReadyPRs(tt.in)
			if len(got) != tt.want {
				t.Errorf("filterReadyPRs returned %d PRs, want %d", len(got), tt.want)
			}
			for _, pr := range got {
				if pr.GetMergeableState() != "clean" {
					t.Errorf("returned PR with MergeableState %q, want \"clean\"", pr.GetMergeableState())
				}
			}
		})
	}
}
