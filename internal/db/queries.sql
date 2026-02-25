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
  last_acknowledged_unix
) VALUES (
  ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)
ON CONFLICT(number) DO UPDATE SET
  title = excluded.title,
  repository = excluded.repository,
  author = excluded.author,
  draft = excluded.draft,
  created_at_unix = excluded.created_at_unix,
  updated_at_unix = excluded.updated_at_unix,
  ci_status = excluded.ci_status,
  last_comment_unix = excluded.last_comment_unix,
  last_commit_unix = excluded.last_commit_unix,
  last_acknowledged_unix = excluded.last_acknowledged_unix;

-- name: GetPullRequestByNumber :one
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
  last_acknowledged_unix
FROM pull_requests
WHERE number = ?
LIMIT 1;
