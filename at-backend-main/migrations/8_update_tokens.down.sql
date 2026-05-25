DROP TABLE IF EXISTS "refresh_tokens"

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

