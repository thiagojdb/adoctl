package filter

import (
	"testing"

	"adoctl/pkg/models"
)

func TestNewStringFilter(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		mode      FilterMode
		wantErr   bool
		errString string
	}{
		{
			name:    "valid exact filter",
			pattern: "test",
			mode:    FilterModeExact,
			wantErr: false,
		},
		{
			name:    "valid contains filter",
			pattern: "test",
			mode:    FilterModeContains,
			wantErr: false,
		},
		{
			name:    "valid regex filter",
			pattern: "^test$",
			mode:    FilterModeRegex,
			wantErr: false,
		},
		{
			name:      "invalid regex filter",
			pattern:   "[invalid(",
			mode:      FilterModeRegex,
			wantErr:   true,
			errString: "invalid regex pattern",
		},
		{
			name:    "valid fuzzy filter",
			pattern: "tst",
			mode:    FilterModeFuzzy,
			wantErr: false,
		},
		{
			name:    "none mode",
			pattern: "",
			mode:    FilterModeNone,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewStringFilter(tt.pattern, tt.mode)
			if tt.wantErr {
				if err == nil {
					t.Errorf("NewStringFilter() expected error, got nil")
				}
				if tt.errString != "" && err != nil {
					if !contains(err.Error(), tt.errString) {
						t.Errorf("NewStringFilter() error = %v, want containing %v", err, tt.errString)
					}
				}
			} else {
				if err != nil {
					t.Errorf("NewStringFilter() unexpected error = %v", err)
				}
				if filter == nil {
					t.Error("NewStringFilter() returned nil filter")
				}
			}
		})
	}
}

