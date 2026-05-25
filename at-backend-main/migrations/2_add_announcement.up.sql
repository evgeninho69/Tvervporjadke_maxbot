CREATE TABLE "announcement" (
    "id" SERIAL PRIMARY KEY,
    "title" TEXT NOT NULL,
    "description" TEXT NOT NULL,
    "datetime" TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'),
    "admin" BIGINT NOT NULL
);

ALTER TABLE "announcement" ADD CONSTRAINT "fk_announcement_admin" FOREIGN KEY ("admin") REFERENCES "admin" ("id");
