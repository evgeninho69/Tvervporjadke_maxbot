DROP INDEX "idx_claim_lat";
DROP INDEX "idx_claim_long";

ALTER TABLE claim ALTER COLUMN latitude SET DATA TYPE TEXT USING cast(latitude as TEXT);
ALTER TABLE claim ALTER COLUMN longitude SET DATA TYPE TEXT USING cast(longitude as TEXT);