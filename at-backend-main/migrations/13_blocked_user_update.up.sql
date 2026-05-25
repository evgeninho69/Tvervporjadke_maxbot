ALTER TABLE public.blocked_user RENAME COLUMN block_date TO blocked_at;

ALTER TABLE public.blocked_user ADD COLUMN blocked_by BIGINT;

ALTER TABLE public.blocked_user ADD CONSTRAINT fk_blocked_by FOREIGN KEY (blocked_by) REFERENCES public.user(id) ON DELETE SET NULL;