package completions

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"adoctl/pkg/devops"

	"github.com/spf13/cobra"
)

type CacheData struct {
	Repositories []string
	Users        []string
}

type Completer struct {
	cachePath string
	cache     *CacheData
	mu        sync.RWMutex
}

func NewCompleter() *Completer {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = "/tmp"
	}
	cachePath := filepath.Join(cacheDir, "adoctl", "completion_cache.json")

	return &Completer{
		cachePath: cachePath,
		cache:     &CacheData{},
	}
}

func (c *Completer) CompleteRepositoryNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	svc, err := devops.NewServiceFromEnv()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	defer svc.Close()

	repos, err := svc.ListRepositories()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	repoNames := make([]string, 0, len(repos))
	for _, repo := range repos {
		repoNames = append(repoNames, repo.Name)
	}

	c.mu.Lock()
	c.cache.Repositories = repoNames
	c.mu.Unlock()

	return c.filterPrefix(repoNames, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteCachedRepositoryNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.cache.Repositories) == 0 {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	return c.filterPrefix(c.cache.Repositories, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteStatus(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	statuses := []string{"active", "completed", "abandoned", "all"}
	results := c.filterPrefix(statuses, toComplete)

	for i, status := range results {
		results[i] = fmt.Sprintf("%s\t%s", status, getStatusDescription(status))
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteFormat(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	formats := []string{"detailed", "modern"}
	results := c.filterPrefix(formats, toComplete)

	for i, format := range results {
		results[i] = fmt.Sprintf("%s\t%s", format, getFormatDescription(format))
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteBranchNames(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	commonBranches := []string{
		"main\tMain production branch",
		"master\tLegacy production branch",
		"develop\tDevelopment branch",
		"dev\tDevelopment branch",
		"staging\tStaging/pre-production",
		"preprod\tPre-production environment",
		"production\tProduction environment",
	}

	results := []string{}
	for _, branch := range commonBranches {
		parts := strings.Split(branch, "\t")
		if strings.HasPrefix(parts[0], toComplete) {
			results = append(results, branch)
		}
	}

	return results, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}

func (c *Completer) CompleteCreator(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	creators := []string{
		"self\tUse authenticated user",
	}

	svc, err := devops.NewServiceFromEnv()
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	defer svc.Close()

	allCreators := svc.GetAllCreators()
	for _, creator := range allCreators {
		name := creator["name"].(string)
		count := creator["count"].(int)
		creators = append(creators, fmt.Sprintf("%s\t%s (%d PRs)", name, name, count))
	}

	c.mu.Lock()
	c.cache.Users = creators
	c.mu.Unlock()

	return c.filterPrefix(creators, toComplete), cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) CompleteWorkItemType(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	types := []string{
		"PBI\tProduct Backlog Item",
		"Bug\tBug/Defect",
		"Task\tTask item",
		"Feature\tFeature request",
		"Epic\tEpic/Large feature",
		"Issue\tGeneral issue",
		"TestCase\tTest case",
		"User Story\tUser story",
	}

	results := []string{}
	for _, wt := range types {
		parts := strings.Split(wt, "\t")
		if strings.HasPrefix(parts[0], toComplete) {
			results = append(results, wt)
		}
	}

	return results, cobra.ShellCompDirectiveNoFileComp
}

func (c *Completer) filterPrefix(items []string, prefix string) []string {
	var result []string
	for _, item := range items {
		itemName := strings.Split(item, "\t")[0]
		if strings.HasPrefix(strings.ToLower(itemName), strings.ToLower(prefix)) {
			result = append(result, item)
		}
	}
	return result
}

func getStatusDescription(status string) string {
	switch status {
	case "active":
		return "Open pull requests"
	case "completed":
		return "Merged/closed PRs"
	case "abandoned":
		return "Abandoned PRs"
	case "all":
		return "All PRs regardless of status"
	default:
		return ""
	}
}

func getFormatDescription(format string) string {
	switch format {
	case "detailed":
		return "Full detailed output with all information"
	case "modern":
		return "Compact table format with emojis"
	default:
		return ""
	}
}

func RegisterCompletions(rootCmd *cobra.Command) {
	completer := NewCompleter()

	prListCmd, _, _ := rootCmd.Find([]string{"pr", "list"})
	if prListCmd != nil {
		prListCmd.RegisterFlagCompletionFunc("repository-name", completer.CompleteCachedRepositoryNames)
		prListCmd.RegisterFlagCompletionFunc("repo-id", completer.CompleteRepositoryNames)
		prListCmd.RegisterFlagCompletionFunc("status", completer.CompleteStatus)
		prListCmd.RegisterFlagCompletionFunc("target-branch", completer.CompleteBranchNames)
		prListCmd.RegisterFlagCompletionFunc("source-branch", completer.CompleteBranchNames)
		prListCmd.RegisterFlagCompletionFunc("creator", completer.CompleteCreator)
	}

	prReportCmd, _, _ := rootCmd.Find([]string{"pr", "report"})
	if prReportCmd != nil {
		prReportCmd.RegisterFlagCompletionFunc("repository-name", completer.CompleteCachedRepositoryNames)
		prReportCmd.RegisterFlagCompletionFunc("repo-id", completer.CompleteRepositoryNames)
		prReportCmd.RegisterFlagCompletionFunc("status", completer.CompleteStatus)
		prReportCmd.RegisterFlagCompletionFunc("target-branch", completer.CompleteBranchNames)
		prReportCmd.RegisterFlagCompletionFunc("source-branch", completer.CompleteBranchNames)
		prReportCmd.RegisterFlagCompletionFunc("creator", completer.CompleteCreator)
	}

	prPipelineStatusCmd, _, _ := rootCmd.Find([]string{"pr", "pipeline-status"})
	if prPipelineStatusCmd != nil {
		prPipelineStatusCmd.RegisterFlagCompletionFunc("format", completer.CompleteFormat)
	}

	prCreateCmd, _, _ := rootCmd.Find([]string{"pr", "create"})
	if prCreateCmd != nil {
		prCreateCmd.RegisterFlagCompletionFunc("repository-name", completer.CompleteCachedRepositoryNames)
		prCreateCmd.RegisterFlagCompletionFunc("source-branch", completer.CompleteBranchNames)
		prCreateCmd.RegisterFlagCompletionFunc("target-branch", completer.CompleteBranchNames)
	}

	prBulkCreateCmd, _, _ := rootCmd.Find([]string{"pr", "bulk-create"})
	if prBulkCreateCmd != nil {
		prBulkCreateCmd.RegisterFlagCompletionFunc("source-branch", completer.CompleteBranchNames)
		prBulkCreateCmd.RegisterFlagCompletionFunc("target-branch", completer.CompleteBranchNames)
	}

	linkWorkItemsCmd, _, _ := rootCmd.Find([]string{"pr", "link-workitems"})
	if linkWorkItemsCmd != nil {
		linkWorkItemsCmd.RegisterFlagCompletionFunc("work-item-id", completer.CompleteWorkItemType)
	}
}
