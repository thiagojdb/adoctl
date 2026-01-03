package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"adoctl/pkg/azure/client"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/strikethrough"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"

	"github.com/spf13/cobra"
)

var (
	workItemOutputDir string
	workItemUpdate    bool
)

// workItemResult holds the formatted output for a processed work item
type workItemResult struct {
	id       int
	markdown string
}

var workItemCmd = &cobra.Command{
	Use:     "workitem <id> [id...]",
	Aliases: []string{"wi", "workitems"},
	Short:   "Get work item content and save to markdown",
	Long: `Get work item content including name, assigned user, description, and comments.
Saves to a markdown file with attachments in a separate directory.
Multiple work item IDs can be provided at once.`,
	Example: `  # Get a single work item
  adoctl workitem 123

  # Get multiple work items
  adoctl workitem 123 456 789

  # Get work item and save to specific directory
  adoctl workitem 123 -o ./work-items

  # Update existing work item data from server
  adoctl workitem 123 -u -o ./work-items`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client, err := client.NewClient()
		if err != nil {
			return fmt.Errorf("failed to create client: %w", err)
		}

		outputDir := workItemOutputDir
		if outputDir == "" {
			if len(args) > 1 {
				return fmt.Errorf("--output directory is required when processing multiple work items")
			}
			outputDir = "."
		}

		ids := parseWorkItemIDs(args)
		if len(ids) == 0 {
			return fmt.Errorf("invalid work item ID(s)")
		}

		if workItemUpdate {
			for _, id := range ids {
				if err := removeExistingWorkItem(outputDir, id); err != nil {
					return fmt.Errorf("failed to remove existing work item %d: %w", id, err)
				}
			}
		}

		fmt.Printf("Processing %d work item(s)...\n\n", len(ids))

		results, errors := processWorkItemsWithCopy(client, ids, outputDir)

		if len(errors) > 0 {
			printWorkItemErrors(ids, errors)
		}

		// Copy summary to clipboard if requested
		if ShouldCopyOutput(cmd) && len(results) > 0 {
			var markdownBuilder strings.Builder

			markdownBuilder.WriteString("**Work Items**\n\n")

			for _, result := range results {
				markdownBuilder.WriteString(result.markdown)
			}

			if err := CopyToClipboard(markdownBuilder.String()); err != nil {
				return fmt.Errorf("failed to copy to clipboard: %w", err)
			}
			fmt.Println("\n✓ Copied summary to clipboard!")
		}

		return nil
	},
}

