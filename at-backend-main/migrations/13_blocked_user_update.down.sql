ALTER TABLE public.blocked_user RENAME COLUMN blocked_at TO block_date;

ALTER TABLE public.blocked_user DROP COLUMN blocked_by;
