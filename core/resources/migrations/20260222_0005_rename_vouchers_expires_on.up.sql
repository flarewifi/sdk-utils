-- Recreate vouchers table with expires_at field
-- Handles both cases: source table has expires_on or expires_at

-- Step 1: Backup existing vouchers table
ALTER TABLE vouchers RENAME TO vouchers_backup;

-- Step 2: Create new vouchers table with correct schema
CREATE TABLE vouchers (
    id INTEGER PRIMARY KEY,
    code VARCHAR(10) NOT NULL DEFAULT '',
    provider_pkg VARCHAR(255) NOT NULL DEFAULT '',
    session_type TEXT NOT NULL DEFAULT 'time',
    time_secs INT NOT NULL DEFAULT 0,
    data_mb INT NOT NULL DEFAULT 0,
    down_speed_mbps INT NOT NULL DEFAULT 0,
    up_speed_mbps INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMP,
    session_id INTEGER REFERENCES sessions(id) ON DELETE SET NULL,
    device_id INTEGER REFERENCES devices(id) ON DELETE SET NULL,
    activated_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_exp_days INTEGER,
    use_global INTEGER NOT NULL DEFAULT 0,
    uuid VARCHAR(36) NOT NULL DEFAULT '',
    batch_uuid TEXT
);

-- Step 3: Copy all data except expires_on/expires_at (expires_at will be NULL)
INSERT INTO vouchers (id, code, provider_pkg, session_type, time_secs, data_mb, down_speed_mbps, up_speed_mbps, session_id, device_id, activated_at, created_at, session_exp_days, use_global, uuid, batch_uuid)
SELECT id, code, provider_pkg, session_type, time_secs, data_mb, down_speed_mbps, up_speed_mbps, session_id, device_id, activated_at, created_at, session_exp_days, use_global, uuid, batch_uuid
FROM vouchers_backup;

-- Step 4: Drop backup table
DROP TABLE vouchers_backup;
