-- Logs are no longer stored in the database. Application logs now go to stdout
-- (syslog/logread) and a rotating file, viewed via the admin log viewer. See
-- core/internal/modules/logger and core/internal/api/logger-api.go.
DROP INDEX IF EXISTS index_package;
DROP INDEX IF EXISTS index_level;
DROP TABLE IF EXISTS logs;
