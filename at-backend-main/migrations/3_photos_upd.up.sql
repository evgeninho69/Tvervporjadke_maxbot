CREATE TABLE IF NOT EXISTS "new_claim" (
    "id" SERIAL PRIMARY KEY,
    "uid" BIGINT NOT NULL,
    "title" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "category" TEXT NOT NULL,
    "status" TEXT NOT NULL,
    "photos" text[],
    "latitude" TEXT NOT NULL,
    "longitude" TEXT NOT NULL,
    "feedback" TEXT DEFAULT '',
    "datetime" TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc')
);

INSERT INTO "new_claim" (
    uid, 
    title, 
    description, 
    category, 
    status, 
    photos, 
    latitude, 
    longitude, 
    feedback, 
    datetime
)
SELECT 
    uid, 
    title,
    description, 
    category, 
    status, 
    ARRAY[photo] AS photos,
    latitude,
    longitude, 
    feedback,
    datetime
FROM "claim";

DROP TABLE "claim";

ALTER TABLE "new_claim" RENAME TO "claim";

CREATE INDEX "idx_claim_uid" ON "claim" ("uid");

ALTER TABLE "claim" ADD CONSTRAINT "fk_claim_uid" FOREIGN KEY ("uid") REFERENCES "user" ("id") ON DELETE CASCADE;
