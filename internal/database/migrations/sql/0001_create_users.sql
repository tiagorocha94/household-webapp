CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Login sessions. Only a hash of the token is stored, so a copy of the
-- database cannot be used to log in. expires_at is Unix seconds.
CREATE TABLE IF NOT EXISTS user_sessions (
    token_hash TEXT PRIMARY KEY,
    user_id INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_expires_at
ON user_sessions (expires_at);

-- Phones that may write data. A row starts as a pending enrolment (a hashed
-- one-time code and its expiry) and becomes enrolled when the phone exchanges
-- the code for a token, of which only the hash is stored, so a copy of the
-- database cannot be used to write. Times are Unix seconds. A revoked device
-- keeps its row and loses both secrets. A person has at most one device that
-- is not revoked, and an active name is unique. Enrolled devices are also
-- written to devices.csv so that a rebuilt database keeps them.
CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    name TEXT NOT NULL COLLATE NOCASE,
    enrol_code_hash TEXT UNIQUE,
    enrol_expires_at INTEGER,
    token_hash TEXT UNIQUE,
    created_at INTEGER NOT NULL,
    enrolled_at INTEGER,
    last_seen_at INTEGER,
    revoked_at INTEGER,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CHECK ((enrol_code_hash IS NULL) = (enrol_expires_at IS NULL))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_active_user
ON devices (user_id) WHERE revoked_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_devices_active_name
ON devices (name) WHERE revoked_at IS NULL;
