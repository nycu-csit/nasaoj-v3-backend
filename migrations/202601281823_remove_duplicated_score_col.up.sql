ALTER TABLE IF EXISTS "submission"
  DROP COLUMN IF EXISTS "score_score";

--bun:split

ALTER TABLE IF EXISTS "submission"
  DROP COLUMN IF EXISTS "score_stages";
