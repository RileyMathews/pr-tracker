package repository

import (
	"context"
	"database/sql"
	"time"

	"git.rileymathews.com/riley/pr-tracker/internal/db/gen"
	"git.rileymathews.com/riley/pr-tracker/internal/models"
)

type DatabaseRepository struct {
	queries *gen.Queries
	ctx context.Context
}

func New(queries *gen.Queries, context context.Context) *DatabaseRepository {
	return &DatabaseRepository {
		queries: queries,
		ctx: context,
	}
}

func (repository *DatabaseRepository) SavePr(internalPR *models.PullRequest) error {
	return repository.queries.UpsertPullRequest(repository.ctx, gen.UpsertPullRequestParams{
		Number:               int64(internalPR.Number),
		Title:                internalPR.Title,
		Repository:           internalPR.Repository,
		Author:               internalPR.Author,
		Draft:                internalPR.Draft,
		CreatedAtUnix:        internalPR.CreatedAt.Unix(),
		UpdatedAtUnix:        internalPR.UpdatedAt.Unix(),
		CiStatus:             int64(internalPR.CiStatus),
		LastCommentUnix:      internalPR.LastCommentAt.Unix(),
		LastCommitUnix:       internalPR.LastCommitAt.Unix(),
		LastAcknowledgedUnix: timeToNullInt64(internalPR.LastAcknowledgedAt),
	})
}

func (repository *DatabaseRepository) GetPr(repoName string, prNumber int) (*models.PullRequest, error) {
	row, err := repository.queries.GetPullRequestByRepoAndNumber(repository.ctx, gen.GetPullRequestByRepoAndNumberParams{
		Repository: repoName,
		Number:     int64(prNumber),
	})
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	var lastAcknowledgedAt *time.Time
	if row.LastAcknowledgedUnix.Valid {
		t := time.Unix(row.LastAcknowledgedUnix.Int64, 0).UTC()
		lastAcknowledgedAt = &t
	}

	return &models.PullRequest{
		Number:               int(row.Number),
		Title:                row.Title,
		Repository:           row.Repository,
		Author:               row.Author,
		Draft:                row.Draft,
		CreatedAt:            time.Unix(row.CreatedAtUnix, 0).UTC(),
		UpdatedAt:            time.Unix(row.UpdatedAtUnix, 0).UTC(),
		CiStatus:             models.CiStatus(row.CiStatus),
		LastCommentAt:        time.Unix(row.LastCommentUnix, 0).UTC(),
		LastCommitAt:         time.Unix(row.LastCommitUnix, 0).UTC(),
		LastCiStatusUpdateAt: time.Unix(row.LastCiStatusUpdateUnix, 0).UTC(),
		LastAcknowledgedAt:   lastAcknowledgedAt,
	}, nil
}


func timeToNullInt64(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: value.Unix(), Valid: true}
}

