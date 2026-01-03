package filter

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"adoctl/pkg/models"
)

type FilterMode int

const (
	FilterModeNone FilterMode = iota
	FilterModeExact
	FilterModeContains
	FilterModeRegex
	FilterModeFuzzy
)

type StringFilter struct {
	Pattern string
	Mode    FilterMode
	regex   *regexp.Regexp
}

func NewStringFilter(pattern string, mode FilterMode) (*StringFilter, error) {
	f := &StringFilter{
		Pattern: pattern,
		Mode:    mode,
	}

	if mode == FilterModeRegex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern '%s': %w", pattern, err)
		}
		f.regex = re
	}

	return f, nil
}

func (f *StringFilter) Match(s string) bool {
	if f.Mode == FilterModeNone {
		return true
	}

	switch f.Mode {
	case FilterModeExact:
		return strings.EqualFold(s, f.Pattern)
	case FilterModeContains:
		return strings.Contains(strings.ToLower(s), strings.ToLower(f.Pattern))
	case FilterModeRegex:
		return f.regex != nil && f.regex.MatchString(s)
	case FilterModeFuzzy:
		return FuzzyMatch(f.Pattern, s)
	default:
		return true
	}
}

func FuzzyMatch(pattern, text string) bool {
	if pattern == "" {
		return true
	}
	if text == "" {
		return false
	}

	pattern = strings.ToLower(pattern)
	text = strings.ToLower(text)

	return fuzzyMatchRecursive(pattern, text, 0, 0, 0)
}

func fuzzyMatchRecursive(pattern, text string, pIdx, tIdx, consecutiveMatches int) bool {
	if pIdx >= len(pattern) {
		return true
	}
	if tIdx >= len(text) {
		return false
	}

	pChar := rune(pattern[pIdx])
	tChar := rune(text[tIdx])

	if pChar == tChar {
		remainingChars := len(text) - tIdx - 1
		remainingPattern := len(pattern) - pIdx - 1

		if remainingPattern == 0 {
			return true
		}

		if remainingChars >= remainingPattern {
			return fuzzyMatchRecursive(pattern, text, pIdx+1, tIdx+1, consecutiveMatches+1)
		}
	}

	return fuzzyMatchRecursive(pattern, text, pIdx, tIdx+1, 0)
}

func FuzzyMatchRanked(pattern, text string, threshold float64) bool {
	if pattern == "" {
		return true
	}
	if text == "" {
		return false
	}

	distance := LevenshteinDistance(pattern, text)
	maxLen := max(len(pattern), len(text))

	if maxLen == 0 {
		return true
	}

	similarity := 1.0 - float64(distance)/float64(maxLen)
	return similarity >= threshold
}

func LevenshteinDistance(s1, s2 string) int {
	if len(s1) == 0 {
		return len(s2)
	}
	if len(s2) == 0 {
		return len(s1)
	}

	previousRow := make([]int, len(s2)+1)
	currentRow := make([]int, len(s2)+1)

	for i := 0; i <= len(s2); i++ {
		previousRow[i] = i
	}

	for i := 0; i < len(s1); i++ {
		currentRow[0] = i + 1

		for j := 0; j < len(s2); j++ {
			cost := 1
			if unicode.ToLower(rune(s1[i])) == unicode.ToLower(rune(s2[j])) {
				cost = 0
			}

			deletion := currentRow[j] + 1
			insertion := previousRow[j+1] + 1
			substitution := previousRow[j] + cost

			currentRow[j+1] = min(min(deletion, insertion), substitution)
		}

		previousRow, currentRow = currentRow, previousRow
	}

	return previousRow[len(s2)]
}

type PRFilter struct {
	TitleRegex   string
	TitleFuzzy   string
	RepoRegex    string
	RepoFuzzy    string
	SourceBranch string
	TargetBranch string
	Status       string
	CreatorID    string
}

func (f *PRFilter) MatchesPR(pr models.PullRequest) (bool, error) {
	if f.TitleRegex != "" {
		re, err := regexp.Compile(f.TitleRegex)
		if err != nil {
			return false, fmt.Errorf("invalid title regex: %w", err)
		}
		if !re.MatchString(pr.Title) {
			return false, nil
		}
	}

	if f.TitleFuzzy != "" {
		if !FuzzyMatch(f.TitleFuzzy, pr.Title) {
			return false, nil
		}
	}

	if f.RepoRegex != "" {
		re, err := regexp.Compile(f.RepoRegex)
		if err != nil {
			return false, fmt.Errorf("invalid repo regex: %w", err)
		}
		if !re.MatchString(pr.Repository.Name) {
			return false, nil
		}
	}

	if f.RepoFuzzy != "" {
		if !FuzzyMatch(f.RepoFuzzy, pr.Repository.Name) {
			return false, nil
		}
	}

	if f.SourceBranch != "" {
		expectedRef := fmt.Sprintf("refs/heads/%s", f.SourceBranch)
		if !strings.EqualFold(pr.SourceBranch, expectedRef) {
			return false, nil
		}
	}

	if f.TargetBranch != "" {
		expectedRef := fmt.Sprintf("refs/heads/%s", f.TargetBranch)
		if !strings.EqualFold(pr.TargetBranch, expectedRef) {
			return false, nil
		}
	}

	if f.Status != "" {
		if f.Status != "all" && !strings.EqualFold(string(pr.Status), f.Status) {
			return false, nil
		}
	}

	if f.CreatorID != "" {
		if pr.CreatedBy.ID != f.CreatorID {
			return false, nil
		}
	}

	return true, nil
}
