package cmd

import (
	"adoctl/pkg/cache"
	"adoctl/pkg/config"
	"adoctl/pkg/devops"
	"adoctl/pkg/logger"
	"bufio"
	"context"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
	"github.com/spf13/cobra"
)

var (
	pipelineStatusPRs        []int
	pipelineStatusFile       string
	pipelineStatusQuiet      bool
	pipelineStatusFormat     string
	pipelineStatusCachedOnly bool
	pipelineStatusWatch      bool
	pipelineStatusInterval   int
)

var pipelineStatusCmd = &cobra.Command{
	Use:   "pipeline-status",
	Short: "Get PR status and CI/CD pipeline status for PRs",
	Long:  `Get pull request status (including approvals), CI/CD pipeline status (builds and deployments) for one or more pull requests.`,
	Example: `  # Get status for a single PR (detailed format)
  adoctl pipeline-status --pr 123

  # Get status for multiple PRs
  adoctl pipeline-status --pr 123 --pr 456 --pr 789

  # Get status in modern table format
  adoctl pipeline-status --pr 123 --pr 456 --format modern

  # Watch PR status, refresh every 30 seconds
  adoctl pipeline-status --pr 123 --watch

  # Watch with custom refresh interval
  adoctl pipeline-status --pr 123 --watch --interval 60

  # Read PR numbers from file
  adoctl pipeline-status --file pr-list.txt

  # Use cached data only (no API sync)
  adoctl pipeline-status --pr 123 --cached-only`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx, cancel := GetContext()
		defer cancel()

		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("failed to create devops service: %w", err)
		}
		defer svc.Close()

		svc.SetSyncOptions(devops.SyncOptions{
			Quiet:    pipelineStatusQuiet,
			SkipSync: pipelineStatusCachedOnly,
		})

		prIDs, err := resolvePRIDs()
		if err != nil {
			return err
		}

		if len(prIDs) == 0 {
			return fmt.Errorf("no PR numbers specified. Use --pr or --file")
		}

		uniquePRs := deduplicatePRs(prIDs)

		shouldCopy := ShouldCopyOutput(cmd)

		if pipelineStatusFormat == "modern" {
			if pipelineStatusWatch {
				return watchModernFormat(svc, uniquePRs)
			}
			return displayModernFormatWithCopy(svc, uniquePRs, shouldCopy)
		}

		if pipelineStatusWatch {
			return watchDetailedFormat(ctx, svc, uniquePRs)
		}
		return displayDetailedFormatWithCopy(ctx, svc, uniquePRs, shouldCopy)
	},
}

func resolvePRIDs() ([]int, error) {
	allPRs := make([]int, 0, len(pipelineStatusPRs))
	allPRs = append(allPRs, pipelineStatusPRs...)

	if pipelineStatusFile != "" {
		filePRs, err := readPRsFromFile(pipelineStatusFile)
		if err != nil {
			return nil, fmt.Errorf("error reading PR file: %w", err)
		}
		allPRs = append(allPRs, filePRs...)
	}

	return allPRs, nil
}

func deduplicatePRs(prs []int) []int {
	seen := make(map[int]bool)
	unique := []int{}
	for _, pr := range prs {
		if !seen[pr] {
			seen[pr] = true
			unique = append(unique, pr)
		}
	}
	return unique
}

func displayDetailedFormat(ctx context.Context, svc *devops.DevOpsService, prIDs []int) error {
	return displayDetailedFormatWithCopy(ctx, svc, prIDs, false)
}

func displayDetailedFormatWithCopy(ctx context.Context, svc *devops.DevOpsService, prIDs []int, shouldCopy bool) error {
	var markdownBuilder strings.Builder

	if shouldCopy {
		markdownBuilder.WriteString("**Pipeline Status**\n\n")
	}

	for _, prID := range prIDs {
		pr, err := svc.GetPullRequest(ctx, prID)
		if err != nil {
			logger.Error().Err(err).Int("pr_id", prID).Msg("Failed to get PR")
			continue
		}

		printDetailedPRStatus(ctx, svc, pr)

		if shouldCopy {
			buildDetailedPRCopyOutput(&markdownBuilder, pr, svc, ctx)
		}
	}

	if shouldCopy {
		if err := CopyToClipboard(markdownBuilder.String()); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Println("\n✓ Copied to clipboard!")
	}

	return nil
}

