-- Remove session_exp_days and use_global columns from vouchers table
-- SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
CREATE TABLE vouchers_backup (
    id INTEGER PRIMARY KEY,
    code VARCHAR(10) NOT NULL DEFAULT '',
    provider_pkg VARCHAR(255) NOT NULL DEFAULT '',
    session_type TEXT NOT NULL DEFAULT 'time',
    time_secs INT NOT NULL DEFAULT 0,
    data_mb INT NOT NULL DEFAULT 0,
    down_speed_mbps INT NOT NULL DEFAULT 0,
    up_speed_mbps INT NOT NULL DEFAULT 0,
    expires_on TIMESTAMP,
    session_id INTEGER REFERENCES sessions(id) ON DELETE SET NULL,
    device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
    activated_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO vouchers_backup (id, code, provider_pkg, session_type, time_secs, data_mb, down_speed_mbps, up_speed_mbps, expires_on, session_id, device_id, activated_at, created_at)
SELECT id, code, provider_pkg, session_type, time_secs, data_mb, down_speed_mbps, up_speed_mbps, expires_on, session_id, device_id, activated_at, created_at
FROM vouchers;

DROP TABLE vouchers;
ALTER TABLE vouchers_backup RENAME TO vouchers;
