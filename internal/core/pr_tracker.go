package core

import (
	"strconv"
	"time"

	"git.rileymathews.com/riley/pr-tracker/internal/models"
)


func ProcessPullRequestSyncResults(prsFromDatabase, prsFromFreshSync []*models.PullRequest) (newPrs, updatedPrs, removedPrs []*models.PullRequest) {
	dbByKey := indexPullRequestsByKey(prsFromDatabase)
	seen := make(map[string]struct{}, len(prsFromFreshSync))
	now := time.Now().UTC()

	for _, incomingPr := range prsFromFreshSync {
		if incomingPr == nil {
			continue
		}

		key := pullRequestKey(incomingPr)
		seen[key] = struct{}{}

		existingPr, exists := dbByKey[key]
		if !exists {
			newPrs = append(newPrs, incomingPr)
			continue
		}

		ciStatusChanged, hasRelevantChanges := pullRequestHasRelevantChanges(existingPr, incomingPr)
		if !hasRelevantChanges {
			continue
		}

		applySyncMetadata(existingPr, incomingPr, ciStatusChanged, now)
		updatedPrs = append(updatedPrs, incomingPr)
	}

	removedPrs = collectRemovedPullRequests(dbByKey, seen)

	return newPrs, updatedPrs, removedPrs
}

func pullRequestKey(pr *models.PullRequest) string {
	return pr.Repository + "#" + strconv.Itoa(pr.Number)
}

func indexPullRequestsByKey(prs []*models.PullRequest) map[string]*models.PullRequest {
	byKey := make(map[string]*models.PullRequest, len(prs))
	for _, pr := range prs {
		if pr == nil {
			continue
		}
		byKey[pullRequestKey(pr)] = pr
	}
	return byKey
}

func pullRequestHasRelevantChanges(existingPr, incomingPr *models.PullRequest) (ciStatusChanged, hasRelevantChanges bool) {
	ciStatusChanged = existingPr.CiStatus != incomingPr.CiStatus
	lastCommentChanged := !existingPr.LastCommentAt.Equal(incomingPr.LastCommentAt)
	lastCommitChanged := !existingPr.LastCommitAt.Equal(incomingPr.LastCommitAt)

	hasRelevantChanges = ciStatusChanged || lastCommentChanged || lastCommitChanged
	return ciStatusChanged, hasRelevantChanges
}

func applySyncMetadata(existingPr, incomingPr *models.PullRequest, ciStatusChanged bool, now time.Time) {
	incomingPr.LastAcknowledgedAt = existingPr.LastAcknowledgedAt
	if ciStatusChanged {
		incomingPr.LastCiStatusUpdateAt = now
		return
	}

	incomingPr.LastCiStatusUpdateAt = existingPr.LastCiStatusUpdateAt
}

func collectRemovedPullRequests(dbByKey map[string]*models.PullRequest, seen map[string]struct{}) []*models.PullRequest {
	removedPrs := make([]*models.PullRequest, 0)
	for key, existingPr := range dbByKey {
		if _, ok := seen[key]; ok {
			continue
		}
		removedPrs = append(removedPrs, existingPr)
	}
	return removedPrs
}
