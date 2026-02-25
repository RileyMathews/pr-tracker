package models

import "time"

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

	LastCommentAt time.Time
	LastCommitAt  time.Time

	LastAcknowledgedAt *time.Time
}
