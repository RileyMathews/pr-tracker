package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"git.rileymathews.com/riley/pr-tracker/internal/core"
	"git.rileymathews.com/riley/pr-tracker/internal/db/gen"
	"git.rileymathews.com/riley/pr-tracker/internal/db/repository"
	"git.rileymathews.com/riley/pr-tracker/internal/github"
	"git.rileymathews.com/riley/pr-tracker/internal/models"
	"git.rileymathews.com/riley/pr-tracker/internal/service"
	_ "modernc.org/sqlite"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	dbConn, err := sql.Open("sqlite", "./db.sqlite3")
	if err != nil {
		log.Fatalf("open sqlite db failed: %v", err)
	}
	defer func() {
		if closeErr := dbConn.Close(); closeErr != nil {
			log.Printf("close sqlite db failed: %v", closeErr)
		}
	}()

	ctx := context.Background()
	if err := repository.ApplyMigrations(ctx, dbConn, "internal/db/migrations"); err != nil {
		log.Fatalf("apply sqlite migrations failed: %v", err)
	}

	queries := gen.New(dbConn)
	repo := repository.New(queries, ctx)

	if os.Args[1] == "auth" {
		log.Println("Authenticating user...")
		dispatchAuthCommand(repo, os.Args[2:])
		log.Println("User authenticated successfully")
		os.Exit(0)
	}

	user, err := repo.GetUser()
	if err != nil {
		log.Fatalf("fetch user failed: %v", err)
	}
	if user == nil {
		log.Fatal("no authenticated user found, please run 'cli auth <token>' to authenticate")
	}

	switch os.Args[1] {
	case "authors":
		dispatchAuthorsCommand(repo, os.Args[2:])

	case "repositories":
		dispatchRepositoriesCommand(repo, os.Args[2:])

	case "sync":
		dispatchSyncCommand(repo, user.AccessToken)

	case "prs":
		dispatchPrsCommand(repo)
	default:
		fmt.Printf("Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func dispatchAuthCommand(repo *repository.DatabaseRepository, args []string) {
	// ensure we don't already have a user configured
	maybeUser, err := repo.GetUser()
	if err != nil {
		log.Fatalf("fetch user failed: %v", err)
	}
	if maybeUser != nil {
		log.Fatalf("a user is already authenticated as '%s', please remove the existing user before authenticating a new one", maybeUser.Username)
	}

	log.Println("Fetching authenticated user...")
	if len(args) < 1 {
		log.Println("Authentication token is required")
		printUsage()
		os.Exit(1)
	}
	auth_token := args[0]
	user, err := github.FetchAuthenticatedUser(auth_token)
	if err != nil {
		log.Fatalf("fetch authenticated user failed: %v", err)
	}
	fmt.Printf("Authenticated as: %s\n", user.Login)

	user_model := &models.User{
		Username:    user.Login,
		AccessToken: auth_token,
	}

	if err := repo.SaveUser(user_model); err != nil {
		log.Fatalf("save user failed: %v", err)
	}
}


func dispatchPrsCommand(repo *repository.DatabaseRepository) {
	prs, err := repo.GetAllPrs()
	if err != nil {
		log.Fatalf("fetch prs failed: %v", err)
	}
	fmt.Printf("PRs:\n")
	for _, pr := range prs {
		fmt.Printf("- #%d: %s (Repository: %s, Author: %s)\n", pr.Number, pr.Title, pr.Repository, pr.Author)
	}
}

func dispatchSyncCommand(repo *repository.DatabaseRepository, token string) {
	fmt.Println("Syncing data...")
	
	repositories, err := repo.GetTrackedRepositories()
	if err != nil {
		log.Fatalf("fetch tracked repositories failed: %v", err)
	}
	if len(repositories) == 0 {
		fmt.Println("No repositories to sync")
		return
	}
	trackedAuthors, err := repo.GetTrackedAuthors()
	if err != nil {
		log.Fatalf("fetch tracked authors failed: %v", err)
	}
	if len(trackedAuthors) == 0 {
		fmt.Println("No authors to sync")
		return
	}

	for _, repository := range repositories {
		fmt.Printf("Syncing repository: %s\n", repository)
		prs, err := github.FetchOpenPullRequests(repository, token)
		if err != nil {
			log.Printf("fetch open prs for repository %s failed: %v", repository, err)
			continue
		}
		log.Printf("fetched %d open prs for repository %s", len(prs), repository)
		
		for _, pr := range prs {
			if core.ShouldTrackPR(&pr, trackedAuthors) {
				log.Printf("tracking pr #%d in repository %s", pr.Number, repository)
				prDetails, err := service.FetchPullRequestDetails(repository, pr.Number, token)
				if err != nil {
					log.Printf("fetch pr details for pr #%d in repository %s failed: %v", pr.Number, repository, err)
					continue
				}
				if err := repo.SavePr(prDetails); err != nil {
					log.Printf("save pr details for pr #%d in repository %s failed: %v", pr.Number, repository, err)
					continue
				}
			}
		}
	}
}



func dispatchRepositoriesCommand(repo *repository.DatabaseRepository, args []string) {
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		// Handle repositories list command
		displayRepositories(repo)
	case "add":
		// Handle repositories add command
		addRepository(repo, args[1:])
	case "remove":
		// Handle repositories remove command
		fmt.Println("Removing repository...")
		deleteRepository(repo, args[1:])
	default:
		fmt.Printf("Unknown repositories command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func deleteRepository(repo *repository.DatabaseRepository, args []string) {
	if len(args) < 1 {
		fmt.Println("Repository name is required")
		printUsage()
		os.Exit(1)
	}
	repository := args[0]

	if err := repo.DeleteTrackedRepository(repository); err != nil {
		log.Fatalf("delete repository failed: %v", err)
	}
	fmt.Printf("Repository '%s' deleted successfully\n", repository)
}

func displayRepositories(repo *repository.DatabaseRepository) {
	repositories, err := repo.GetTrackedRepositories()
	if err != nil {
		log.Fatalf("list repositories failed: %v", err)
	}

	fmt.Println("Repositories:")
	for _, repository := range repositories {
		fmt.Printf("- %s\n", repository)
	}
}

func addRepository(repo *repository.DatabaseRepository, args []string) {
	if len(args) < 1 {
		fmt.Println("Repository name is required")
		printUsage()
		os.Exit(1)
	}
	repository := args[0]

	if err := repo.SaveTrackedRepository(repository); err != nil {
		log.Fatalf("add repository failed: %v", err)
	}
	fmt.Printf("Repository '%s' added successfully\n", repository)
}


func dispatchAuthorsCommand(repo *repository.DatabaseRepository, args []string) {
	if len(args) < 1 {
		printUsage()
		os.Exit(1)
	}
	switch args[0] {
	case "list":
		// Handle authors list command
		displayAuthors(repo)
	case "add":
		// Handle authors add command
		addAuthor(repo, args[1:])
	case "remove":
		// Handle authors remove command
		fmt.Println("Removing author...")
	default:
		fmt.Printf("Unknown authors command: %s\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

func displayAuthors(repo *repository.DatabaseRepository) {
	authors, err := repo.GetTrackedAuthors()
	if err != nil {
		log.Fatalf("list authors failed: %v", err)
	}

	fmt.Println("Authors:")
	for _, author := range authors {
		fmt.Printf("- %s\n", author)
	}
}

func addAuthor(repo *repository.DatabaseRepository, args []string) {
	if len(args) < 1 {
		fmt.Println("Author login is required")
		printUsage()
		os.Exit(1)
	}
	login := args[0]

	if err := repo.SaveTrackedAuthor(login); err != nil {
		log.Fatalf("add author failed: %v", err)
	}
	fmt.Printf("Author '%s' added successfully\n", login)
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  cli authors <command>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  authors list    List authors")
	fmt.Println("  authors add     Add author")
	fmt.Println("  authors remove  Remove author")
}