func parseWorkItemIDs(args []string) []int {
	ids := make([]int, 0, len(args))
	for _, arg := range args {
		var id int
		_, err := fmt.Sscanf(arg, "%d", &id)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func processWorkItemsWithCopy(c *client.Client, ids []int, outputDir string) ([]workItemResult, map[int]error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make(map[int]error)
	results := make([]workItemResult, 0, len(ids))

	for _, id := range ids {
		wg.Add(1)
		go func(workItemID int) {
			defer wg.Done()
			result, err := processSingleWorkItemWithCopy(c, workItemID, outputDir)
			if err != nil {
				mu.Lock()
				errors[workItemID] = err
				mu.Unlock()
				return
			}
			mu.Lock()
			results = append(results, result)
			mu.Unlock()
		}(id)
	}

	wg.Wait()
	return results, errors
}

func processSingleWorkItemWithCopy(c *client.Client, id int, outputDir string) (workItemResult, error) {
	fmt.Printf("Fetching work item %d...\n", id)

	workItem, err := c.GetWorkItem(context.Background(), id)
	if err != nil {
		return workItemResult{}, fmt.Errorf("failed to get work item: %w", err)
	}

	relations, err := c.GetWorkItemRelations(context.Background(), id)
	if err != nil {
		fmt.Printf("Warning: failed to get relations: %v\n", err)
		relations = []map[string]any{}
	}

	comments, err := c.GetWorkItemComments(context.Background(), id)
	if err != nil {
		fmt.Printf("Warning: failed to get comments: %v\n", err)
		comments = []map[string]any{}
	}

	title := extractWorkItemTitle(workItem)
	if title == "" {
		title = fmt.Sprintf("workitem-%d", id)
	}

	workItemDir := filepath.Join(outputDir, title)
	attachmentsDir := filepath.Join(workItemDir, "attachments")
	markdownFile := filepath.Join(workItemDir, fmt.Sprintf("%s.md", title))

	if err := os.MkdirAll(attachmentsDir, 0755); err != nil {
		return workItemResult{}, fmt.Errorf("failed to create directories: %w", err)
	}

	markdownContent := generateMarkdown(workItem, comments, relations, workItemDir, c)

	if err := os.WriteFile(markdownFile, []byte(markdownContent), 0644); err != nil {
		return workItemResult{}, fmt.Errorf("failed to write markdown file: %w", err)
	}

	fmt.Printf("\nWork item saved to: %s\n", workItemDir)

	downloadAttachments(c, relations, attachmentsDir, id)

	// Build summary for clipboard
	fields, ok := workItem["fields"].(map[string]any)
	if !ok {
		return workItemResult{}, fmt.Errorf("work item fields have invalid format")
	}
	workItemType := getStringField(fields, "System.WorkItemType")
	state := getStringField(fields, "System.State")
	assignedTo := getDisplayNameField(fields, "System.AssignedTo")
	url := extractWorkItemURL(workItem)

	var markdownBuilder strings.Builder

	// Markdown format for Teams
	if url != "" {
		fmt.Fprintf(&markdownBuilder, "- **[Work Item #%d: %s](%s)**\n", id, title, url)
	} else {
		fmt.Fprintf(&markdownBuilder, "- **Work Item #%d: %s**\n", id, title)
	}
	fmt.Fprintf(&markdownBuilder, "  Type: %s | State: %s | Assigned To: %s\n\n", workItemType, state, assignedTo)

	return workItemResult{
		id:       id,
		markdown: markdownBuilder.String(),
	}, nil
}

func extractWorkItemTitle(workItem map[string]any) string {
	if fields, ok := workItem["fields"].(map[string]any); ok {
		if val, ok := fields["System.Title"].(string); ok {
			return sanitizeFilename(val)
		}
	}
	return ""
}

func downloadAttachments(c *client.Client, relations []map[string]any, attachmentsDir string, workItemID int) {
	attachmentCount := 0
	var wg sync.WaitGroup
	var mu sync.Mutex
	attachmentErrors := make(map[string]error)
	skippedCount := 0

	for _, rel := range relations {
		if relRel, ok := rel["rel"].(string); ok && relRel == "AttachedFile" {
			attachmentCount++
			url := getRelationURL(rel)
			filename := getRelationFilename(rel)
			sanitizedFilename := sanitizeFilename(filename)
			destPath := filepath.Join(attachmentsDir, sanitizedFilename)

			if _, err := os.Stat(destPath); err == nil {
				wg.Add(1)
				go func(fname string) {
					defer wg.Done()
					mu.Lock()
					skippedCount++
					mu.Unlock()
					fmt.Printf("Skipping existing attachment: %s\n", fname)
				}(sanitizedFilename)
				continue
			}

			wg.Add(1)
			go func(fname, fpath, furl string) {
				defer wg.Done()
				fmt.Printf("Downloading attachment: %s\n", fname)
				if err := c.DownloadAttachment(context.Background(), furl, fpath); err != nil {
					mu.Lock()
					attachmentErrors[fname] = err
					mu.Unlock()
					fmt.Printf("Warning: failed to download attachment %s: %v\n", fname, err)
				}
			}(sanitizedFilename, destPath, url)
		}
	}

	wg.Wait()

	successCount := attachmentCount - len(attachmentErrors)
	if skippedCount > 0 {
		fmt.Printf("Attachments: %d downloaded, %d skipped (already exists)\n", successCount, skippedCount)
	} else {
		fmt.Printf("Attachments downloaded: %d/%d\n", successCount, attachmentCount)
	}
}

func getRelationURL(rel map[string]any) string {
	if urlVal, ok := rel["url"].(string); ok {
		return urlVal
	}
	return ""
}

func getRelationFilename(rel map[string]any) string {
	if attrs, ok := rel["attributes"].(map[string]any); ok {
		if name, ok := attrs["name"].(string); ok {
			return name
		}
	}
	return "unknown"
}

func printWorkItemErrors(ids []int, errors map[int]error) {
	fmt.Println("\n" + strings.Repeat("═", 64))
	fmt.Println("Summary:")
	for _, id := range ids {
		if err, ok := errors[id]; ok {
			fmt.Printf("Error processing work item %d: %v\n", id, err)
		}
	}
}

func sanitizeFilename(name string) string {
	invalidChars := "<>:\"/|?*"
	for _, c := range invalidChars {
		name = strings.ReplaceAll(name, string(c), "_")
	}
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "\n", "_")
	name = strings.ReplaceAll(name, "\r", "_")
	name = strings.ReplaceAll(name, "\t", "_")
	name = strings.Join(strings.Fields(name), "_")
	if len(name) > 200 {
		name = name[:200]
	}
	return name
}

func generateMarkdown(workItem map[string]any, comments []map[string]any, relations []map[string]any, workItemDir string, c *client.Client) string {
	var sb strings.Builder

	fields, ok := workItem["fields"].(map[string]any)
	if !ok {
		return "# Error: Invalid work item format\n\nWork item fields have invalid format."
	}

	title := getStringField(fields, "System.Title")
	assignedTo := getDisplayNameField(fields, "System.AssignedTo")
	state := getStringField(fields, "System.State")
	workItemType := getStringField(fields, "System.WorkItemType")
	createdBy := getDisplayNameField(fields, "System.CreatedBy")
	createdDate := getStringField(fields, "System.CreatedDate")
	changedBy := getDisplayNameField(fields, "System.ChangedBy")
	changedDate := getStringField(fields, "System.ChangedDate")
	description := getStringField(fields, "System.Description")
	acceptanceCriteria := getField(fields, "Microsoft.VSTS.Common.AcceptanceCriteria", "System.AcceptanceCriteria")
	conclusion := getField(fields, "Microsoft.VSTS.Common.Conclusion", "System.Conclusion")
	url := extractWorkItemURL(workItem)

	wiId := 0
	if idVal, ok := workItem["id"].(float64); ok {
		wiId = int(idVal)
	}

	fmt.Fprintf(&sb, "# %s\n\n", title)
	fmt.Fprintf(&sb, "**Work Item ID::** [%d](%s)\n\n", wiId, url)
	fmt.Fprintf(&sb, "**Type:** %s\n\n", workItemType)
	fmt.Fprintf(&sb, "**State:** %s\n\n", state)
	fmt.Fprintf(&sb, "**Assigned To:** %s\n\n", assignedTo)
	fmt.Fprintf(&sb, "**Created By:** %s\n\n", createdBy)
	fmt.Fprintf(&sb, "**Created Date:** %s\n\n", createdDate)
	fmt.Fprintf(&sb, "**Changed By:** %s\n\n", changedBy)
	fmt.Fprintf(&sb, "**Changed Date:** %s\n\n", changedDate)

	attachmentsDir := filepath.Join(workItemDir, "attachments")

	sb.WriteString("---\n\n")
	sb.WriteString("## Description\n\n")
	sb.WriteString(formatDescription(fields, description, attachmentsDir, c))

	if acceptanceCriteria != "" {
		sb.WriteString("---\n\n")
		sb.WriteString("## Acceptance Criteria\n\n")
		sb.WriteString(formatField(acceptanceCriteria, attachmentsDir, c))
	}

	if conclusion != "" {
		sb.WriteString("---\n\n")
		sb.WriteString("## Conclusion\n\n")
		sb.WriteString(formatField(conclusion, attachmentsDir, c))
	}

	if len(comments) > 0 {
		sb.WriteString("---\n\n")
		fmt.Fprintf(&sb, "## Comments (%d)\n\n", len(comments))
		for _, comment := range comments {
			formatComment(&sb, comment, attachmentsDir, c)
		}
	}

	if len(relations) > 0 {
		sb.WriteString("---\n\n")
		sb.WriteString("## Attachments\n\n")
		for _, rel := range relations {
			if relRel, ok := rel["rel"].(string); ok && relRel == "AttachedFile" {
				filename := getRelationFilename(rel)
				sanitizedFilename := sanitizeFilename(filename)
				relativePath := filepath.Join("attachments", sanitizedFilename)
				sb.WriteString(fmt.Sprintf("- [%s](%s)\n", filename, relativePath))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func getStringField(fields map[string]any, key string) string {
	if val, ok := fields[key].(string); ok {
		return val
	}
	return ""
}

func getDisplayNameField(fields map[string]any, key string) string {
	if val, ok := fields[key].(map[string]any); ok {
		if displayName, ok := val["displayName"].(string); ok {
			return displayName
		}
	}
	return ""
}

func getField(fields map[string]any, keys ...string) string {
	for _, key := range keys {
		if val, ok := fields[key].(string); ok {
			return val
		}
	}
	return ""
}

func extractWorkItemURL(workItem map[string]any) string {
	if links, ok := workItem["_links"].(map[string]any); ok {
		if html, ok := links["html"].(map[string]any); ok {
			if href, ok := html["href"].(string); ok {
				return href
			}
		}
	}
	return ""
}

func formatDescription(fields map[string]any, description, attachmentsDir string, c *client.Client) string {
	if description == "" {
		return "*No description*\n\n"
	}
	processedDesc, err := replaceImageURLs(description, attachmentsDir, c)
	if err != nil {
		fmt.Printf("Warning: failed to process images in description: %v\n", err)
		processedDesc = description
	}
	return fmt.Sprintf("%s\n\n", htmlToMarkdown(processedDesc))
}

func formatField(field, attachmentsDir string, c *client.Client) string {
	processed, err := replaceImageURLs(field, attachmentsDir, c)
	if err != nil {
		fmt.Printf("Warning: failed to process images: %v\n", err)
		processed = field
	}
	return fmt.Sprintf("%s\n\n", htmlToMarkdown(processed))
}

func formatComment(sb *strings.Builder, comment map[string]any, attachmentsDir string, c *client.Client) {
	author := getCommentAuthor(comment)

	sb.WriteString(fmt.Sprintf("### %s\n\n", author))
	if createdDate, ok := comment["createdDate"].(string); ok {
		sb.WriteString(fmt.Sprintf("**Posted:** %s\n\n", createdDate))
	}
	if modifiedDate, ok := comment["modifiedDate"].(string); ok {
		if createdDate, ok := comment["createdDate"].(string); ok && modifiedDate != createdDate {
			sb.WriteString(fmt.Sprintf("**Modified:** %s\n\n", modifiedDate))
		}
	}
	if text, ok := comment["text"].(string); ok {
		processedComment, err := replaceImageURLs(text, attachmentsDir, c)
		if err != nil {
			fmt.Printf("Warning: failed to process images in comment: %v\n", err)
			processedComment = text
		}
		sb.WriteString(fmt.Sprintf("%s\n\n", htmlToMarkdown(processedComment)))
	}
}

func getCommentAuthor(comment map[string]any) string {
	if createdBy, ok := comment["createdBy"].(map[string]any); ok {
		if displayName, ok := createdBy["displayName"].(string); ok {
			return displayName
		} else if uniqueName, ok := createdBy["uniqueName"].(string); ok {
			return uniqueName
		}
	}
	return ""
}

func htmlToMarkdown(html string) string {
	if html == "" {
		return ""
	}

	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			strikethrough.NewStrikethroughPlugin(),
			table.NewTablePlugin(),
		),
	)

	markdown, err := conv.ConvertString(html)
	if err != nil {
		return html
	}

	markdown = strings.ReplaceAll(markdown, `\!\[`, `![`)
	markdown = strings.ReplaceAll(markdown, `\[`, `[`)
	markdown = strings.ReplaceAll(markdown, `\]`, `]`)
	markdown = strings.ReplaceAll(markdown, `\_`, `_`)

	return markdown
}

func removeExistingWorkItem(outputDir string, id int) error {
	workItem, err := client.NewClient()
	if err != nil {
		return err
	}

	wi, err := workItem.GetWorkItem(context.Background(), id)
	if err != nil {
		return fmt.Errorf("failed to get work item: %w", err)
	}

	title := extractWorkItemTitle(wi)
	if title == "" {
		title = fmt.Sprintf("workitem-%d", id)
	}

	workItemDir := filepath.Join(outputDir, title)

	if _, err := os.Stat(workItemDir); os.IsNotExist(err) {
		return nil
	}

	fmt.Printf("Removing existing directory: %s\n", workItemDir)
	return os.RemoveAll(workItemDir)
}

var azureDevOpsImageTag = regexp.MustCompile(`<img[^>]+src="(https://dev\.azure\.com/[^"]+/_apis/wit/attachments/[^"]+fileName=[^"]+)"[^>]*>`)
var azureDevOpsMarkdownImage = regexp.MustCompile(`!\[([^\]]*)\]\((https://dev\.azure\.com/[^)]+/_apis/wit/attachments/[^)]+)\)`)

func extractImageURLs(content string) []string {
	uniqueURLs := make(map[string]bool)

	matches := azureDevOpsImageTag.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			uniqueURLs[match[1]] = true
		}
	}

	matches = azureDevOpsMarkdownImage.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 2 {
			uniqueURLs[match[2]] = true
		}
	}

	urls := make([]string, 0, len(uniqueURLs))
	for url := range uniqueURLs {
		urls = append(urls, url)
	}
	return urls
}

