BEGIN TRANSACTION;

ALTER TABLE public.announcement DROP CONSTRAINT fk_announcement_created_by;

ALTER TABLE public.announcement ADD COLUMN admin BIGINT;

UPDATE public.announcement AS a
SET 
    admin = ad.id
FROM public.admin ad
WHERE a.created_by = ad.uid;

ALTER TABLE public.announcement DROP COLUMN created_by;

ALTER TABLE public.announcement ADD CONSTRAINT fk_announcement_admin FOREIGN KEY (admin) REFERENCES public.admin(id);

ALTER TABLE public.announcement DROP COLUMN updated_at;

ALTER TABLE public.announcement DROP CONSTRAINT fk_announcement_updated_by;

ALTER TABLE public.announcement DROP COLUMN updated_by;

COMMIT;