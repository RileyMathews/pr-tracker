package service

import (
	"fmt"
	"time"

	gh "git.rileymathews.com/riley/pr-tracker/internal/github"
	"git.rileymathews.com/riley/pr-tracker/internal/models"
)

func FetchPullRequestDetails(repoName string, prID int, authToken string) (*models.PullRequest, error) {
	return fetchPullRequestDetails(repoName, prID, authToken)
}

func fetchPullRequestDetails(repoName string, prID int, authToken string) (*models.PullRequest, error) {
	prDetails, err := gh.FetchPullRequestDetails(repoName, prID, authToken)
	if err != nil {
		return nil, fmt.Errorf("fetch github pr details: %w", err)
	}

	ciStatuses, err := gh.FetchPullRequestCIStatuses(repoName, prID, authToken)
	if err != nil {
		return nil, fmt.Errorf("fetch github pr ci statuses: %w", err)
	}

	createdAt, err := parseGitHubTimestamp(prDetails.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse pr created_at: %w", err)
	}

	updatedAt, err := parseGitHubTimestamp(prDetails.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("parse pr updated_at: %w", err)
	}

	reviewerLogins := make([]string, 0, len(prDetails.RequestedReviewers))
	for _, r := range prDetails.RequestedReviewers {
		reviewerLogins = append(reviewerLogins, r.Login)
	}

	return &models.PullRequest{
		Number:             prDetails.Number,
		Title:              prDetails.Title,
		Repository:         repoName,
		Author:             prDetails.User.Login,
		Draft:              prDetails.Draft,
		CreatedAt:          createdAt,
		UpdatedAt:          updatedAt,
		CiStatus:           mapCIStatus(ciStatuses),
		LastCommentAt:      latestCommentTime(prDetails),
		LastCommitAt:       latestCommitActivityTime(ciStatuses),
		RequestedReviewers: reviewerLogins,
	}, nil
}

func parseGitHubTimestamp(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Time{}, err
	}

	return t, nil
}

func latestCommentTime(prDetails *gh.PullRequestDetails) time.Time {
	var latest time.Time

	for _, comment := range prDetails.IssueComments {
		if t, err := parseGitHubTimestamp(comment.UpdatedAt); err == nil && t.After(latest) {
			latest = t
		}
	}

	for _, comment := range prDetails.ReviewComments {
		if t, err := parseGitHubTimestamp(comment.UpdatedAt); err == nil && t.After(latest) {
			latest = t
		}
	}

	return latest
}

func latestCommitActivityTime(ciStatuses *gh.PullRequestCIStatuses) time.Time {
	var latest time.Time

	for _, checkRun := range ciStatuses.CheckRuns {
		if t, err := parseGitHubTimestamp(checkRun.CompletedAt); err == nil && t.After(latest) {
			latest = t
		}
		if t, err := parseGitHubTimestamp(checkRun.StartedAt); err == nil && t.After(latest) {
			latest = t
		}
	}

	for _, status := range ciStatuses.Statuses {
		if t, err := parseGitHubTimestamp(status.UpdatedAt); err == nil && t.After(latest) {
			latest = t
		}
		if t, err := parseGitHubTimestamp(status.CreatedAt); err == nil && t.After(latest) {
			latest = t
		}
	}

	return latest
}

func mapCIStatus(ciStatuses *gh.PullRequestCIStatuses) models.CiStatus {
	if hasFailingCheckRun(ciStatuses.CheckRuns) {
		return models.CiStatusFailure
	}

	if hasPendingCheckRun(ciStatuses.CheckRuns) {
		return models.CiStatusPending
	}

	switch ciStatuses.CombinedState {
	case "success":
		return models.CiStatusSuccess
	case "failure", "error":
		return models.CiStatusFailure
	default:
		return models.CiStatusPending
	}
}

func hasFailingCheckRun(checkRuns []gh.CheckRun) bool {
	for _, checkRun := range checkRuns {
		switch checkRun.Conclusion {
		case "failure", "timed_out", "cancelled", "startup_failure", "action_required", "stale":
			return true
		}
	}

	return false
}

func hasPendingCheckRun(checkRuns []gh.CheckRun) bool {
	for _, checkRun := range checkRuns {
		switch checkRun.Status {
		case "queued", "in_progress", "waiting", "requested", "pending":
			return true
		}
	}

	return false
}
