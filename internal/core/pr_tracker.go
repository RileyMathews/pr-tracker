package core

import (
	"slices"

	"git.rileymathews.com/riley/pr-tracker/internal/github"
)

func ShouldTrackPR(pr *github.PullRequest, authorsToTrack []string) bool {
	if slices.Contains(authorsToTrack, pr.User.Login) {
		return true
	}

	return false
}
