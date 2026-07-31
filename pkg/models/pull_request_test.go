package models

import (
	"testing"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

func TestPullRequestFromAzurePrefersBrowserURL(t *testing.T) {
	prID := 42
	restURL := "https://dev.azure.com/acme/project/_apis/git/repositories/repo/pullRequests/42"
	webURL := "https://dev.azure.com/acme/project/_git/repo/pullrequest/42"

	pr := PullRequestFromAzure(&git.GitPullRequest{
		PullRequestId: &prID,
		Url:           &restURL,
		Links: map[string]interface{}{
			"web": map[string]interface{}{"href": webURL},
		},
	})

	if pr.URL != webURL {
		t.Fatalf("PullRequestFromAzure().URL = %q, want browser URL %q", pr.URL, webURL)
	}
	if pr.WebURL != webURL {
		t.Fatalf("PullRequestFromAzure().WebURL = %q, want browser URL %q", pr.WebURL, webURL)
	}
}