func buildDetailedPRCopyOutput(markdownBuilder *strings.Builder, pr *git.GitPullRequest, svc *devops.DevOpsService, ctx context.Context) {
	title := ""
	if pr.Title != nil {
		title = *pr.Title
	}
	prID := 0
	if pr.PullRequestId != nil {
		prID = *pr.PullRequestId
	}

	status := ""
	if pr.Status != nil {
		status = string(*pr.Status)
	}

	// Build markdown with clickable links for Teams
	url := ""
	if pr.Url != nil {
		url = *pr.Url
	}
	if url != "" {
		fmt.Fprintf(markdownBuilder, "- **[PR #%d: %s](%s)**\n", prID, strings.TrimSpace(title), url)
	} else {
		fmt.Fprintf(markdownBuilder, "- **PR #%d: %s**\n", prID, strings.TrimSpace(title))
	}
	fmt.Fprintf(markdownBuilder, "  Status: %s\n", status)

	// Add approval status
	var reviewersList []git.IdentityRefWithVote
	if pr.Reviewers != nil {
		reviewersList = *pr.Reviewers
	}
	approvalResult := devops.GetApprovalStatus(&reviewersList)
	fmt.Fprintf(markdownBuilder, "  Approvals: %s (%d/%d approved", approvalResult.Status, approvalResult.Approved, approvalResult.Total)
	if approvalResult.Rejected > 0 {
		fmt.Fprintf(markdownBuilder, ", %d rejected", approvalResult.Rejected)
	}
	markdownBuilder.WriteString(")\n")

	// Add build/deployment info
	if pr.LastMergeCommit != nil && pr.LastMergeCommit.CommitId != nil {
		builds, err := svc.GetBuildsForCommit(*pr.LastMergeCommit.CommitId, "", "")
		if err == nil && len(builds) > 0 {
			markdownBuilder.WriteString("  Builds:\n")
			for _, build := range builds {
				fmt.Fprintf(markdownBuilder, "    - Build #%d: %s (%s)\n", build.BuildID, build.Status, build.Result)
			}
		}
	}

	markdownBuilder.WriteString("\n")
}

func printDetailedPRStatus(ctx context.Context, svc *devops.DevOpsService, pr *git.GitPullRequest) {
	fmt.Println(strings.Repeat("═", 68))
	title := ""
	if pr.Title != nil {
		title = *pr.Title
	}
	fmt.Printf("PR #%d: %s\n", *pr.PullRequestId, strings.TrimSpace(title))
	fmt.Println(strings.Repeat("═", 68))

	status := ""
	if pr.Status != nil {
		status = string(*pr.Status)
	}
	fmt.Printf("Status:    %s\n", status)

	commitId := ""
	if pr.LastMergeCommit != nil && *pr.LastMergeCommit.CommitId != "" && pr.LastMergeCommit.CommitId != nil {
		commitId = *pr.LastMergeCommit.CommitId
	}
	fmt.Printf("Commit:    %s\n", commitId)

	printApprovalStatus(pr.Reviewers)

	fmt.Println()

	printBuildsAndDeployments(ctx, svc, pr)
}

func printApprovalStatus(reviewers *[]git.IdentityRefWithVote) {
	var reviewersList []git.IdentityRefWithVote
	if reviewers != nil {
		reviewersList = *reviewers
	}
	approvalResult := devops.GetApprovalStatus(&reviewersList)
	fmt.Printf("Approvals: %s (%d/%d approved", approvalResult.Status, approvalResult.Approved, approvalResult.Total)
	if approvalResult.Rejected > 0 {
		fmt.Printf(", %d rejected", approvalResult.Rejected)
	}
	fmt.Printf(")\n")
}

func printBuildsAndDeployments(ctx context.Context, svc *devops.DevOpsService, pr *git.GitPullRequest) {
	if pr.LastMergeCommit == nil || pr.LastMergeCommit.CommitId == nil {
		return
	}

	builds, err := svc.GetBuildsForCommit(*pr.LastMergeCommit.CommitId, "", "")
	if err != nil {
		logger.Error().Err(err).Msg("Failed to get builds")
		return
	}

	if len(builds) == 0 {
		fmt.Println("No builds found for this PR")
		return
	}

	for i, build := range builds {
		if i > 0 {
			fmt.Println(strings.Repeat("─", 64))
		}
		fmt.Printf("Build #%d\n", build.BuildID)
		fmt.Printf("  Status:     %s\n", build.Status)
		fmt.Printf("  Result:     %s\n", build.Result)
		fmt.Printf("  Started:    %s\n", time.Time(build.StartTime).Format("2006-01-02 15:04:05"))

		if build.Status == "inProgress" {
			duration := FormatDuration(time.Time(build.StartTime))
			if duration != "" {
				fmt.Printf("  Duration:   %s\n", duration)
			}
		}

		deployments, err := svc.GetDeploymentsForBuild(build.BuildID)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to get deployments")
			continue
		}

		if len(deployments) == 0 {
			fmt.Printf("  Deployments: None\n\n")
			continue
		}

		fmt.Printf("  Deployments:\n")
		for _, deployment := range deployments {
			fmt.Printf("    • #%d: %s (%s)", deployment.DeploymentID, deployment.Status, deployment.OperationStatus)
			if deployment.Status == "inProgress" {
				duration := FormatDuration(deployment.StartTime)
				if duration != "" {
					fmt.Printf(" - %s", duration)
				}
			}
			fmt.Println()
		}
		fmt.Println()
	}
}

