package main

import (
	"log"
	"os"

	"git.rileymathews.com/riley/pr-tracker/internal/service"
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
}
