package devops

import (
	"fmt"
	"strings"
	"time"

	"adoctl/pkg/cache"

	"github.com/microsoft/azure-devops-go-api/azuredevops/v7/git"
)

type PRSummary struct {
	ID             int
	Title          string
	Repository     string
	Author         string
	CreationDate   string
	CIStatus       string
	CIDescription  string
	CDStatus       string
	CDDescription  string
	CDEnvironment  string
	LastUpdateTime string
	ApprovalStatus string
	PRStatus       string
	ReviewersCount int
	ApprovedCount  int
	RejectedCount  int
	PendingCount   int
}

type ApprovalResult struct {
	Status   string
	Total    int
	Approved int
	Rejected int
	Pending  int
}

func GetApprovalStatus(reviewers *[]git.IdentityRefWithVote) ApprovalResult {
	if reviewers == nil || len(*reviewers) == 0 {
		return ApprovalResult{Status: "No reviewers"}
	}

	total := len(*reviewers)
	approved := 0
	rejected := 0
	pending := 0

	for _, reviewer := range *reviewers {
		if reviewer.Vote == nil {
			pending++
			continue
		}
		vote := *reviewer.Vote
		if vote >= 10 {
			approved++
		} else if vote <= -5 {
			rejected++
		} else {
			pending++
		}
	}

	if rejected > 0 {
		return ApprovalResult{Status: "Rejected", Total: total, Approved: approved, Rejected: rejected}
	}
	if approved > 0 && pending == 0 {
		return ApprovalResult{Status: "Approved", Total: total, Approved: approved, Rejected: rejected}
	}
	if approved > 0 {
		return ApprovalResult{Status: "Partial", Total: total, Approved: approved, Rejected: rejected}
	}
	return ApprovalResult{Status: "Pending", Total: total, Approved: approved, Rejected: rejected, Pending: pending}
}

func getPRStatus(pr *git.GitPullRequest, builds []cache.Build, deployments []DeploymentStatusInfo) string {
	prStatus := ""
	if pr.Status != nil {
		prStatus = string(*pr.Status)
	}

	if prStatus == "abandoned" {
		return "Abandoned"
	}

	if prStatus == "completed" {
		return "Merged"
	}

	if pr.WorkItemRefs == nil || len(*pr.WorkItemRefs) == 0 {
		return "Needs work items"
	}

	if pr.MergeStatus != nil && string(*pr.MergeStatus) == "conflicts" {
		return "Needs conflict resolution"
	}

	approvalResult := GetApprovalStatus(pr.Reviewers)
	if approvalResult.Status == "Rejected" || approvalResult.Status == "Pending" || approvalResult.Status == "Partial" || approvalResult.Status == "No reviewers" {
		return "Needs approval"
	}

	if len(builds) > 0 {
		build := builds[0]
		if build.Status == "inProgress" {
			return "CI in progress"
		}
		if build.Status == "completed" && build.Result == "failed" {
			return "CI failed"
		}
		if build.Status == "notStarted" {
			return "CI pending"
		}
	}

	if len(deployments) > 0 {
		deployment := deployments[0]
		if deployment.Status == "inProgress" {
			return "CD in progress"
		}
		if deployment.Status == "failed" {
			return "CD failed"
		}
	}

	if len(builds) == 0 {
		return "CI pending"
	}

	return "Completed"
}

func BuildPRSummary(pr *git.GitPullRequest, builds []cache.Build, deployments []DeploymentStatusInfo) *PRSummary {
	author := ""
	if pr.CreatedBy != nil && pr.CreatedBy.DisplayName != nil {
		author = *pr.CreatedBy.DisplayName
	}

	repository := ""
	if pr.Repository != nil && pr.Repository.Name != nil {
		repository = *pr.Repository.Name
	}

	title := ""
	if pr.Title != nil {
		title = *pr.Title
	}

	creationDate := ""
	if pr.CreationDate != nil {
		creationDate = pr.CreationDate.String()
	}

	prStatus := getPRStatus(pr, builds, deployments)
	approvalResult := GetApprovalStatus(pr.Reviewers)

	ciStatus := "◌"
	ciDesc := "Pending"
	cdStatus := "◌"
	cdDesc := ""
	cdEnv := ""
	lastUpdate := ""

	if len(builds) > 0 {
		build := builds[0]
		if build.Status == "completed" && build.Result == "succeeded" {
			ciStatus = "✔"
			ciDesc = "Succeeded"
		} else if build.Status == "inProgress" {
			ciStatus = "⟳"
			duration := FormatDuration(time.Time(build.StartTime))
			if duration != "" {
				ciDesc = fmt.Sprintf("Running (%s)", duration)
			} else {
				ciDesc = "Running"
			}
		} else if build.Status == "completed" && build.Result == "failed" {
			ciStatus = "✖"
			ciDesc = "Failed"
		} else if build.Status == "notStarted" {
			ciStatus = "○"
			ciDesc = "Queued"
		}

		if build.EndTime.Valid {
			lastUpdate = build.EndTime.Time.Format("02/01 15:04")
		} else if !build.StartTime.IsZero() {
			lastUpdate = build.StartTime.Format("02/01 15:04")
		}

		if len(deployments) > 0 {
			deployment := deployments[0]
			if deployment.Status == "succeeded" {
				cdStatus = "✔"
				cdDesc = "Succeeded"
				cdEnv = deployment.Environment
			} else if deployment.Status == "inProgress" {
				cdStatus = "⟳"
				duration := FormatDuration(deployment.StartTime)
				if duration != "" {
					cdDesc = fmt.Sprintf("Running (%s)", duration)
				} else {
					cdDesc = "Running"
				}
				cdEnv = deployment.Environment
			} else if deployment.Status == "failed" {
				cdStatus = "✖"
				cdDesc = "Failed"
				cdEnv = deployment.Environment
			} else if deployment.Status == "notDeployed" {
				cdStatus = "○"
				cdDesc = "Not Deployed"
				cdEnv = deployment.Environment
			}
		}
	}

	return &PRSummary{
		ID:             *pr.PullRequestId,
		Title:          strings.TrimSpace(title),
		Repository:     repository,
		Author:         author,
		CreationDate:   creationDate,
		CIStatus:       ciStatus,
		CIDescription:  ciDesc,
		CDStatus:       cdStatus,
		CDDescription:  cdDesc,
		CDEnvironment:  cdEnv,
		LastUpdateTime: lastUpdate,
		ApprovalStatus: approvalResult.Status,
		PRStatus:       prStatus,
		ReviewersCount: approvalResult.Total,
		ApprovedCount:  approvalResult.Approved,
		RejectedCount:  approvalResult.Rejected,
		PendingCount:   approvalResult.Pending,
	}
}

func FormatDuration(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}

	duration := time.Since(ts)

	if duration.Seconds() < 60 {
		seconds := int(duration.Seconds())
		if seconds == 1 {
			return "1s"
		}
		return fmt.Sprintf("%ds", seconds)
	}

	minutes := int(duration.Minutes())
	if minutes < 60 {
		if minutes == 1 {
			return "1m"
		}
		return fmt.Sprintf("%dm", minutes)
	}

	hours := int(duration.Hours())
	if hours == 1 {
		return "1h"
	}
	return fmt.Sprintf("%dh", hours)
}
