package main

import (
	"log"
	"os"

	gh "git.rileymathews.com/riley/pr-tracker/internal/github"
)

func main() {
	token := os.Getenv("LOCAL_GH_TOKEN")
	if token == "" {
		log.Fatal("LOCAL_GH_TOKEN environment variable is required")
	}

	const repoName = "MercuryTechnologies/mercury-web-backend"
	const prID = 65326

	// openPRs, err := gh.FetchOpenPullRequests(repoName, token)
	// if err != nil {
	// 	log.Fatalf("fetch open PRs failed: %v", err)
	// }
	// log.Printf("Open PRs response (%d PRs): %+v", len(openPRs), openPRs)
	//
	// prDetails, err := gh.FetchPullRequestDetails(repoName, prID, token)
	// if err != nil {
	// 	log.Fatalf("fetch PR details failed: %v", err)
	// }
	// log.Printf("PR details response: %+v", prDetails)

	ciStatuses, err := gh.FetchPullRequestCIStatuses(repoName, prID, token)
	if err != nil {
		log.Fatalf("fetch PR CI statuses failed: %v", err)
	}
	log.Printf("PR CI statuses response: %+v", ciStatuses)
}
