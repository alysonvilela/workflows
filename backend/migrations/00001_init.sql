-- +goose Up
CREATE TABLE companies (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_context  jsonb NOT NULL DEFAULT '[]'::jsonb,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE documents (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id  uuid NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    s3_key      text NOT NULL,
    title       text NOT NULL,
    kind        text NOT NULL CHECK (kind IN ('markdown', 'csv')),
    raw_text    text,
    summary     jsonb,
    status      text NOT NULL DEFAULT 'parsing' CHECK (status IN ('parsing', 'ready', 'failed')),
    error       text,
    checksum    text NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_documents_company_id ON documents (company_id);

CREATE TABLE runs (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id         uuid NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    theme              text,
    opportunity_count  int NOT NULL,
    max_depth          int NOT NULL,
    max_nodes          int NOT NULL,
    status             text NOT NULL DEFAULT 'running'
        CHECK (status IN ('running', 'awaiting_input', 'synthesizing', 'done', 'failed')),
    result             jsonb,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX ix_runs_company_id ON runs (company_id);

CREATE TABLE nodes (
    id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id      uuid NOT NULL REFERENCES runs (id) ON DELETE CASCADE,
    parent_id   uuid REFERENCES nodes (id) ON DELETE CASCADE,
    ord         int NOT NULL DEFAULT 0,
    kind        text NOT NULL,
    status      text NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'running', 'blocked', 'done', 'failed')),
    depth       int NOT NULL DEFAULT 0,
    input       jsonb,
    output      jsonb,
    error       text,
    claimed_at  timestamptz,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_nodes_parent_kind_ord ON nodes (parent_id, kind, ord);
CREATE INDEX ix_nodes_run_id ON nodes (run_id);
CREATE INDEX ix_nodes_run_status ON nodes (run_id, status);

-- +goose Down
DROP TABLE nodes;
DROP TABLE runs;
DROP TABLE documents;
DROP TABLE companies;
