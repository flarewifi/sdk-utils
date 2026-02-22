CREATE TABLE vouchers_new (
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

INSERT INTO vouchers_new (id, code, provider_pkg, session_type, time_secs, session_id, device_id, activated_at, created_at)
SELECT
    id,
    code,
    provider_pkg,
    'time',
    CASE validity_unit
        WHEN 1 THEN validity_count * 3600
        WHEN 2 THEN validity_count * 86400
        WHEN 3 THEN validity_count * 2592000
        ELSE 0
    END,
    session_id,
    device_id,
    CASE WHEN status = 2 THEN CURRENT_TIMESTAMP ELSE NULL END,
    created_at
FROM vouchers;

DROP TABLE vouchers;
ALTER TABLE vouchers_new RENAME TO vouchers;
