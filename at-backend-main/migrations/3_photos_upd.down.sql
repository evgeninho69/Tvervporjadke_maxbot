DROP TABLE "claim";

CREATE TABLE "claim" (
    "id" SERIAL PRIMARY KEY,
    "uid" BIGINT NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "category" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "photo" TEXT,
    "latitude" TEXT NOT NULL,
    "longitude" TEXT NOT NULL,
    "feedback" TEXT DEFAULT '',
    "datetime" TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

CREATE INDEX "idx_claim_uid" ON "claim" ("uid");

ALTER TABLE "claim" ADD CONSTRAINT "fk_claim_uid" FOREIGN KEY ("uid") REFERENCES "user" ("id") ON DELETE CASCADE;
