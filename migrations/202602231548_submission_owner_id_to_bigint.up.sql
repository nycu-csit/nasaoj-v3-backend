DELETE FROM "submission";

--bun:split

ALTER TABLE IF EXISTS "submission"
  ALTER COLUMN "owner_id" TYPE BIGINT USING "owner_id"::BIGINT;
