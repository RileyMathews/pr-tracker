package main

// This file is meant to be used as a local development entrypoint
// for testing out various bits of code while iterating that would otherwise be 'internal' code that doesn't have
// a real entrypoint yet.

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"time"

	"git.rileymathews.com/riley/pr-tracker/internal/db/gen"
	"git.rileymathews.com/riley/pr-tracker/internal/models"
	"git.rileymathews.com/riley/pr-tracker/internal/service"
	_ "modernc.org/sqlite"
)

func main() {
	token := os.Getenv("LOCAL_GH_TOKEN")
	if token == "" {
		log.Fatal("LOCAL_GH_TOKEN environment variable is required")
	}

	const repoName = "MercuryTechnologies/mercury-web-backend"
	const prID = 65326

	internalPR, err := service.FetchPullRequestDetails(repoName, prID, token)
	if err != nil {
		log.Fatalf("transform github PR to internal model failed: %v", err)
	}
	log.Printf("Internal PR response: %+v", internalPR)

	dbConn, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatalf("open sqlite db failed: %v", err)
	}
	defer func() {
		if closeErr := dbConn.Close(); closeErr != nil {
			log.Printf("close sqlite db failed: %v", closeErr)
		}
	}()

	ctx := context.Background()
	if err := applyMigrations(ctx, dbConn, "internal/db/migrations"); err != nil {
		log.Fatalf("apply sqlite migrations failed: %v", err)
	}

	queries := gen.New(dbConn)

	if err := queries.UpsertPullRequest(ctx, gen.UpsertPullRequestParams{
		Number:               int64(internalPR.Number),
		Title:                internalPR.Title,
		Repository:           internalPR.Repository,
		Author:               internalPR.Author,
		Draft:                boolToInt64(internalPR.Draft),
		CreatedAtUnix:        internalPR.CreatedAt.Unix(),
		UpdatedAtUnix:        internalPR.UpdatedAt.Unix(),
		CiStatus:             int64(internalPR.CiStatus),
		LastCommentUnix:      internalPR.LastCommentAt.Unix(),
		LastCommitUnix:       internalPR.LastCommitAt.Unix(),
		LastAcknowledgedUnix: timeToNullInt64(internalPR.LastAcknowledgedAt),
	}); err != nil {
		log.Fatalf("persist internal PR failed: %v", err)
	}

	row, err := queries.GetPullRequestByNumber(ctx, int64(internalPR.Number))
	if err != nil {
		log.Fatalf("read persisted internal PR failed: %v", err)
	}

	persistedPR := &models.PullRequest{
		Number:             int(row.Number),
		Title:              row.Title,
		Repository:         row.Repository,
		Author:             row.Author,
		Draft:              row.Draft == 1,
		CreatedAt:          time.Unix(row.CreatedAtUnix, 0).UTC(),
		UpdatedAt:          time.Unix(row.UpdatedAtUnix, 0).UTC(),
		CiStatus:           models.CiStatus(row.CiStatus),
		LastCommentAt:      time.Unix(row.LastCommentUnix, 0).UTC(),
		LastCommitAt:       time.Unix(row.LastCommitUnix, 0).UTC(),
		LastAcknowledgedAt: nullInt64ToTimePtr(row.LastAcknowledgedUnix),
	}

	log.Printf("Persisted internal PR response: %+v", persistedPR)
}

func applyMigrations(ctx context.Context, dbConn *sql.DB, migrationsDir string) error {
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

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}

	return 0
}

func timeToNullInt64(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}

	return sql.NullInt64{Int64: value.Unix(), Valid: true}
}

func nullInt64ToTimePtr(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}

	t := time.Unix(value.Int64, 0).UTC()
	return &t
}
