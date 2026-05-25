ALTER TABLE public.claim RENAME COLUMN datetime TO created_at;

ALTER tABLE public.claim ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc');

ALTER TABLE public.claim ADD COLUMN status_updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc');

CREATE INDEX idx_claim_status_updated_at ON public.claim (status_updated_at);

ALTER TABLE public.claim ADD COLUMN feedback_updated_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'); 

ALTER TABLE public.announcement RENAME COLUMN datetime TO created_at;

ALTER TABLE public.user ADD COLUMN registered_at TIMESTAMP NOT NULL DEFAULT (now() AT TIME ZONE 'utc'); 