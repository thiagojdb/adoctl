package devops

import (
	"testing"

	"adoctl/pkg/models"
)

func TestPullRequestWebURLUsesBrowserLinkAndEscapesFallback(t *testing.T) {
	pr := models.PullRequest{
		ID:  42,
		URL: "https://dev.azure.com/acme/project/_apis/git/repositories/repo/pullRequests/42",
		Repository: models.Repository{
			Name:    "repo name",
			Project: models.Project{Name: "project name"},
		},
	}

	want := "https://dev.azure.com/acme/project%20name/_git/repo%20name/pullrequest/42"
	if got := pullRequestWebURL(pr, "acme"); got != want {
		t.Fatalf("pullRequestWebURL() = %q, want %q", got, want)
	}

	pr.WebURL = "https://dev.azure.com/acme/project/_git/repo/pullrequest/42"
	if got := pullRequestWebURL(pr, "ignored"); got != pr.WebURL {
		t.Fatalf("pullRequestWebURL() = %q, want supplied browser URL %q", got, pr.WebURL)
	}
}
