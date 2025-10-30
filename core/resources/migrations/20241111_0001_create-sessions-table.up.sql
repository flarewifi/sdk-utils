CREATE TABLE IF NOT EXISTS sessions (
    id INTEGER PRIMARY KEY,
    device_id INTEGER NOT NULL,
    session_type VARCHAR(20) NOT NULL,
    time_secs INT DEFAULT 0 NOT NULL,
    data_mbytes DECIMAL(18, 9) NOT NULL DEFAULT 0.0,
    consumption_secs INT NOT NULL DEFAULT 0,
    consumption_mb DECIMAL(18, 9) NOT NULL DEFAULT 0.0,
    started_at TIMESTAMP,
    exp_days INT DEFAULT NULL,
    down_mbits INT NOT NULL DEFAULT 0,
    up_mbits INT NOT NULL DEFAULT 0,
    use_global BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);
