# Database Migrations and Queries Guide

This comprehensive guide covers database migrations and query development for Flarewifi plugins. Flarewifi supports both PostgreSQL and SQLite databases using `sqlc` for type-safe query generation.

## Table of Contents

1. [Overview](#overview)
2. [Creating Migration Files](#creating-migration-files)
3. [Migration File Best Practices](#migration-file-best-practices)
4. [Query Development Guidelines](#query-development-guidelines)
5. [Plugin-Specific Considerations](#plugin-specific-considerations)
6. [SQL Compatibility Between PostgreSQL and SQLite](#sql-compatibility-between-postgresql-and-sqlite)
7. [sqlc Configuration](#sqlc-configuration)
8. [Code Generation Workflow](#code-generation-workflow)

---

## Overview

### Architecture

Flarewifi uses a dual-database architecture:

- **PostgreSQL**: Production environment with extensive type overrides
- **SQLite**: Development/OpenWRT environment with minimal overrides
- **sqlc**: Code generation tool for type-safe database queries

### Directory Structure

```
{plugin-path}/
├── resources/
│   ├── migrations/          # Database schema migrations
│   └── queries/             # SQL queries for sqlc
│       ├── sqlite/          # SQLite-specific queries (optional)
│       └── postgres/        # PostgreSQL-specific queries (optional)
├── db/
│   └── queries/             # Generated Go code from sqlc
├── sqlc.postgres.yml        # PostgreSQL sqlc configuration (optional)
└── sqlc.sqlite.yml          # SQLite sqlc configuration (optional)
```

---

## Creating Migration Files

To create migration files for your database, you can use the `create-migration` command:

**Windows:**
```powershell
.\scripts\flare.bat create-migration
```

**Linux/Mac:**
```bash
./scripts/flare.sh create-migration
```

This will create new migration files in the `resources/migrations` directory of your plugin. The files will be named with a timestamp and a description of the migration.

**Important:** SQL commands must be compatible with **both PostgreSQL and SQLite** databases.

---

## Migration File Best Practices

### Naming Convention

Migration files follow a strict naming pattern:

```
YYYYMMDD_NNNN_description.{up,down}.sql
```

**Components:**

- `YYYYMMDD`: Date in year-month-day format (e.g., `20241111`)
- `NNNN`: Sequential number within the same date (e.g., `0001`, `0002`)
- `description`: Brief description using hyphens or underscores
- `.up.sql`: Forward migration (creates/modifies schema)
- `.down.sql`: Rollback migration (reverts changes)

**Examples:**
```
20241111_0000_create-devices-table.up.sql
20241111_0000_create-devices-table.down.sql
20251121_0010_add_webhook_route_to_purchases.up.sql
20251121_0010_add_webhook_route_to_purchases.down.sql
```

> **Note:** Plugin migrations can use longer timestamps (with microseconds) for uniqueness.

### Up Migration Structure

#### Creating Tables

```sql
CREATE TABLE IF NOT EXISTS {plugin_prefix}_table_name (
    id INTEGER PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    quantity INTEGER DEFAULT 0 NOT NULL,
    price DECIMAL(18, 9) NOT NULL DEFAULT 0.0,
    is_active BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (related_id) REFERENCES related_table (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS index_table_name_column ON {plugin_prefix}_table_name(column);
CREATE UNIQUE INDEX IF NOT EXISTS index_table_name_unique_field ON {plugin_prefix}_table_name(unique_field);
```

**Real example from a plugin:**
```sql
CREATE TABLE IF NOT EXISTS my_plugin_vouchers (
    id INTEGER PRIMARY KEY,
    code VARCHAR(10) NOT NULL DEFAULT '',
    validity_count INT NOT NULL DEFAULT 0,
    validity_unit INT NOT NULL DEFAULT 0,
    speed INT NOT NULL DEFAULT 0,
    status INT NOT NULL DEFAULT 1,
    device_ip VARCHAR(17) NOT NULL DEFAULT 'N/A',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_vouchers_code ON my_plugin_vouchers(code);
CREATE INDEX IF NOT EXISTS idx_vouchers_status ON my_plugin_vouchers(status);
```

#### Adding Columns

```sql
ALTER TABLE table_name ADD COLUMN new_column VARCHAR(255) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS index_new_column ON table_name(new_column);
```

#### Renaming Columns

```sql
ALTER TABLE table_name RENAME COLUMN old_name TO new_name;
```

### Down Migration Structure

#### Dropping Tables

```sql
DROP TABLE IF EXISTS {plugin_prefix}_table_name;
```

#### Removing Columns

```sql
ALTER TABLE table_name DROP COLUMN IF EXISTS column_name;
```

> **Note:** SQLite has limited `ALTER TABLE` support. For complex column changes, you may need to recreate the table.

#### Dropping Indexes

```sql
DROP INDEX IF EXISTS index_name;
```

### Data Types Reference

| Purpose | Recommended Type | Notes |
|---------|------------------|-------|
| Primary Key | `INTEGER PRIMARY KEY` | Auto-incrementing in both databases |
| Short Text | `VARCHAR(255)` | Use appropriate length |
| Long Text | `TEXT` | For descriptions, content |
| Whole Numbers | `INTEGER` or `INT` | Maps to `int64` in Go |
| Precise Decimals | `DECIMAL(18, 9)` | For prices, measurements |
| Currency | `DECIMAL(8, 2)` | For monetary values |
| Boolean | `BOOLEAN` | `TRUE`/`FALSE` (SQLite stores as 0/1) |
| Timestamps | `TIMESTAMP` | Use `DEFAULT CURRENT_TIMESTAMP` |
| UUID/Unique ID | `VARCHAR(36)` | Store as string |

### Common Pitfalls to Avoid

1. **Don't use PostgreSQL-only syntax:**
   - ❌ `SERIAL` - Use `INTEGER PRIMARY KEY` instead
   - ❌ `UUID` type - Use `VARCHAR(36)` instead
   - ❌ `JSONB` - Use `TEXT` and parse in Go
   - ❌ `ARRAY` types - Use separate tables or JSON in TEXT
   - ❌ `gen_random_uuid()` - Generate UUIDs in Go code

2. **Don't use SQLite-only syntax:**
   - ❌ `AUTOINCREMENT` keyword - `INTEGER PRIMARY KEY` auto-increments by default

3. **Always use `IF NOT EXISTS` / `IF EXISTS`:**
   ```sql
   CREATE TABLE IF NOT EXISTS ...
   CREATE INDEX IF NOT EXISTS ...
   DROP TABLE IF EXISTS ...
   DROP INDEX IF EXISTS ...
   ```

4. **Always create paired up/down migrations:**
   - Every `.up.sql` must have a corresponding `.down.sql`

5. **Use consistent naming conventions:**
   - Table names: `{plugin_prefix}_table_name` (snake_case)
   - Index names: `idx_{table}_{column}` or `index_{table}_{column}`

### Running Migration Files

- The `up` migration files are run automatically during plugin installation
- The `down` migrations are run automatically when the plugin is being uninstalled

---

## Query Development Guidelines

### sqlc Query Structure

Every query must have a name comment and return type:

```sql
-- name: QueryName :return_type
SELECT/INSERT/UPDATE/DELETE ...
```

### Return Types

| Type | Description | Use Case |
|------|-------------|----------|
| `:one` | Single row result | Find by ID, find by unique field |
| `:many` | Multiple rows result | List queries, search queries |
| `:exec` | No return value | UPDATE, DELETE without returning |

### Named Parameters

**Always use `@parameter_name` syntax:**

```sql
-- name: FindUserByEmail :one
SELECT * FROM users WHERE email = @email LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (name, email) VALUES (@name, @email) RETURNING id;

-- name: UpdateUser :exec
UPDATE users SET name = @name WHERE id = @id;
```

### CRUD Operation Patterns

#### Create (INSERT)

```sql
-- name: CreateVoucher :exec
INSERT INTO my_plugin_vouchers
    (code, validity_count, validity_unit, speed)
VALUES
    (@code, @validity_count, @validity_unit, @speed);

-- With RETURNING (to get the ID)
-- name: CreatePaymentTrace :one
INSERT INTO my_plugin_payment_traces (
    trace_number, device_id, device_mac, amount, status
) VALUES (
    @trace_number, @device_id, @device_mac, @amount, @status
) RETURNING *;
```

#### Read (SELECT)

```sql
-- Single row
-- name: FindVoucherByID :one
SELECT * FROM my_plugin_vouchers WHERE id = @id LIMIT 1;

-- Multiple rows with pagination
-- name: GetAllVouchers :many
SELECT * FROM my_plugin_vouchers
ORDER BY created_at DESC 
LIMIT @row_limit OFFSET @row_offset;

-- Count query
-- name: GetAllVouchersCount :one
SELECT COUNT(id) FROM my_plugin_vouchers;
```

#### Update

```sql
-- name: UpdateVoucher :exec
UPDATE my_plugin_vouchers
SET
    status = @status,
    device_ip = @device_ip,
    validity_count = @validity_count,
    validity_unit = @validity_unit,
    speed = @speed
WHERE id = @id;

-- With timestamp update
-- name: UpdatePaymentTraceStatus :exec
UPDATE my_plugin_payment_traces
SET status = @status, updated_at = CURRENT_TIMESTAMP
WHERE trace_number = @trace_number;
```

#### Delete

```sql
-- name: DeleteVoucherByID :exec
DELETE FROM my_plugin_vouchers WHERE id = @id;

-- Conditional delete
-- name: DeleteOldPaymentTraces :exec
DELETE FROM my_plugin_payment_traces
WHERE created_at < @before
AND status IN ('paid', 'failed', 'refunded');
```

### Complex Query Patterns

#### Joins with Core Tables

```sql
-- name: FindSales :many
SELECT
    p.id as purchase_id,
    COALESCE(SUM(py.amount), 0) AS amount,
    p.confirmed_at AS date,
    d.mac_address,
    d.id AS device_id,
    s.id AS session_id,
    s.time_secs,
    s.data_mbytes
FROM
    purchases p
JOIN
    devices d ON p.device_id = d.id
LEFT JOIN
    payments py ON py.purchase_id = p.id
LEFT JOIN
    my_plugin_sessions_purchases sp ON sp.purchase_id = p.id
LEFT JOIN
    sessions s ON sp.session_id = s.id
WHERE
    (@mac_address = '' OR LOWER(REPLACE(d.mac_address, ':', '')) LIKE '%' || LOWER(@mac_address) || '%')
    AND (p.confirmed_at BETWEEN @start_date AND @end_date)
GROUP BY p.id, d.id, s.id
ORDER BY p.confirmed_at DESC
LIMIT @row_limit OFFSET @row_offset;
```

#### Conditional Filtering

```sql
-- name: SearchLogs :many
SELECT * FROM logs
WHERE (@package = '' OR package = @package)
AND (@level = '' OR level = @level)
AND (@search_text = '' OR LOWER(message) LIKE '%' || LOWER(@search_text) || '%')
ORDER BY created_at DESC
LIMIT @row_limit OFFSET @row_offset;
```

#### Upsert (INSERT ... ON CONFLICT)

```sql
-- name: UpsertQuickAccessNav :exec
INSERT INTO quick_access_navs (
    plugin_pkg, route_name, route_params, visit_count, updated_at
)
VALUES (@plugin_pkg, @route_name, @route_params, 1, CURRENT_TIMESTAMP)
ON CONFLICT(plugin_pkg, route_name, route_params)
DO UPDATE SET
    visit_count = quick_access_navs.visit_count + 1,
    updated_at = CURRENT_TIMESTAMP;
```

### Database-Specific Queries

When queries require database-specific syntax, create separate files:

```
resources/queries/
├── common-queries.sql      # Works on both databases
├── sqlite/
│   └── time-queries.sql    # SQLite-specific
└── postgres/
    └── time-queries.sql    # PostgreSQL-specific
```

**Add engine comment to database-specific queries:**

```sql
-- name: FindAvailableSession :one
-- engine: sqlite
SELECT * FROM sessions WHERE ...
```

```sql
-- name: FindAvailableSession :one
-- engine: postgresql
SELECT * FROM sessions WHERE ...
```

---

## Plugin-Specific Considerations

### Critical Rules

1. **NEVER modify core migrations** - Plugins cannot alter `core/resources/migrations/`
2. **NEVER modify core tables** - Use foreign keys and JOINs instead
3. **Use plugin-specific table prefixes** - Prevents naming conflicts
4. **Each plugin has its own migrations directory**

### Table Naming Convention

Use a unique prefix for all plugin tables:

```
{vendor}_{plugin_name}_{table_name}
```

**Examples:**
```
my_company_hotspot_vouchers
my_company_hotspot_pause_counts
my_company_payment_traces
```

### Referencing Core Tables

#### Using Foreign Keys

```sql
CREATE TABLE IF NOT EXISTS my_plugin_free_trial_usage (
    id INTEGER PRIMARY KEY,
    device_id INTEGER NOT NULL,
    used_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE CASCADE
);
```

#### Using JOINs in Queries

```sql
-- name: GetConsumedVouchers :many
SELECT v.*
FROM my_plugin_vouchers v
INNER JOIN sessions s ON v.session_id = s.id
WHERE v.status = @status
AND s.time_secs - s.consumption_secs <= 0
ORDER BY v.created_at DESC 
LIMIT @row_limit OFFSET @row_offset;
```

### Junction Tables for Many-to-Many Relationships

```sql
CREATE TABLE IF NOT EXISTS my_plugin_sessions_purchases (
    id INTEGER PRIMARY KEY,
    purchase_id INTEGER NOT NULL,
    session_id INTEGER NOT NULL,

    FOREIGN KEY (purchase_id) REFERENCES purchases (id) ON DELETE CASCADE,
    FOREIGN KEY (session_id) REFERENCES sessions (id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sessions_purchases_purchase_id ON my_plugin_sessions_purchases(purchase_id);
CREATE INDEX IF NOT EXISTS idx_sessions_purchases_session_id ON my_plugin_sessions_purchases(session_id);
```

---

## SQL Compatibility Between PostgreSQL and SQLite

### Data Type Mapping

| PostgreSQL | SQLite | Go Type | Notes |
|------------|--------|---------|-------|
| `INTEGER` | `INTEGER` | `int64` | Use for all integers |
| `SERIAL` | `INTEGER PRIMARY KEY` | `int64` | Auto-increment |
| `VARCHAR(n)` | `VARCHAR(n)` | `string` | Compatible |
| `TEXT` | `TEXT` | `string` | Compatible |
| `DECIMAL(p,s)` | `DECIMAL(p,s)` | `float64` | Compatible |
| `BOOLEAN` | `BOOLEAN` | `bool` | SQLite stores as 0/1 |
| `TIMESTAMP` | `TIMESTAMP` | `time.Time` | Compatible |
| `UUID` | `VARCHAR(36)` | `string` | Use VARCHAR for compatibility |
| `JSONB` | `TEXT` | `[]byte` | Parse JSON in Go |

### Function Differences

#### Current Timestamp

| PostgreSQL | SQLite |
|------------|--------|
| `NOW()` | `datetime('now')` |
| `CURRENT_TIMESTAMP` | `CURRENT_TIMESTAMP` |

**Use `CURRENT_TIMESTAMP` for compatibility in migrations and simple queries.**

#### Date/Time Arithmetic

**SQLite:**
```sql
datetime(started_at, '+' || exp_days || ' days')
datetime('now') < datetime(started_at, '+' || exp_days || ' days')
CAST((julianday('now') - julianday(started_at)) * 86400 AS INTEGER)
```

**PostgreSQL:**
```sql
started_at + (exp_days * interval '1 day')
NOW() < started_at + (exp_days * interval '1 day')
EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
```

#### Type Casting

**SQLite:**
```sql
CAST(value AS INTEGER)
CAST(value AS REAL)
```

**PostgreSQL:**
```sql
value::INTEGER
value::REAL
```

### Complete Example: Database-Specific Queries

**SQLite (`resources/queries/sqlite/sessions-sqlite.sql`):**
```sql
-- name: FindAvailableSessionForDevice :one
-- engine: sqlite
SELECT * FROM sessions
WHERE device_id = @device_id
AND (
    (session_type = 'time' AND consumption_secs < time_secs)
    OR (session_type = 'data' AND consumption_mb < data_mbytes)
    OR (session_type = 'time-or-data' AND consumption_mb < data_mbytes AND consumption_secs < time_secs)
)
AND (
    (exp_days IS NULL OR started_at IS NULL)
    OR (
        exp_days IS NOT NULL
        AND started_at IS NOT NULL
        AND datetime('now') < datetime(started_at, '+' || exp_days || ' days')
    )
)
LIMIT 1;

-- name: BulkUpdateTimeConsumption :exec
-- engine: sqlite
UPDATE sessions
SET consumption_secs = consumption_secs + CAST((julianday('now') - julianday(started_at)) * 86400 AS INTEGER)
WHERE started_at IS NOT NULL;
```

**PostgreSQL (`resources/queries/postgres/sessions-pg.sql`):**
```sql
-- name: FindAvailableSessionForDevice :one
-- engine: postgresql
SELECT * FROM sessions
WHERE device_id = @device_id
AND (
    (session_type = 'time' AND consumption_secs < time_secs)
    OR (session_type = 'data' AND consumption_mb < data_mbytes)
    OR (session_type = 'time-or-data' AND consumption_mb < data_mbytes AND consumption_secs < time_secs)
)
AND (
    (exp_days IS NULL OR started_at IS NULL)
    OR (
        exp_days IS NOT NULL
        AND started_at IS NOT NULL
        AND NOW() < started_at + (exp_days * interval '1 day')
    )
)
LIMIT 1;

-- name: BulkUpdateTimeConsumption :exec
-- engine: postgresql
UPDATE sessions
SET consumption_secs = consumption_secs + EXTRACT(EPOCH FROM (NOW() - started_at))::INTEGER
WHERE started_at IS NOT NULL;
```

---

## sqlc Configuration

### Plugin sqlc Configuration Files

Plugins can define custom type overrides by creating `sqlc.postgres.yml` and/or `sqlc.sqlite.yml` in the plugin root.

**Example `sqlc.postgres.yml`:**
```yaml
version: '2'
sql:
  - engine: postgresql
    queries: [resources/queries]
    schema: [resources/migrations]
    gen:
      go:
        package: queries
        out: db/queries
        sql_package: database/sql
        overrides:
          # Override SUM result to handle NULL
          - column: findsales.amount
            go_type:
              import: database/sql
              type: NullFloat64
```

**Example `sqlc.sqlite.yml`:**
```yaml
version: '2'
sql:
  - engine: sqlite
    queries: [resources/queries]
    schema: [resources/migrations]
    gen:
      go:
        package: queries
        out: db/queries
        sql_package: database/sql
        overrides:
          - column: findsales.amount
            go_type: float64
```

### Common Type Overrides

```yaml
overrides:
  # Force integers to int64
  - db_type: "integer"
    go_type: "int64"
  
  # Force decimals to float64
  - db_type: "decimal"
    go_type: "float64"
  
  # Nullable integers
  - db_type: "integer"
    go_type:
      import: "database/sql"
      type: "NullInt64"
    nullable: true
  
  # JSON/metadata columns as []byte
  - column: "table_name.metadata"
    go_type:
      type: "[]byte"
  
  # Nullable timestamps
  - column: "table_name.confirmed_at"
    go_type:
      type: "sql.NullTime"
```

---

## Code Generation Workflow

### Running sqlc Generation

```bash
# Generate for PostgreSQL
./scripts/sqlc-gen.sh ./data/plugins/local/my-plugin postgres

# Generate for SQLite
./scripts/sqlc-gen.sh ./data/plugins/local/my-plugin sqlite
```

### What the Script Does

1. Creates a temporary directory
2. Copies core migrations (for foreign key references)
3. Copies plugin migrations and queries
4. Merges core and plugin sqlc configurations
5. Runs `sqlc generate`
6. Copies generated files to `{plugin}/db/queries/`
7. Cleans up temporary directory

### Generated Code Usage

```go
package main

import (
    "context"
    "database/sql"
    "your-plugin/db/queries"
)

func main() {
    db, _ := sql.Open("sqlite3", "database.db")
    q := queries.New(db)
    
    // Create
    err := q.CreateVoucher(context.Background(), queries.CreateVoucherParams{
        Code:          "ABC123",
        ValidityCount: 30,
        ValidityUnit:  1,
        Speed:         10,
    })
    
    // Read
    voucher, err := q.FindVoucherByID(context.Background(), 1)
    
    // List with pagination
    vouchers, err := q.GetAllVouchers(context.Background(), queries.GetAllVouchersParams{
        RowLimit:  10,
        RowOffset: 0,
    })
    
    // Update
    err = q.UpdateVoucher(context.Background(), queries.UpdateVoucherParams{
        ID:            1,
        Status:        2,
        DeviceIp:      "192.168.1.100",
        ValidityCount: 60,
        ValidityUnit:  1,
        Speed:         20,
    })
    
    // Delete
    err = q.DeleteVoucherByID(context.Background(), 1)
}
```

---

## Best Practices Summary

### Migrations

| Do | Don't |
|----|-------|
| ✅ Always create paired up/down migration files | ❌ Never modify core migrations |
| ✅ Use `IF NOT EXISTS` / `IF EXISTS` for idempotency | ❌ Never use database-specific syntax in migrations |
| ✅ Use plugin-specific table prefixes | ❌ Never use `SERIAL`, `UUID` type, `JSONB`, or `ARRAY` |
| ✅ Add indexes on foreign keys and frequently queried columns | ❌ Never use `gen_random_uuid()` |
| ✅ Use `ON DELETE CASCADE` for foreign keys | |

### Queries

| Do | Don't |
|----|-------|
| ✅ Use named parameters (`@param_name`) | ❌ Never hardcode values that should be parameters |
| ✅ Use `LIMIT 1` for `:one` queries | |
| ✅ Include `RETURNING id` for INSERT with `:one` | |
| ✅ Create database-specific files only when necessary | |
| ✅ Use `COALESCE` for nullable aggregations | |

### Performance

| Do |
|----|
| ✅ Add indexes on columns used in WHERE clauses |
| ✅ Add indexes on foreign key columns |
| ✅ Use pagination for list queries (`LIMIT`/`OFFSET`) |
| ✅ Use appropriate data types to minimize storage |
| ✅ Test queries with both PostgreSQL and SQLite |

### Plugin Isolation

| Do | Don't |
|----|-------|
| ✅ Use unique table prefixes for all plugin tables | ❌ Never alter core tables from plugins |
| ✅ Reference core tables via foreign keys only | |
| ✅ Use JOINs to access core table data | |
| ✅ Keep plugin migrations in plugin directory | |

---

## Related

- [ISessionsMgrApi](../api/sessions-mgr-api.md) — Core sessions table schema reference (foreign key target for plugin tables)
- [IClientDevice](../api/client-device.md) — Core devices table schema reference
- [IVouchersApi](../api/voucher-api.md) — Core vouchers table schema reference
- [Saving Data](./saving-data.md) — For key-value plugin config that does not require a migration
