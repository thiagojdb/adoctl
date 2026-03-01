package devops

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"adoctl/pkg/config"
	"adoctl/pkg/models"
)

type prEntry struct {
	id       int
	title    string
	repoName string
	webURL   string
	warnings []string
}

type prGroupData struct {
	target string
	source string
	prs    []prEntry
}

func (s *DevOpsService) buildPRGroups(
	ctx context.Context,
	prs []models.PullRequest,
	showWarnings bool,
	workItemFilters []string,
) []prGroupData {
	checker := NewPRRequirementsChecker(s.client)
	grouped := map[string][]prEntry{}

	filteredPRs := []models.PullRequest{}
	for _, pr := range prs {
		repoID := pr.Repository.ID
		prID := pr.ID

		if len(workItemFilters) > 0 {
			if !checker.HasWorkItemMatch(ctx, repoID, prID, workItemFilters) {
				continue
			}
		}

		filteredPRs = append(filteredPRs, pr)
	}

	workItemsCounts := map[string]int{}
	if showWarnings {
		items := []struct {
			RepoID string
			PRID   int
		}{}
		for _, pr := range filteredPRs {
			repoID := pr.Repository.ID
			prID := pr.ID
			items = append(items, struct {
				RepoID string
				PRID   int
			}{repoID, prID})
		}
		workItemsCounts = checker.GetWorkItemsCountsBatch(ctx, items, config.DefaultParallelProcesses)
	}

	for _, pr := range filteredPRs {
		repoID := pr.Repository.ID
		prID := pr.ID
		repoName := pr.Repository.Name

		targetRefName := strings.Replace(pr.TargetBranch, "refs/heads/", "", 1)
		sourceRefName := strings.Replace(pr.SourceBranch, "refs/heads/", "", 1)
		targetRef := strings.ToLower(targetRefName)
		sourceRef := strings.ToLower(sourceRefName)
		groupKey := fmt.Sprintf("%s:%s", targetRef, sourceRef)

		project := pr.Repository.Project.Name
		if project == "" {
			project = "unknown"
		}

		webURL := fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s/pullrequest/%d", s.client.GetOrganization(), project, repoName, prID)

		warnings := []string{}
		if showWarnings {
			cacheKey := fmt.Sprintf("%s:%d", repoID, prID)
			workItemsCount := workItemsCounts[cacheKey]
			mergeWarnings := checker.CheckPRRequirements(pr)

			if workItemsCount == 0 {
				warnings = append(warnings, "No work items linked")
			}
			warnings = append(warnings, mergeWarnings...)
		}

		entry := prEntry{
			id:       prID,
			title:    pr.Title,
			repoName: repoName,
			webURL:   webURL,
			warnings: warnings,
		}

		grouped[groupKey] = append(grouped[groupKey], entry)
	}

	keys := []string{}
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := make([]prGroupData, 0, len(keys))
	for _, k := range keys {
		parts := strings.Split(k, ":")
		target, source := parts[0], parts[1]

		entries := grouped[k]
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].id < entries[j].id
		})

		result = append(result, prGroupData{
			target: target,
			source: source,
			prs:    entries,
		})
	}

	return result
}

// GenerateMessageReport generates a Markdown report (for terminal output).
func (s *DevOpsService) GenerateMessageReport(ctx context.Context, prs []models.PullRequest, showWarnings bool, workItemFilters []string) string {
	groups := s.buildPRGroups(ctx, prs, showWarnings, workItemFilters)

	lines := []string{}
	for _, g := range groups {
		lines = append(lines, fmt.Sprintf("PR's para %s (from %s):", g.target, g.source))
		lines = append(lines, "")

		for _, pr := range g.prs {
			warningStr := ""
			if len(pr.warnings) > 0 {
				warningStr = "\n    " + strings.Join(pr.warnings, " ")
			}
			prLine := fmt.Sprintf("%s: %s, [PR #%d](%s)%s", pr.repoName, pr.title, pr.id, pr.webURL, warningStr)
			lines = append(lines, prLine)
		}

		lines = append(lines, "")
	}

	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

// GeneratePlainTextReport generates a plain-text report (no URLs, no Markdown).
func (s *DevOpsService) GeneratePlainTextReport(ctx context.Context, prs []models.PullRequest, showWarnings bool, workItemFilters []string) string {
	groups := s.buildPRGroups(ctx, prs, showWarnings, workItemFilters)

	lines := []string{}
	for _, g := range groups {
		lines = append(lines, fmt.Sprintf("PR's para %s (from %s):", g.target, g.source))
		lines = append(lines, "")

		for _, pr := range g.prs {
			warningStr := ""
			if len(pr.warnings) > 0 {
				warningStr = "\n    " + strings.Join(pr.warnings, " ")
			}
			prLine := fmt.Sprintf("%s: %s, PR #%d%s", pr.repoName, pr.title, pr.id, warningStr)
			lines = append(lines, prLine)
		}

		lines = append(lines, "")
	}

	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

// GenerateHTMLMessageReport generates an HTML fragment report (for clipboard rich text).
func (s *DevOpsService) GenerateHTMLMessageReport(ctx context.Context, prs []models.PullRequest, showWarnings bool, workItemFilters []string) string {
	groups := s.buildPRGroups(ctx, prs, showWarnings, workItemFilters)

	lines := []string{}
	for _, g := range groups {
		lines = append(lines, fmt.Sprintf("PR's para %s (from %s):", g.target, g.source))
		lines = append(lines, "")

		for _, pr := range g.prs {
			warningStr := ""
			if len(pr.warnings) > 0 {
				warningStr = "<br>&nbsp;&nbsp;&nbsp;&nbsp;" + strings.Join(pr.warnings, " ")
			}
			prLine := fmt.Sprintf(`%s: %s, <a href="%s">PR #%d</a>%s`, pr.repoName, pr.title, pr.webURL, pr.id, warningStr)
			lines = append(lines, prLine)
		}

		lines = append(lines, "")
	}

	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "<br>")
}

// WrapHTMLForClipboard wraps an HTML fragment for the text/html clipboard MIME type.
func WrapHTMLForClipboard(fragment string) string {
	return "<html><body>" + fragment + "</body></html>"
}