func watchDetailedFormat(ctx context.Context, svc *devops.DevOpsService, prIDs []int) error {
	interval := time.Duration(pipelineStatusInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		ClearScreen()
		fmt.Printf("🔄 Watching %d PR(s) - Refreshing every %ds (Ctrl+C to exit)\n", len(prIDs), pipelineStatusInterval)
		fmt.Println()

		for _, prID := range prIDs {
			pr, err := svc.GetPullRequest(ctx, prID)
			if err != nil {
				logger.Error().Err(err).Int("pr_id", prID).Msg("Failed to get PR")
				continue
			}

			printDetailedPRStatus(ctx, svc, pr)
		}

		fmt.Printf("\nLast update: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
		<-ticker.C
	}
}

func displayModernFormat(svc *devops.DevOpsService, prIDs []int) error {
	return displayModernFormatWithCopy(svc, prIDs, false)
}

func displayModernFormatWithCopy(svc *devops.DevOpsService, prIDs []int, shouldCopy bool) error {
	summaries := collectPRSummaries(svc, prIDs)
	printModernTable(summaries)

	if shouldCopy {
		var markdownBuilder strings.Builder

		markdownBuilder.WriteString("**Pipeline Status**\n\n")

		for _, s := range summaries {
			fmt.Fprintf(&markdownBuilder, "- **PR #%d**: %s\n", s.ID, s.Title)
			fmt.Fprintf(&markdownBuilder, "  Repository: %s | Status: %s | CI: %s %s\n",
				s.Repository, s.PRStatus, s.CIStatus, s.CIDescription)
			if s.CDStatus != "" {
				fmt.Fprintf(&markdownBuilder, "  CD: %s", s.CDStatus)
				if s.CDDescription != "" {
					fmt.Fprintf(&markdownBuilder, " %s", s.CDDescription)
				}
				if s.CDEnvironment != "" {
					fmt.Fprintf(&markdownBuilder, " (%s)", s.CDEnvironment)
				}
				markdownBuilder.WriteString("\n")
			}
			markdownBuilder.WriteString("\n")
		}

		if err := CopyToClipboard(markdownBuilder.String()); err != nil {
			return fmt.Errorf("failed to copy to clipboard: %w", err)
		}
		fmt.Println("\n✓ Copied to clipboard!")
	}

	return nil
}

func watchModernFormat(svc *devops.DevOpsService, prIDs []int) error {
	interval := time.Duration(pipelineStatusInterval) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		ClearScreen()
		fmt.Printf("🔄 Watching %d PR(s) - Refreshing every %ds (Ctrl+C to exit)\n", len(prIDs), pipelineStatusInterval)
		fmt.Println()

		summaries := collectPRSummaries(svc, prIDs)
		printModernTable(summaries)

		fmt.Printf("\nLast update: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
		<-ticker.C
	}
}

func collectPRSummaries(svc *devops.DevOpsService, prIDs []int) []*devops.PRSummary {
	summaries := make([]*devops.PRSummary, 0, len(prIDs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	type PRJob struct {
		prID        int
		pr          *git.GitPullRequest
		builds      []cache.Build
		deployments []devops.DeploymentStatusInfo
		err         error
	}

	jobs := make(chan int, len(prIDs))
	results := make(chan *PRJob, len(prIDs))

	for i := 0; i < config.DefaultParallelProcesses; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for prID := range jobs {
				result := &PRJob{prID: prID}

				pr, err := svc.GetPullRequest(context.Background(), prID)
				if err != nil {
					result.err = err
					results <- result
					continue
				}
				result.pr = pr

				if pr.LastMergeCommit != nil && pr.LastMergeCommit.CommitId != nil && *pr.LastMergeCommit.CommitId != "" {
					builds, err := svc.GetBuildsForCommit(*pr.LastMergeCommit.CommitId, "", "")
					if err == nil {
						result.builds = builds
					}

					if len(builds) > 0 {
						deployments, _ := svc.GetDeploymentsForBuild(builds[0].BuildID)
						result.deployments = deployments
					}
				}

				results <- result
			}
		}()
	}

	for _, prID := range prIDs {
		jobs <- prID
	}
	close(jobs)
	wg.Wait()
	close(results)

	for res := range results {
		if res.err != nil {
			logger.Error().Err(res.err).Int("pr_id", res.prID).Msg("Failed to get PR")
			continue
		}

		summary := devops.BuildPRSummary(res.pr, res.builds, convertDeployments(res.deployments))
		mu.Lock()
		summaries = append(summaries, summary)
		mu.Unlock()
	}

	return summaries
}

func convertDeployments(src []devops.DeploymentStatusInfo) []devops.DeploymentStatusInfo {
	return src
}

func printModernTable(summaries []*devops.PRSummary) {
	fmt.Printf("🔀 Pull Requests (%d found)\n", len(summaries))
	fmt.Println()

	if len(summaries) > 0 {
		const (
			colPR      = 7
			colTitle   = 32
			colRepo    = 30
			colStatus  = 14
			colCI      = 13
			colCD      = 16
			colUpdated = 13
		)

		header := fmt.Sprintf("│ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │",
			colPR-2, "PR",
			colTitle-2, "Title",
			colRepo-2, "Repository",
			colStatus-2, "Status",
			colCI-2, "CI",
			colCD-2, "CD",
			colUpdated-2, "Updated")
		separator := fmt.Sprintf("┼%-*s─┼%-*s─┼%-*s─┼%-*s─┼%-*s─┼%-*s─┼%-*s─┤",
			colPR-1, strings.Repeat("─", colPR-1),
			colTitle-1, strings.Repeat("─", colTitle-1),
			colRepo-1, strings.Repeat("─", colRepo-1),
			colStatus-1, strings.Repeat("─", colStatus-1),
			colCI-1, strings.Repeat("─", colCI-1),
			colCD-1, strings.Repeat("─", colCD-1),
			colUpdated-1, strings.Repeat("─", colUpdated-1))

		fmt.Println(header)
		fmt.Println(separator)

		for _, s := range summaries {
			row := formatPRSummaryRow(s, colPR, colTitle, colRepo, colStatus, colCI, colCD, colUpdated)
			fmt.Println(row)
		}
	}
}

func formatPRSummaryRow(s *devops.PRSummary, colPR, colTitle, colRepo, colStatus, colCI, colCD, colUpdated int) string {
	title := s.Title
	if len(title) > colTitle-4 {
		title = title[:colTitle-7] + "..."
	}

	repo := s.Repository
	if len(repo) > colRepo-2 {
		repo = repo[:colRepo-5] + "..."
	}

	ciStr := fmt.Sprintf("%s %s", s.CIStatus, s.CIDescription)
	cdStr := s.CDStatus
	if s.CDDescription != "" {
		if s.CDEnvironment != "" {
			cdStr = fmt.Sprintf("%s %s (%s)", s.CDStatus, s.CDDescription, s.CDEnvironment)
		} else {
			cdStr = fmt.Sprintf("%s %s", s.CDStatus, s.CDDescription)
		}
	}
	if len(cdStr) > colCD-2 {
		cdStr = cdStr[:colCD-5] + "..."
	}

	statusStr := s.PRStatus

	lastUpdate := s.LastUpdateTime
	if lastUpdate == "" {
		lastUpdate = "-"
	}

	return fmt.Sprintf("│ %-*d │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │ %-*s │",
		colPR-2, s.ID,
		colTitle-2, title,
		colRepo-2, repo,
		colStatus-2, statusStr,
		colCI-2, ciStr,
		colCD-2, cdStr,
		colUpdated-2, lastUpdate)
}

func readPRsFromFile(filename string) ([]int, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var prs []int
	numberPattern := regexp.MustCompile(`^\s*(\d+)\s*$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		matches := numberPattern.FindStringSubmatch(line)
		if matches != nil {
			prNum, err := strconv.Atoi(matches[1])
			if err != nil {
				continue
			}
			prs = append(prs, prNum)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return prs, nil
}

func init() {
	pipelineStatusCmd.Flags().IntSliceVar(&pipelineStatusPRs, "pr", []int{}, "PR number(s) (can specify multiple)")
	pipelineStatusCmd.Flags().StringVar(&pipelineStatusFile, "file", "", "Read PR numbers from a file")
	pipelineStatusCmd.Flags().BoolVar(&pipelineStatusQuiet, "quiet", false, "Suppress sync messages")
	pipelineStatusCmd.Flags().StringVar(&pipelineStatusFormat, "format", "detailed", "Output format: 'detailed' or 'modern'")
	pipelineStatusCmd.Flags().BoolVar(&pipelineStatusCachedOnly, "cached-only", false, "Use cached data only, skip syncing")
	pipelineStatusCmd.Flags().BoolVar(&pipelineStatusWatch, "watch", false, "Watch mode: continuously refresh PR status")
	pipelineStatusCmd.Flags().IntVar(&pipelineStatusInterval, "interval", 30, "Refresh interval in seconds (default: 30)")
	pipelineStatusCmd.MarkFlagsOneRequired("pr", "file")
}
