CREATE TABLE IF NOT EXISTS logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    package VARCHAR(255),
    level VARCHAR(50) NOT NULL,
    message TEXT NOT NULL,
    filepath VARCHAR(512) NOT NULL,
    line_number INTEGER NOT NULL,

    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS index_package ON logs(package);
