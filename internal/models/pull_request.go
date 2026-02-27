package models

import (
	"fmt"
	"time"
)

type CiStatus int

const (
	CiStatusPending CiStatus = iota
	CiStatusSuccess
	CiStatusFailure
)

type PullRequest struct {
	Number     int
	Title      string
	Repository string
	Author     string
	Draft      bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
	CiStatus   CiStatus

	LastCommentAt        time.Time
	LastCommitAt         time.Time
	LastCiStatusUpdateAt time.Time

	LastAcknowledgedAt *time.Time

	RequestedReviewers []string
}

func (pr PullRequest) DisplayString() string {
	return fmt.Sprintf("%s %s : %s/%d", pr.Author, pr.Title, pr.Repository, pr.Number)
}

func (pr PullRequest) UpdatesSinceLastAck() string {
	if pr.LastAcknowledgedAt != nil {
		updates := "  "
		if pr.LastCommentAt.After(*pr.LastAcknowledgedAt) {
			updates += "New Comment | "
		}

		if pr.LastCommitAt.After(*pr.LastAcknowledgedAt) {
			updates += "New Commits | "
		}

		if pr.LastCiStatusUpdateAt.After(*pr.LastAcknowledgedAt) {
			updates += "CI Status Changed | "
		}

		return updates 
	}

	return "  New PR"
}

type User struct {
	AccessToken string
	Username string
}
