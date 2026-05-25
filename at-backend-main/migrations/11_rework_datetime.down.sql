DROP INDEX idx_claim_status_updated_at;

ALTER TABLE public.claim DROP COLUMN status_updated_at;

ALTER TABLE public.claim DROP COLUMN feedback_updated_at;

ALTER TABLE public.claim DROP COLUMN updated_at;

ALTER TABLE public.user DROP COLUMN registered_at;

ALTER TABLE public.claim RENAME COLUMN created_at TO datetime;

ALTER TABLE public.announcement RENAME COLUMN created_at TO datetime;

