DROP TABLE IF EXISTS public.refresh_token;
DROP TABLE IF EXISTS public.mail_token;

CREATE TABLE "refresh_tokens" (
    id SERIAL PRIMARY KEY,
    uid BIGINT NOT NULL,
    hash TEXT UNIQUE NOT NULL,
    salt TEXT NOT NULL,
    expire TIMESTAMP NOT NULL
);

CREATE INDEX "idx_refresh_tokens_hash" ON public.refresh_tokens (hash);

ALTER TABLE public.refresh_tokens ADD CONSTRAINT "fk_refresh_tokens_uid" FOREIGN KEY (uid) REFERENCES public.user (id) ON DELETE CASCADE;