func TestStringFilter_Match(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		mode    FilterMode
		input   string
		want    bool
	}{
		// Exact matches
		{
			name:    "exact match - case insensitive",
			pattern: "Test",
			mode:    FilterModeExact,
			input:   "test",
			want:    true,
		},
		{
			name:    "exact match - exact case",
			pattern: "Test",
			mode:    FilterModeExact,
			input:   "Test",
			want:    true,
		},
		{
			name:    "exact no match",
			pattern: "Test",
			mode:    FilterModeExact,
			input:   "Testing",
			want:    false,
		},
		// Contains matches
		{
			name:    "contains match",
			pattern: "test",
			mode:    FilterModeContains,
			input:   "this is a test string",
			want:    true,
		},
		{
			name:    "contains match - case insensitive",
			pattern: "TEST",
			mode:    FilterModeContains,
			input:   "this is a test string",
			want:    true,
		},
		{
			name:    "contains no match",
			pattern: "xyz",
			mode:    FilterModeContains,
			input:   "this is a test string",
			want:    false,
		},
		// Regex matches
		{
			name:    "regex match - simple",
			pattern: "^test$",
			mode:    FilterModeRegex,
			input:   "test",
			want:    true,
		},
		{
			name:    "regex match - pattern",
			pattern: "^test.*",
			mode:    FilterModeRegex,
			input:   "testing",
			want:    true,
		},
		{
			name:    "regex no match",
			pattern: "^test$",
			mode:    FilterModeRegex,
			input:   "testing",
			want:    false,
		},
		{
			name:    "regex match - alternation",
			pattern: "(foo|bar)",
			mode:    FilterModeRegex,
			input:   "foobar",
			want:    true,
		},
		// Fuzzy matches
		{
			name:    "fuzzy match - simple",
			pattern: "tst",
			mode:    FilterModeFuzzy,
			input:   "test",
			want:    true,
		},
		{
			name:    "fuzzy match - with gaps",
			pattern: "api",
			mode:    FilterModeFuzzy,
			input:   "application programming interface",
			want:    true,
		},
		{
			name:    "fuzzy no match",
			pattern: "xyz",
			mode:    FilterModeFuzzy,
			input:   "test",
			want:    false,
		},
		// None mode
		{
			name:    "none mode - always matches",
			pattern: "",
			mode:    FilterModeNone,
			input:   "anything",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter, err := NewStringFilter(tt.pattern, tt.mode)
			if err != nil {
				t.Fatalf("NewStringFilter() failed: %v", err)
			}

			got := filter.Match(tt.input)
			if got != tt.want {
				t.Errorf("Match(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		name    string
		pattern string
		text    string
		want    bool
	}{
		{
			name:    "empty pattern matches everything",
			pattern: "",
			text:    "anything",
			want:    true,
		},
		{
			name:    "empty text with non-empty pattern",
			pattern: "abc",
			text:    "",
			want:    false,
		},
		{
			name:    "exact match",
			pattern: "test",
			text:    "test",
			want:    true,
		},
		{
			name:    "case insensitive",
			pattern: "TEST",
			text:    "test",
			want:    true,
		},
		{
			name:    "subsequence match",
			pattern: "api",
			text:    "application programming interface",
			want:    true,
		},
		{
			name:    "common abbreviation",
			pattern: "fmt",
			text:    "format",
			want:    true,
		},
		{
			name:    "reforma tributaria",
			pattern: "rfrmt",
			text:    "reforma tributaria",
			want:    true,
		},
		{
			name:    "no match - missing characters",
			pattern: "xyz",
			text:    "test",
			want:    false,
		},
		{
			name:    "no match - wrong order",
			pattern: "cba",
			text:    "abc",
			want:    false,
		},
		{
			name:    "longer pattern than text",
			pattern: "abcdef",
			text:    "abc",
			want:    false,
		},
		{
			name:    "single character match",
			pattern: "a",
			text:    "application",
			want:    true,
		},
		{
			name:    "numbers in pattern",
			pattern: "123",
			text:    "test-123-abc",
			want:    true,
		},
		{
			name:    "special characters in text",
			pattern: "test",
			text:    "test_123-abc.xyz",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FuzzyMatch(tt.pattern, tt.text)
			if got != tt.want {
				t.Errorf("FuzzyMatch(%q, %q) = %v, want %v", tt.pattern, tt.text, got, tt.want)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want int
	}{
		{
			name: "both empty",
			s1:   "",
			s2:   "",
			want: 0,
		},
		{
			name: "first empty",
			s1:   "",
			s2:   "test",
			want: 4,
		},
		{
			name: "second empty",
			s1:   "test",
			s2:   "",
			want: 4,
		},
		{
			name: "identical strings",
			s1:   "test",
			s2:   "test",
			want: 0,
		},
		{
			name: "single substitution",
			s1:   "test",
			s2:   "tent",
			want: 1,
		},
		{
			name: "single insertion",
			s1:   "test",
			s2:   "tests",
			want: 1,
		},
		{
			name: "single deletion",
			s1:   "tests",
			s2:   "test",
			want: 1,
		},
		{
			name: "completely different",
			s1:   "abc",
			s2:   "xyz",
			want: 3,
		},
		{
			name: "case insensitive",
			s1:   "Test",
			s2:   "test",
			want: 0,
		},
		{
			name: "kitten to sitting",
			s1:   "kitten",
			s2:   "sitting",
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LevenshteinDistance(tt.s1, tt.s2)
			if got != tt.want {
				t.Errorf("LevenshteinDistance(%q, %q) = %d, want %d", tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

func TestFuzzyMatchRanked(t *testing.T) {
	tests := []struct {
		name      string
		pattern   string
		text      string
		threshold float64
		want      bool
	}{
		{
			name:      "exact match",
			pattern:   "test",
			text:      "test",
			threshold: 0.8,
			want:      true,
		},
		{
			name:      "similar strings",
			pattern:   "test",
			text:      "tested",
			threshold: 0.5,
			want:      true,
		},
		{
			name:      "different strings",
			pattern:   "test",
			text:      "completely different",
			threshold: 0.8,
			want:      false,
		},
		{
			name:      "empty pattern",
			pattern:   "",
			text:      "anything",
			threshold: 0.5,
			want:      true,
		},
		{
			name:      "empty text",
			pattern:   "test",
			text:      "",
			threshold: 0.5,
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FuzzyMatchRanked(tt.pattern, tt.text, tt.threshold)
			if got != tt.want {
				t.Errorf("FuzzyMatchRanked(%q, %q, %f) = %v, want %v", tt.pattern, tt.text, tt.threshold, got, tt.want)
			}
		})
	}
}

func TestPRFilter_MatchesPR(t *testing.T) {
	tests := []struct {
		name    string
		filter  PRFilter
		pr      models.PullRequest
		want    bool
		wantErr bool
	}{
		{
			name:   "empty filter matches everything",
			filter: PRFilter{},
			pr: models.PullRequest{
				Title: "test PR",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "title regex match",
			filter: PRFilter{
				TitleRegex: "^feat:",
			},
			pr: models.PullRequest{
				Title: "feat: new feature",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "title regex no match",
			filter: PRFilter{
				TitleRegex: "^feat:",
			},
			pr: models.PullRequest{
				Title: "fix: bug fix",
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "title fuzzy match",
			filter: PRFilter{
				TitleFuzzy: "nwftr",
			},
			pr: models.PullRequest{
				Title: "new feature added",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "invalid title regex",
			filter: PRFilter{
				TitleRegex: "[invalid(",
			},
			pr: models.PullRequest{
				Title: "test",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "status match",
			filter: PRFilter{
				Status: "active",
			},
			pr: models.PullRequest{
				Status: models.PRStatusActive,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "status no match",
			filter: PRFilter{
				Status: "completed",
			},
			pr: models.PullRequest{
				Status: models.PRStatusActive,
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "status all matches any",
			filter: PRFilter{
				Status: "all",
			},
			pr: models.PullRequest{
				Status: models.PRStatusActive,
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "target branch match",
			filter: PRFilter{
				TargetBranch: "main",
			},
			pr: models.PullRequest{
				TargetBranch: "refs/heads/main",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "target branch case insensitive",
			filter: PRFilter{
				TargetBranch: "MAIN",
			},
			pr: models.PullRequest{
				TargetBranch: "refs/heads/main",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "source branch match",
			filter: PRFilter{
				SourceBranch: "feature/test",
			},
			pr: models.PullRequest{
				SourceBranch: "refs/heads/feature/test",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "repo regex match",
			filter: PRFilter{
				RepoRegex: "^backend-",
			},
			pr: models.PullRequest{
				Repository: models.Repository{
					Name: "backend-api",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "repo regex no match",
			filter: PRFilter{
				RepoRegex: "^frontend-",
			},
			pr: models.PullRequest{
				Repository: models.Repository{
					Name: "backend-api",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "repo fuzzy match",
			filter: PRFilter{
				RepoFuzzy: "bcknd",
			},
			pr: models.PullRequest{
				Repository: models.Repository{
					Name: "backend-api",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "creator ID match",
			filter: PRFilter{
				CreatorID: "user123",
			},
			pr: models.PullRequest{
				CreatedBy: models.Identity{
					ID: "user123",
				},
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "creator ID no match",
			filter: PRFilter{
				CreatorID: "user123",
			},
			pr: models.PullRequest{
				CreatedBy: models.Identity{
					ID: "user456",
				},
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "missing required field",
			filter: PRFilter{
				TitleRegex: "^feat:",
			},
			pr: models.PullRequest{
				Title:  "",
				Status: models.PRStatusActive,
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.filter.MatchesPR(tt.pr)
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchesPR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("MatchesPR() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
