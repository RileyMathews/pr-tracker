package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"git.rileymathews.com/riley/pr-tracker/internal/db/gen"
	"git.rileymathews.com/riley/pr-tracker/internal/models"
)

type DatabaseRepository struct {
	queries *gen.Queries
	ctx     context.Context
}

func New(queries *gen.Queries, context context.Context) *DatabaseRepository {
	return &DatabaseRepository{
		queries: queries,
		ctx:     context,
	}
}

func (repository *DatabaseRepository) SavePr(internalPR *models.PullRequest) error {
	reviewersJSON, err := json.Marshal(internalPR.RequestedReviewers)
	if err != nil {
		return fmt.Errorf("marshal requested_reviewers: %w", err)
	}

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
		RequestedReviewers:   string(reviewersJSON),
	})
}

func (repository *DatabaseRepository) GetAllPrs() ([]*models.PullRequest, error) {
	rows, err := repository.queries.GetAllPullRequests(repository.ctx)
	if err != nil {
		return nil, err
	}

	prs := make([]*models.PullRequest, 0, len(rows))
	for _, row := range rows {
		var lastAcknowledgedAt *time.Time
		if row.LastAcknowledgedUnix.Valid {
			t := time.Unix(row.LastAcknowledgedUnix.Int64, 0).UTC()
			lastAcknowledgedAt = &t
		}

		var reviewerLogins []string
		if err := json.Unmarshal([]byte(row.RequestedReviewers), &reviewerLogins); err != nil {
			return nil, fmt.Errorf("unmarshal requested_reviewers for pr %d: %w", row.Number, err)
		}

		prs = append(prs, &models.PullRequest{
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
			RequestedReviewers:   reviewerLogins,
		})
	}

	return prs, nil
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

	var reviewerLogins []string
	if err := json.Unmarshal([]byte(row.RequestedReviewers), &reviewerLogins); err != nil {
		return nil, fmt.Errorf("unmarshal requested_reviewers for pr %d: %w", row.Number, err)
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
		RequestedReviewers:   reviewerLogins,
	}, nil
}

func (repository *DatabaseRepository) GetTrackedAuthors() ([]string, error) {
	return repository.queries.GetTrackedAuthors(repository.ctx)
}

func (repository *DatabaseRepository) SaveTrackedAuthor(author string) error {
	return repository.queries.SaveTrackedAuthor(repository.ctx, author)
}

func (repository *DatabaseRepository) GetTrackedRepositories() ([]string, error) {
	return repository.queries.GetTrackedRepositories(repository.ctx)
}

func (repository *DatabaseRepository) SaveTrackedRepository(repo string) error {
	return repository.queries.SaveTrackedRepository(repository.ctx, repo)
}

func (repository *DatabaseRepository) DeleteTrackedRepository(repo string) error {
	return repository.queries.DeleteTrackedRepository(repository.ctx, repo)
}

func timeToNullInt64(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: value.Unix(), Valid: true}
}

func ApplyMigrations(ctx context.Context, dbConn *sql.DB, migrationsDir string) error {
	pattern := filepath.Join(migrationsDir, "*.sql")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("glob migration files: %w", err)
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found in %s", migrationsDir)
	}

	sort.Strings(files)
	for _, file := range files {
		sqlBytes, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("read migration file %s: %w", file, err)
		}
		if _, err := dbConn.ExecContext(ctx, string(sqlBytes)); err != nil {
			return fmt.Errorf("execute migration file %s: %w", file, err)
		}
	}

	return nil
}
