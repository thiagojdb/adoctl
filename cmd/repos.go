package cmd

import (
	"fmt"
	"regexp"
	"strings"

	"adoctl/pkg/devops"
	"adoctl/pkg/filter"
	"adoctl/pkg/logger"
	"adoctl/pkg/models"

	"github.com/spf13/cobra"
)

var (
	reposFilterRegex string
	reposFilterFuzzy string
)

// RepositoryOutput represents a repository for structured output
type RepositoryOutput struct {
	ID      string `json:"id" yaml:"id"`
	Name    string `json:"name" yaml:"name"`
	URL     string `json:"url" yaml:"url"`
	Project string `json:"project" yaml:"project"`
}

var reposCmd = &cobra.Command{
	Use:     "repos",
	Aliases: []string{"repo"},
	Short:   "List repositories",
	Long:    `List all repositories in the Azure DevOps project.`,
	Example: `  # List all repositories
  adoctl repos

  # Filter repositories by regex pattern
  adoctl repos --filter-regex "frontend.*|backend.*"

  # Filter repositories by fuzzy match
  adoctl repos --filter-fuzzy "api"

  # Output as JSON
  adoctl repos --format json

  # Output as YAML
  adoctl repos --format yaml`,
	RunE: func(cmd *cobra.Command, args []string) error {
		svc, err := devops.NewServiceFromEnv()
		if err != nil {
			return fmt.Errorf("error creating service: %w", err)
		}
		defer svc.Close()

		repos, err := svc.ListRepositories()
		if err != nil {
			return fmt.Errorf("error listing repositories: %w", err)
		}

		filteredRepos := filterRepositories(repos)

		logger.Info().Int("count", len(filteredRepos)).Msg("Found repositories")

		// Check if structured output is requested
		format := cmd.Flag("format").Value.String()
		output := NewOutputWriter(format)

		if output.IsStructured() {
			// Convert to structured output
			repoOutputs := make([]RepositoryOutput, 0, len(filteredRepos))
			for _, repo := range filteredRepos {
				repoOutputs = append(repoOutputs, mapToRepositoryOutput(repo))
			}
			return output.Write(repoOutputs)
		}

		// Default table output
		var plainBuilder strings.Builder
		var markdownBuilder strings.Builder

		markdownBuilder.WriteString("**Repositories**\n\n")

		fmt.Println()
		for _, repo := range filteredRepos {
			name := repo.Name
			id := repo.ID
			webURL := repo.URL
			if webURL == "" {
				webURL = "N/A"
			}

			// Plain text output
			fmt.Fprintf(&plainBuilder, "Name: %s\n", name)
			fmt.Fprintf(&plainBuilder, "  ID: %s\n", id)
			fmt.Fprintf(&plainBuilder, "  URL: %s\n", webURL)
			plainBuilder.WriteString("\n")

			// Markdown output with clickable links for Teams
			if webURL != "N/A" {
				fmt.Fprintf(&markdownBuilder, "- [%s](%s) - ID: %s\n", name, webURL, id)
			} else {
				fmt.Fprintf(&markdownBuilder, "- %s - ID: %s\n", name, id)
			}

			// Print to terminal
			fmt.Printf("Name: %s\n", name)
			fmt.Printf("  ID: %s\n", id)
			fmt.Printf("  URL: %s\n", webURL)
			fmt.Println()
		}

		// Copy to clipboard if requested (using markdown format for Teams)
		if ShouldCopyOutput(cmd) {
			if err := CopyToClipboard(markdownBuilder.String()); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}
			fmt.Println("✓ Copied to clipboard!")
		}

		return nil
	},
}

func mapToRepositoryOutput(repo models.Repository) RepositoryOutput {
	return RepositoryOutput{
		ID:      repo.ID,
		Name:    repo.Name,
		URL:     repo.URL,
		Project: repo.Project.Name,
	}
}

func filterRepositories(repos []models.Repository) []models.Repository {
	filtered := []models.Repository{}
	for _, repo := range repos {
		name := repo.Name

		include := true

		if reposFilterRegex != "" {
			re, err := regexp.Compile(reposFilterRegex)
			if err != nil {
				continue
			}
			if !re.MatchString(name) {
				include = false
			}
		}

		if include && reposFilterFuzzy != "" {
			if !filter.FuzzyMatch(reposFilterFuzzy, name) {
				include = false
			}
		}

		if include {
			filtered = append(filtered, repo)
		}
	}
	return filtered
}

func init() {
	reposCmd.Flags().StringVar(&reposFilterRegex, "filter-regex", "", "Filter repository name by regex pattern")
	reposCmd.Flags().StringVar(&reposFilterFuzzy, "filter-fuzzy", "", "Filter repository name by fuzzy match")
}
