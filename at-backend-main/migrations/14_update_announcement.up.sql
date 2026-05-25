BEGIN TRANSACTION;

ALTER TABLE public.announcement ADD COLUMN updated_at TIMESTAMP NOT NULL DEFAULT (NOW() AT TIME ZONE 'utc');

ALTER TABLE public.announcement DROP CONSTRAINT fk_announcement_admin;

ALTER TABLE public.announcement ADD COLUMN created_by BIGINT;

UPDATE public.announcement
SET
    created_by = ad.uid
FROM public.admin AS ad
WHERE ad.id = admin;

ALTER TABLE public.announcement DROP COLUMN admin;

ALTER TABLE public.announcement ADD CONSTRAINT fk_announcement_created_by FOREIGN KEY (created_by) REFERENCES public.user(id) ON DELETE SET NULL;

ALTER TABLE public.announcement ADD updated_by BIGINT;

UPDATE public.announcement
SET
    updated_by = created_by;

ALTER TABLE public.announcement ADD CONSTRAINT fk_announcement_updated_by FOREIGN KEY (updated_by) REFERENCES public.user(id) ON DELETE SET NULL;

COMMIT;