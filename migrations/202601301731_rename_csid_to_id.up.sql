ALTER TABLE IF EXISTS "submission"
  RENAME "cs_id" TO "owner_id";

--bun:split

ALTER TABLE IF EXISTS "session"
  RENAME "cs_id" TO "id";
