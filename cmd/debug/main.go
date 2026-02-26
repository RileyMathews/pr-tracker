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

	"git.rileymathews.com/riley/pr-tracker/internal/db/gen"
	"git.rileymathews.com/riley/pr-tracker/internal/db/repository"
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

	repository := repository.New(queries, ctx)

	saveErr := repository.SavePr(internalPR)
	if saveErr != nil {
		log.Fatalf("saving PR failed: %v", saveErr)
	}

	persistedPR, fetchErr := repository.GetPr(internalPR.Repository, internalPR.Number)
	if fetchErr != nil {
		log.Fatalf("Fetching PR failed: %v", fetchErr)
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

