-- +goose Up
ALTER TABLE companies ADD COLUMN name text NOT NULL;
ALTER TABLE companies ADD COLUMN slug text NOT NULL;

CREATE UNIQUE INDEX ux_companies_slug ON companies (slug);

-- +goose Down
DROP INDEX ux_companies_slug;
ALTER TABLE companies DROP COLUMN slug;
ALTER TABLE companies DROP COLUMN name;
