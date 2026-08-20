CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE organization_memberships (
    org_id TEXT NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (org_id, user_id)
);

CREATE INDEX organization_memberships_user_idx ON organization_memberships (user_id);

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS active_org_id TEXT NOT NULL DEFAULT '';
