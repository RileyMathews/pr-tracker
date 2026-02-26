CREATE TABLE IF NOT EXISTS pull_requests (
  number INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  repository TEXT NOT NULL,
  author TEXT NOT NULL,
  draft INTEGER NOT NULL,
  created_at_unix INTEGER NOT NULL,
  updated_at_unix INTEGER NOT NULL,
  ci_status INTEGER NOT NULL,
  last_comment_unix INTEGER NOT NULL,
  last_commit_unix INTEGER NOT NULL,
  ci_status_last_updated_at_unix INTEGER NOT NULL,
  last_acknowledged_unix INTEGER
);
