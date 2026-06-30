-- Optional scope on a personal access token. NULL = full access (inherits the
-- user's permissions, current behaviour). 'tracker' = restricted to the activity
-- tracker endpoints, used by the browser extension token.
ALTER TABLE personal_access_tokens ADD COLUMN IF NOT EXISTS scope TEXT;
