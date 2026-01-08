CREATE TABLE IF NOT EXISTS device_fingerprints (
    id INTEGER PRIMARY KEY,
    device_id INTEGER NOT NULL,
    fingerprint_hash VARCHAR(64) NOT NULL,
    user_agent TEXT NOT NULL DEFAULT '',
    browser_name VARCHAR(50) NOT NULL DEFAULT '',
    os_family VARCHAR(50) NOT NULL DEFAULT '',
    screen_resolution VARCHAR(20) NOT NULL DEFAULT '',
    language VARCHAR(10) NOT NULL DEFAULT '',
    is_cna BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fingerprints_device_id ON device_fingerprints(device_id);
CREATE INDEX IF NOT EXISTS idx_fingerprints_hash ON device_fingerprints(fingerprint_hash);
CREATE INDEX IF NOT EXISTS idx_fingerprints_created ON device_fingerprints(created_at);
CREATE INDEX IF NOT EXISTS idx_fingerprints_device_hash ON device_fingerprints(device_id, fingerprint_hash);
