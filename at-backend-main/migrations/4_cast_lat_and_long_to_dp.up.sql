ALTER TABLE claim ALTER COLUMN latitude SET DATA TYPE double precision USING cast(latitude as double precision);
ALTER TABLE claim ALTER COLUMN longitude SET DATA TYPE double precision USING cast(longitude as double precision);

CREATE INDEX "idx_claim_lat" ON "claim" ("latitude");
CREATE INDEX "idx_claim_long" ON "claim" ("longitude");