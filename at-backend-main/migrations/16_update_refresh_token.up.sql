BEGIN TRANSACTION;

DELETE FROM public.refresh_tokens;

ALTER TABLE public.refresh_tokens DROP COLUMN id;

ALTER TABLE public.refresh_tokens ADD COLUMN id UUID NOT NULL DEFAULT gen_random_uuid();

ALTER TABLE public.refresh_tokens ADD PRIMARY KEY (id);

ALTER TABLE public.refresh_tokens ADD COLUMN revoked BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE public.refresh_tokens RENAME COLUMN uid TO user_id;

COMMIT;