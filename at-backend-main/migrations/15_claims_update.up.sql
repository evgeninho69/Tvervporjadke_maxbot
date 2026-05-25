BEGIN TRANSACTION;

ALTER TABLE public.claim ALTER COLUMN uid DROP NOT NULL;

ALTER TABLE public.claim
DROP CONSTRAINT fk_claim_uid;

ALTER TABLE public.claim RENAME COLUMN uid TO created_by;

ALTER TABLE public.claim
ADD CONSTRAINT fk_claim_created_by
FOREIGN KEY (created_by) REFERENCES public.user(id)
ON DELETE SET NULL;

COMMIT;