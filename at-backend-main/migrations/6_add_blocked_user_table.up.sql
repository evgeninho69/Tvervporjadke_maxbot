CREATE TABLE "blocked_user" (
	"id" SERIAL PRIMARY KEY,
	"uid" BIGINT UNIQUE NOT NULL,
	"reason" TEXT DEFAULT '',
	"block_date" TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

ALTER TABLE "blocked_user" ADD CONSTRAINT "fk_blocked_user_uid" FOREIGN KEY ("uid") REFERENCES "user" ("id") ON DELETE CASCADE;

CREATE INDEX "idx_blocked_user_uid" ON "blocked_user" ("uid");