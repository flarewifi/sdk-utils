CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY,
    uid VARCHAR(255) NOT NULL UNIQUE,
    provider_pkg VARCHAR(255) NOT NULL,
    device_id INTEGER NOT NULL,
    session_type VARCHAR(20) NOT NULL,
    time_secs INT DEFAULT 0 NOT NULL,
    data_mbytes DECIMAL(18, 9) NOT NULL DEFAULT 0.0,
    consumption_secs INT NOT NULL DEFAULT 0,
    consumption_mb DECIMAL(18, 9) NOT NULL DEFAULT 0.0,
    exp_days INT DEFAULT NULL,
    down_mbits INT NOT NULL DEFAULT 0,
    up_mbits INT NOT NULL DEFAULT 0,
    use_global BOOLEAN NOT NULL DEFAULT FALSE,
    started_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_sessions_device_id ON sessions(device_id);


CREATE INDEX IF NOT EXISTS index_sessions_device_id ON sessions(device_id);
