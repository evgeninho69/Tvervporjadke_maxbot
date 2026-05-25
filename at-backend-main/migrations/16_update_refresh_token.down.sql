BEGIN TRANSACTION;

DELETE FROM public.refresh_tokens;

ALTER TABLE public.refresh_tokens DROP COLUMN id;

ALTER TABLE public.refresh_tokens ADD COLUMN id SERIAL NOT NULL;

ALTER TABLE public.refresh_tokens ADD PRIMARY KEY (id);

ALTER TABLE public.refresh_tokens DROP COLUMN revoked;

ALTER TABLE public.refresh_tokens RENAME COLUMN user_id TO uid;

COMMIT;