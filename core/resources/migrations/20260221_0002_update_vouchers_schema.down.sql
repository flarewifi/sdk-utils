CREATE TABLE vouchers_old (
    id INTEGER PRIMARY KEY,
    code VARCHAR(10) NOT NULL DEFAULT '',
    provider_pkg VARCHAR(255) NOT NULL DEFAULT '',
    validity_count INT NOT NULL DEFAULT 0,
    validity_unit INT NOT NULL DEFAULT 1,
    speed INT NOT NULL DEFAULT 0,
    status INT NOT NULL DEFAULT 1,
    device_ip VARCHAR(17) NOT NULL DEFAULT 'N/A',
    session_id INTEGER REFERENCES sessions(id) ON DELETE SET NULL,
    device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

INSERT INTO vouchers_old (id, code, provider_pkg, validity_count, validity_unit, status, session_id, device_id, created_at)
SELECT
    id,
    code,
    provider_pkg,
    time_secs / 3600,
    1,
    CASE WHEN activated_at IS NOT NULL THEN 2 ELSE 1 END,
    session_id,
    device_id,
    created_at
FROM vouchers;

DROP TABLE vouchers;
ALTER TABLE vouchers_old RENAME TO vouchers;