func downloadImage(c *client.Client, imageURL, attachmentsDir string) (string, error) {
	filename := "image"

	uuid := ""
	uuidMatch := regexp.MustCompile(`/attachments/([a-f0-9-]{36})\?fileName=`).FindStringSubmatch(imageURL)
	if len(uuidMatch) > 1 {
		uuid = uuidMatch[1]
	}

	if idx := strings.LastIndex(imageURL, "="); idx > 0 && idx+1 < len(imageURL) {
		filePart := imageURL[idx+1:]
		if extIdx := strings.LastIndex(filePart, "."); extIdx > 0 {
			ext := filePart[extIdx:]
			if uuid != "" {
				filename = "img_" + uuid + ext
			} else {
				filename = filePart
			}
		} else if uuid != "" {
			filename = "img_" + uuid
		} else {
			filename = filePart
		}
	} else if uuid != "" {
		filename = "img_" + uuid
	}

	sanitizedFilename := sanitizeFilename(filename)
	destPath := filepath.Join(attachmentsDir, sanitizedFilename)

	if _, err := os.Stat(destPath); err == nil {
		return sanitizedFilename, nil
	}

	if err := c.DownloadAttachment(context.Background(), imageURL, destPath); err != nil {
		return "", err
	}

	return sanitizedFilename, nil
}

