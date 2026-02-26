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


func timeToNullInt64(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: value.Unix(), Valid: true}
}

