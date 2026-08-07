-- +goose Up
CREATE UNIQUE INDEX ux_documents_company_checksum ON documents (company_id, checksum);

-- +goose Down
DROP INDEX ux_documents_company_checksum;
