ALTER TABLE sessions ADD COLUMN sid TEXT NOT NULL DEFAULT '';
CREATE INDEX sessions_sid ON sessions (sid);

ALTER TABLE auth_codes ADD COLUMN session_sid TEXT NOT NULL DEFAULT '';
ALTER TABLE refresh_tokens ADD COLUMN session_sid TEXT NOT NULL DEFAULT '';

ALTER TABLE clients ADD COLUMN frontchannel_logout_uri TEXT NOT NULL DEFAULT '';

CREATE TABLE session_clients (
    sid TEXT NOT NULL,
    client_id TEXT NOT NULL,
    PRIMARY KEY (sid, client_id)
);
