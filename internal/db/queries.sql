-- name: UpsertPullRequest :exec
INSERT INTO pull_requests (
  number,
  title,
  repository,
  author,
  draft,
  created_at_unix,
  updated_at_unix,
  ci_status,
  last_comment_unix,
  last_commit_unix,
  last_ci_status_update_unix,
  last_acknowledged_unix
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(repository, number) DO UPDATE SET
  title = excluded.title,
  repository = excluded.repository,
  author = excluded.author,
  draft = excluded.draft,
  updated_at_unix = excluded.updated_at_unix,
  ci_status = excluded.ci_status,
  last_comment_unix = excluded.last_comment_unix,
  last_commit_unix = excluded.last_commit_unix,
  last_ci_status_update_unix = excluded.last_ci_status_update_unix,
  last_acknowledged_unix = excluded.last_acknowledged_unix;

-- name: GetAllPullRequests :many
SELECT
  number,
  title,
  repository,
  author,
  draft,
  created_at_unix,
  updated_at_unix,
  ci_status,
  last_comment_unix,
  last_commit_unix,
  last_ci_status_update_unix,
  last_acknowledged_unix
FROM pull_requests;

-- name: GetPullRequestByRepoAndNumber :one
SELECT
  number,
  title,
  repository,
  author,
  draft,
  created_at_unix,
  updated_at_unix,
  ci_status,
  last_comment_unix,
  last_commit_unix,
  last_ci_status_update_unix,
  last_acknowledged_unix
FROM pull_requests
WHERE repository = ?
AND number = ?
LIMIT 1;

-- name: SaveTrackedAuthor :exec
INSERT INTO tracked_authors (author) VALUES (?);

-- name: GetTrackedAuthors :many
SELECT author FROM tracked_authors;

-- name: SaveTrackedRepository :exec
INSERT INTO tracked_repositories (repository) VALUES (?);

-- name: GetTrackedRepositories :many
SELECT repository FROM tracked_repositories;

-- name: DeleteTrackedRepository :exec
DELETE FROM tracked_repositories
WHERE repository = ?;
