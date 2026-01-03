package git

import (
	"testing"
)

func TestExtractWorkItemID(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		want       string
	}{
		{
			name:       "PBI prefix with hyphen",
			branchName: "feature/PBI-12345",
			want:       "12345",
		},
		{
			name:       "PBI prefix with underscore",
			branchName: "feature/PBI_12345",
			want:       "12345",
		},
		{
			name:       "WI prefix",
			branchName: "bugfix/WI-9876",
			want:       "9876",
		},
		{
			name:       "BUG prefix",
			branchName: "BUG-123-fix-issue",
			want:       "123",
		},
		{
			name:       "TASK prefix",
			branchName: "TASK-456-add-feature",
			want:       "456",
		},
		{
			name:       "FEATURE prefix",
			branchName: "FEATURE-789-new-ui",
			want:       "789",
		},
		{
			name:       "Hash prefix",
			branchName: "fix-bug-#1234",
			want:       "1234",
		},
		{
			name:       "Feature branch with number",
			branchName: "feature/12345-description",
			want:       "12345",
		},
		{
			name:       "Bugfix branch with number",
			branchName: "bugfix/12345-description",
			want:       "12345",
		},
		{
			name:       "Hotfix branch with number",
			branchName: "hotfix/12345-description",
			want:       "12345",
		},
		{
			name:       "Release branch with number",
			branchName: "release/12345-description",
			want:       "12345",
		},
		{
			name:       "No work item ID",
			branchName: "feature/my-awesome-feature",
			want:       "",
		},
		{
			name:       "Just number in branch",
			branchName: "fix-something-123",
			want:       "",
		},
		{
			name:       "Empty branch name",
			branchName: "",
			want:       "",
		},
		{
			name:       "Complex branch name",
			branchName: "feature/PBI-12345-add-login-page",
			want:       "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractWorkItemID(tt.branchName)
			if got != tt.want {
				t.Errorf("ExtractWorkItemID(%q) = %q, want %q", tt.branchName, got, tt.want)
			}
		})
	}
}

func TestParseAzureDevOpsURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		want    *RemoteInfo
		wantErr bool
	}{
		{
			name: "HTTPS format - dev.azure.com",
			url:  "https://dev.azure.com/myorg/myproject/_git/myrepo",
			want: &RemoteInfo{
				Organization: "myorg",
				Project:      "myproject",
				Repository:   "myrepo",
			},
			wantErr: false,
		},
		{
			name: "HTTPS format - dev.azure.com with .git suffix",
			url:  "https://dev.azure.com/myorg/myproject/_git/myrepo.git",
			want: &RemoteInfo{
				Organization: "myorg",
				Project:      "myproject",
				Repository:   "myrepo",
			},
			wantErr: false,
		},
		{
			name: "Old Visual Studio format",
			url:  "https://myorg.visualstudio.com/myproject/_git/myrepo",
			want: &RemoteInfo{
				Organization: "myorg",
				Project:      "myproject",
				Repository:   "myrepo",
			},
			wantErr: false,
		},
		{
			name: "SSH format",
			url:  "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo",
			want: &RemoteInfo{
				Organization: "myorg",
				Project:      "myproject",
				Repository:   "myrepo",
			},
			wantErr: false,
		},
		{
			name: "SSH format with .git suffix",
			url:  "git@ssh.dev.azure.com:v3/myorg/myproject/myrepo.git",
			want: &RemoteInfo{
				Organization: "myorg",
				Project:      "myproject",
				Repository:   "myrepo",
			},
			wantErr: false,
		},
		{
			name:    "Empty URL",
			url:     "",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid URL",
			url:     "https://github.com/user/repo",
			want:    nil,
			wantErr: true,
		},
		{
			name: "HTTPS format with hyphenated names",
			url:  "https://dev.azure.com/my-org/my-project/_git/my-repo",
			want: &RemoteInfo{
				Organization: "my-org",
				Project:      "my-project",
				Repository:   "my-repo",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseAzureDevOpsURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseAzureDevOpsURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != nil {
				if got.Organization != tt.want.Organization {
					t.Errorf("Organization = %q, want %q", got.Organization, tt.want.Organization)
				}
				if got.Project != tt.want.Project {
					t.Errorf("Project = %q, want %q", got.Project, tt.want.Project)
				}
				if got.Repository != tt.want.Repository {
					t.Errorf("Repository = %q, want %q", got.Repository, tt.want.Repository)
				}
			}
		})
	}
}

func TestParseCommits(t *testing.T) {
	input := `abc123` + "\x00" + `First commit` + "\x00" + `This is the body` + "\x00" + "\x01" +
		`def456` + "\x00" + `Second commit` + "\x00" + "" + "\x00" + "\x01"

	commits := parseCommits(input)

	if len(commits) != 2 {
		t.Errorf("Expected 2 commits, got %d", len(commits))
	}

	if commits[0].Hash != "abc123" {
		t.Errorf("Expected first commit hash to be 'abc123', got %q", commits[0].Hash)
	}

	if commits[0].Subject != "First commit" {
		t.Errorf("Expected first commit subject to be 'First commit', got %q", commits[0].Subject)
	}

	if commits[0].Body != "This is the body" {
		t.Errorf("Expected first commit body to be 'This is the body', got %q", commits[0].Body)
	}

	if commits[1].Hash != "def456" {
		t.Errorf("Expected second commit hash to be 'def456', got %q", commits[1].Hash)
	}

	if commits[1].Subject != "Second commit" {
		t.Errorf("Expected second commit subject to be 'Second commit', got %q", commits[1].Subject)
	}
}

func TestParseCommitsEmpty(t *testing.T) {
	commits := parseCommits("")
	if len(commits) != 0 {
		t.Errorf("Expected 0 commits for empty input, got %d", len(commits))
	}
}