func replaceImageURLs(content, attachmentsDir string, c *client.Client) (string, error) {
	imageURLs := extractImageURLs(content)
	for _, imageURL := range imageURLs {
		localPath, err := downloadImage(c, imageURL, attachmentsDir)
		if err != nil {
			fmt.Printf("Warning: failed to download image: %v\n", err)
			continue
		}
		localRef := filepath.Join("attachments", localPath)
		markdownImg := fmt.Sprintf("![Image](%s)", localRef)

		imgTag := fmt.Sprintf(`<img src="%s">`, imageURL)
		content = strings.ReplaceAll(content, imgTag, markdownImg)

		imgTagWithAttr := fmt.Sprintf(`<img src="%s"`, imageURL)
		if idx := strings.Index(content, imgTagWithAttr); idx >= 0 {
			endIdx := idx + len(imgTagWithAttr)
			for endIdx < len(content) && content[endIdx] != '>' {
				endIdx++
			}
			if endIdx < len(content) && content[endIdx] == '>' {
				oldTag := content[idx : endIdx+1]
				content = strings.ReplaceAll(content, oldTag, markdownImg)
			}
		}

		content = strings.ReplaceAll(content, imageURL, localRef)

		markdownImgAlt := fmt.Sprintf(`![%s](%s)`, "Image", localRef)
		oldMarkdownImg := fmt.Sprintf(`![%s](%s)`, "Image", imageURL)
		content = strings.ReplaceAll(content, oldMarkdownImg, markdownImgAlt)

		altText := "image.png"
		oldMarkdownImgWithAlt := fmt.Sprintf(`![%s](%s)`, altText, imageURL)
		markdownImgWithAlt := fmt.Sprintf(`![%s](%s)`, altText, localRef)
		content = strings.ReplaceAll(content, oldMarkdownImgWithAlt, markdownImgWithAlt)
	}
	return content, nil
}

func init() {
	AddCommand(rootCmd, workItemCmd)
	workItemCmd.Flags().StringVarP(&workItemOutputDir, "output", "o", "", "Output directory for work item files")
	workItemCmd.Flags().BoolVarP(&workItemUpdate, "update", "u", false, "Update existing work item from server")
}
