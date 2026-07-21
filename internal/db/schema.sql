-- VMT schema. Applied idempotently on startup.
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    token      TEXT PRIMARY KEY,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS vehicles (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          TEXT NOT NULL,
    make          TEXT NOT NULL DEFAULT '',
    model         TEXT NOT NULL DEFAULT '',
    year          INTEGER,
    vin           TEXT NOT NULL DEFAULT '',
    license_plate TEXT NOT NULL DEFAULT '',
    color         TEXT NOT NULL DEFAULT '',
    odometer      INTEGER NOT NULL DEFAULT 0,
    purchase_date TEXT,
    notes         TEXT NOT NULL DEFAULT '',
    photo_id      INTEGER REFERENCES attachments(id) ON DELETE SET NULL,
    archived      INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS service_records (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    vehicle_id  INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    date        TEXT NOT NULL,
    odometer    INTEGER,
    category    TEXT NOT NULL DEFAULT 'Other',
    description TEXT NOT NULL,
    vendor      TEXT NOT NULL DEFAULT '',
    cost        REAL NOT NULL DEFAULT 0,
    notes       TEXT NOT NULL DEFAULT '',
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_service_vehicle ON service_records(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_service_date ON service_records(date);

CREATE TABLE IF NOT EXISTS reminders (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    vehicle_id      INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    due_date        TEXT,
    due_odometer    INTEGER,
    interval_months INTEGER,
    interval_miles  INTEGER,
    notes           TEXT NOT NULL DEFAULT '',
    completed       INTEGER NOT NULL DEFAULT 0,
    notify          INTEGER NOT NULL DEFAULT 0,
    last_notified   TEXT,
    created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reminder_vehicle ON reminders(vehicle_id);

CREATE TABLE IF NOT EXISTS reference_items (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    vehicle_id   INTEGER NOT NULL REFERENCES vehicles(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL DEFAULT 'part', -- 'part' or 'fluid'
    name         TEXT NOT NULL,
    part_number  TEXT NOT NULL DEFAULT '',
    manufacturer TEXT NOT NULL DEFAULT '',
    capacity     TEXT NOT NULL DEFAULT '',
    spec         TEXT NOT NULL DEFAULT '',
    notes        TEXT NOT NULL DEFAULT '',
    position     INTEGER NOT NULL DEFAULT 0,
    created_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_reference_vehicle ON reference_items(vehicle_id);

CREATE TABLE IF NOT EXISTS attachments (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    vehicle_id    INTEGER REFERENCES vehicles(id) ON DELETE CASCADE,
    service_id    INTEGER REFERENCES service_records(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL DEFAULT 'document', -- 'photo' or 'document'
    stored_name   TEXT NOT NULL,
    original_name TEXT NOT NULL,
    content_type  TEXT NOT NULL DEFAULT '',
    size          INTEGER NOT NULL DEFAULT 0,
    created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_attach_vehicle ON attachments(vehicle_id);
CREATE INDEX IF NOT EXISTS idx_attach_service ON attachments(service_id);
