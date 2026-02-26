package core

import (
	"testing"
	"time"

	"git.rileymathews.com/riley/pr-tracker/internal/models"
)

// newPR is a helper to construct a minimal PullRequest for tests.
func newPR(repo string, number int) *models.PullRequest {
	return &models.PullRequest{
		Repository: repo,
		Number:     number,
	}
}

// TestProcessPullRequestSyncResults_NewPR verifies that a PR present in the
// fresh sync but absent from the database is classified as new.
func TestProcessPullRequestSyncResults_NewPR(t *testing.T) {
	pr := newPR("acme/repo", 1)

	newPrs, updatedPrs, removedPrs := ProcessPullRequestSyncResults(
		nil,
		[]*models.PullRequest{pr},
	)

	if len(newPrs) != 1 || newPrs[0] != pr {
		t.Errorf("expected 1 new PR, got %d", len(newPrs))
	}
	if len(updatedPrs) != 0 {
		t.Errorf("expected 0 updated PRs, got %d", len(updatedPrs))
	}
	if len(removedPrs) != 0 {
		t.Errorf("expected 0 removed PRs, got %d", len(removedPrs))
	}
}

// TestProcessPullRequestSyncResults_UpdatedPR verifies that a PR present in
// both sources but with a relevant change is classified as updated.
func TestProcessPullRequestSyncResults_UpdatedPR(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	dbPR := &models.PullRequest{
		Repository:    "acme/repo",
		Number:        1,
		CiStatus:      models.CiStatusPending,
		LastCommentAt: base,
		LastCommitAt:  base,
	}
	freshPR := &models.PullRequest{
		Repository:    "acme/repo",
		Number:        1,
		CiStatus:      models.CiStatusSuccess, // CI status changed
		LastCommentAt: base,
		LastCommitAt:  base,
	}

	newPrs, updatedPrs, removedPrs := ProcessPullRequestSyncResults(
		[]*models.PullRequest{dbPR},
		[]*models.PullRequest{freshPR},
	)

	if len(newPrs) != 0 {
		t.Errorf("expected 0 new PRs, got %d", len(newPrs))
	}
	if len(updatedPrs) != 1 || updatedPrs[0] != freshPR {
		t.Errorf("expected 1 updated PR, got %d", len(updatedPrs))
	}
	if len(removedPrs) != 0 {
		t.Errorf("expected 0 removed PRs, got %d", len(removedPrs))
	}
}

// TestProcessPullRequestSyncResults_RemovedPR verifies that a PR present in
// the database but absent from the fresh sync is classified as removed.
func TestProcessPullRequestSyncResults_RemovedPR(t *testing.T) {
	pr := newPR("acme/repo", 1)

	newPrs, updatedPrs, removedPrs := ProcessPullRequestSyncResults(
		[]*models.PullRequest{pr},
		nil,
	)

	if len(newPrs) != 0 {
		t.Errorf("expected 0 new PRs, got %d", len(newPrs))
	}
	if len(updatedPrs) != 0 {
		t.Errorf("expected 0 updated PRs, got %d", len(updatedPrs))
	}
	if len(removedPrs) != 1 || removedPrs[0] != pr {
		t.Errorf("expected 1 removed PR, got %d", len(removedPrs))
	}
}

// TestProcessPullRequestSyncResults_Mixed feeds a variety of PRs and asserts
// that each one ends up in the correct output list.
//
// Setup:
//   - PR #1  exists in DB, no changes          → skipped (not in any output)
//   - PR #2  exists in DB, new commit           → updated
//   - PR #3  exists in DB, new comment          → updated
//   - PR #4  exists in DB, CI status changed    → updated (LastCiStatusUpdateAt refreshed)
//   - PR #5  not in DB                          → new
//   - PR #6  not in fresh sync                  → removed
func TestProcessPullRequestSyncResults_Mixed(t *testing.T) {
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	later := base.Add(time.Hour)

	// --- database state ---
	db1 := &models.PullRequest{Repository: "org/repo", Number: 1,
		CiStatus: models.CiStatusPending, LastCommentAt: base, LastCommitAt: base}
	db2 := &models.PullRequest{Repository: "org/repo", Number: 2,
		CiStatus: models.CiStatusPending, LastCommentAt: base, LastCommitAt: base}
	db3 := &models.PullRequest{Repository: "org/repo", Number: 3,
		CiStatus: models.CiStatusPending, LastCommentAt: base, LastCommitAt: base}
	db4 := &models.PullRequest{Repository: "org/repo", Number: 4,
		CiStatus: models.CiStatusPending, LastCommentAt: base, LastCommitAt: base}
	db6 := &models.PullRequest{Repository: "org/repo", Number: 6,
		CiStatus: models.CiStatusPending, LastCommentAt: base, LastCommitAt: base}

	// --- fresh sync ---
	fresh1 := &models.PullRequest{Repository: "org/repo", Number: 1,
		CiStatus: models.CiStatusPending, LastCommentAt: base, LastCommitAt: base} // unchanged
	fresh2 := &models.PullRequest{Repository: "org/repo", Number: 2,
		CiStatus: models.CiStatusPending, LastCommentAt: base, LastCommitAt: later} // new commit
	fresh3 := &models.PullRequest{Repository: "org/repo", Number: 3,
		CiStatus: models.CiStatusPending, LastCommentAt: later, LastCommitAt: base} // new comment
	fresh4 := &models.PullRequest{Repository: "org/repo", Number: 4,
		CiStatus: models.CiStatusFailure, LastCommentAt: base, LastCommitAt: base} // CI changed
	fresh5 := &models.PullRequest{Repository: "org/repo", Number: 5,
		CiStatus: models.CiStatusSuccess, LastCommentAt: base, LastCommitAt: base} // brand new

	dbPRs := []*models.PullRequest{db1, db2, db3, db4, db6}
	freshPRs := []*models.PullRequest{fresh1, fresh2, fresh3, fresh4, fresh5}

	newPrs, updatedPrs, removedPrs := ProcessPullRequestSyncResults(dbPRs, freshPRs)

	// --- new ---
	if len(newPrs) != 1 {
		t.Fatalf("expected 1 new PR, got %d", len(newPrs))
	}
	if newPrs[0].Number != 5 {
		t.Errorf("expected new PR #5, got #%d", newPrs[0].Number)
	}

	// --- updated ---
	if len(updatedPrs) != 3 {
		t.Fatalf("expected 3 updated PRs, got %d", len(updatedPrs))
	}
	updatedNumbers := map[int]bool{}
	for _, pr := range updatedPrs {
		updatedNumbers[pr.Number] = true
	}
	for _, n := range []int{2, 3, 4} {
		if !updatedNumbers[n] {
			t.Errorf("expected PR #%d in updated list", n)
		}
	}

	// PR #4 had a CI status change, so LastCiStatusUpdateAt should have been set.
	for _, pr := range updatedPrs {
		if pr.Number == 4 && pr.LastCiStatusUpdateAt.IsZero() {
			t.Error("expected LastCiStatusUpdateAt to be set for PR #4 after CI status change")
		}
	}

	// --- removed ---
	if len(removedPrs) != 1 {
		t.Fatalf("expected 1 removed PR, got %d", len(removedPrs))
	}
	if removedPrs[0].Number != 6 {
		t.Errorf("expected removed PR #6, got #%d", removedPrs[0].Number)
	}
}
