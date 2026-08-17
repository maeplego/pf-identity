CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    at TIMESTAMPTZ NOT NULL,
    subject TEXT NOT NULL DEFAULT '',
    client_id TEXT NOT NULL DEFAULT '',
    ip TEXT NOT NULL DEFAULT '',
    note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX audit_events_at_desc ON audit_events (at DESC);
