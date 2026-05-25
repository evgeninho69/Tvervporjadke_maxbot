UPDATE public.announcement
SET admin = 1
WHERE admin IS NULL;

ALTER TABLE public.announcement ALTER COLUMN admin SET NOT NULL;

ALTER TABLE announcement
DROP CONSTRAINT fk_announcement_admin;

ALTER TABLE public.announcement ADD CONSTRAINT fk_announcement_admin FOREIGN KEY (admin) REFERENCES public.admin(id);
