CREATE TABLE "user" (
    "id" SERIAL PRIMARY KEY,
    "first_name" TEXT NOT NULL,
    "second_name" TEXT NOT NULL,
    "email" TEXT UNIQUE NOT NULL
);

CREATE INDEX "idx_user_email" ON "user" ("email");

CREATE TABLE "admin" (
    "id" SERIAL PRIMARY KEY,
    "uid" BIGINT UNIQUE NOT NULL
);

CREATE INDEX "idx_admin_uid" ON "admin" ("uid");

ALTER TABLE "admin" ADD CONSTRAINT "fk_admin_user" FOREIGN KEY ("uid") REFERENCES "user" ("id");

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

CREATE TABLE "mail_token" (
    "id" SERIAL PRIMARY KEY,
    "token" TEXT UNIQUE,
    "uid" BIGINT NOT NULL
);

CREATE INDEX "idx_mail_token_uid" ON "mail_token" ("uid");

ALTER TABLE "mail_token" ADD CONSTRAINT "fk_mail_token_uid" FOREIGN KEY ("uid") REFERENCES "user" ("id") ON DELETE CASCADE;

CREATE TABLE "refresh_token" (
    "id" SERIAL PRIMARY KEY,
    "token" TEXT UNIQUE,
    "uid" BIGINT NOT NULL
);

CREATE INDEX "idx_refresh_token_uid" ON "refresh_token" ("uid");

ALTER TABLE "refresh_token" ADD CONSTRAINT "fk_refresh_token_uid" FOREIGN KEY ("uid") REFERENCES "user" ("id") ON DELETE CASCADE

