BEGIN TRANSACTION;

ALTER TABLE public.claim
DROP CONSTRAINT fk_claim_created_by;

ALTER TABLE public.claim RENAME COLUMN created_by TO uid;

UPDATE public.claim
SET uid = 1
WHERE uid IS NULL;

ALTER TABLE public.claim ALTER COLUMN uid SET NOT NULL;

ALTER TABLE public.claim ADD CONSTRAINT fk_claim_uid FOREIGN KEY (uid) REFERENCES public.uid(id);

ALTER TABLE public.claim RENAME COLUMN created_by TO uid;

COMMIT;