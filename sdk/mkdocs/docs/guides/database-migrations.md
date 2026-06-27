# Database Migrations and Queries Guide

This comprehensive guide covers database migrations and query development for Flarewifi plugins. Flarewifi uses `sqlc` for type-safe query generation against the plugin database.

## Table of Contents

1. [Overview](#overview)
2. [Creating Migration Files](#creating-migration-files)
3. [Migration File Best Practices](#migration-file-best-practices)
4. [Query Development Guidelines](#query-development-guidelines)
5. [Plugin-Specific Considerations](#plugin-specific-considerations)
6. [Writing Portable SQL](#writing-portable-sql)
7. [sqlc Configuration](#sqlc-configuration)
8. [Code Generation Workflow](#code-generation-workflow)

---

## Overview

### Architecture

- **sqlc**: Code generation tool that turns your SQL into type-safe Go queries
- Plugins own their tables and reference core tables via foreign keys
- Write standard, portable SQL so queries run unchanged across environments

### Directory Structure

```
{plugin-path}/
├── resources/
│   ├── migrations/          # Database schema migrations
│   └── queries/             # SQL queries for sqlc
├── db/
│   └── queries/             # Generated Go code from sqlc
└── sqlc.yml                 # sqlc configuration (optional)
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

**Important:** Use standard, portable SQL in migrations (see [Writing Portable SQL](#writing-portable-sql)).

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
    session_type VARCHAR(10) NOT NULL DEFAULT 'time',
    time_secs INTEGER NOT NULL DEFAULT 0,
    data_mb INTEGER NOT NULL DEFAULT 0,
    down_speed_mbps INTEGER NOT NULL DEFAULT 0,
    up_speed_mbps INTEGER NOT NULL DEFAULT 0,
    device_id INTEGER,                          -- FK to core devices (local, stable id)
    session_uuid VARCHAR(255),                  -- link sessions by uuid, NOT by id
    activated_at TIMESTAMP,                     -- NULL = available, non-NULL = activated
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (device_id) REFERENCES devices (id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_vouchers_code ON my_plugin_vouchers(code);
CREATE INDEX IF NOT EXISTS idx_vouchers_activated_at ON my_plugin_vouchers(activated_at);
```

> **Why `session_uuid`, not `session_id`?** A session may live only in the cloud
> without a stable local integer id, so plugins reference sessions by their
> `uuid` (a plain `VARCHAR` column, **no** foreign key to `sessions.id`). A
> foreign key to a local `devices.id` is fine — client devices are always local.

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

> **Note:** `ALTER TABLE` support can be limited. For complex column changes, you may need to recreate the table.

#### Dropping Indexes

```sql
DROP INDEX IF EXISTS index_name;
```

### Data Types Reference

| Purpose | Recommended Type | Notes |
|---------|------------------|-------|
| Primary Key | `INTEGER PRIMARY KEY` | Auto-incrementing by default |
| Short Text | `VARCHAR(255)` | Use appropriate length |
| Long Text | `TEXT` | For descriptions, content |
| Whole Numbers | `INTEGER` or `INT` | Maps to `int64` in Go |
| Precise Decimals | `DECIMAL(18, 9)` | For prices, measurements |
| Currency | `DECIMAL(8, 2)` | For monetary values |
| Boolean | `BOOLEAN` | `TRUE`/`FALSE` (may be stored as 0/1) |
| Timestamps | `TIMESTAMP` | Use `DEFAULT CURRENT_TIMESTAMP` |
| UUID/Unique ID | `VARCHAR(36)` | Store as string |

### Common Pitfalls to Avoid

1. **Avoid vendor-specific column types and functions:**
   - ❌ `SERIAL` - Use `INTEGER PRIMARY KEY` instead
   - ❌ `UUID` type - Use `VARCHAR(36)` instead
   - ❌ `JSONB` - Use `TEXT` and parse in Go
   - ❌ `ARRAY` types - Use separate tables or JSON in TEXT
   - ❌ `gen_random_uuid()` - Generate UUIDs in Go code
   - ❌ `AUTOINCREMENT` keyword - `INTEGER PRIMARY KEY` auto-increments by default

2. **Always use `IF NOT EXISTS` / `IF EXISTS`:**
   ```sql
   CREATE TABLE IF NOT EXISTS ...
   CREATE INDEX IF NOT EXISTS ...
   DROP TABLE IF EXISTS ...
   DROP INDEX IF EXISTS ...
   ```

3. **Always create paired up/down migrations:**
   - Every `.up.sql` must have a corresponding `.down.sql`

4. **Use consistent naming conventions:**
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
    (code, session_type, time_secs, down_speed_mbps, up_speed_mbps)
VALUES
    (@code, @session_type, @time_secs, @down_speed_mbps, @up_speed_mbps);

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
    session_type = @session_type,
    time_secs = @time_secs,
    data_mb = @data_mb,
    down_speed_mbps = @down_speed_mbps,
    up_speed_mbps = @up_speed_mbps
WHERE id = @id;

-- Mark a voucher activated (NULL activated_at means still available)
-- name: ActivateVoucher :exec
UPDATE my_plugin_vouchers
SET
    device_id = @device_id,
    session_uuid = @session_uuid,
    activated_at = CURRENT_TIMESTAMP
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

### Keep Queries Portable

Prefer standard SQL so a single query works across environments. When you need
date/time math, compute the bounds in Go and pass them as parameters instead of
using database-specific date functions:

```go
cutoff := time.Now().UTC().AddDate(0, 0, -30)
rows, err := q.FindRecent(ctx, cutoff)
```

```sql
-- name: FindRecent :many
SELECT * FROM my_plugin_events WHERE created_at >= @cutoff ORDER BY created_at DESC;
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

!!! danger "Core tables are read-only"
    Plugins may **only read** from core tables (`devices`, `sessions`, `vouchers`,
    `purchases`, `payments`, etc.). **Never** `INSERT`, `UPDATE`, `DELETE`,
    `ALTER`, or `DROP` a core table from a plugin migration or query, and never
    add columns or indexes to one.

    - Access core data through **`SELECT` + `JOIN`** only.
    - Link plugin rows to core rows with **foreign keys** pointing *at* core
      tables — never the reverse.
    - To mutate core data (create a session, activate a voucher, confirm a
      purchase), call the **SDK APIs** (`api.SessionsMgr()`, `api.Vouchers()`,
      `api.Payments()`, …), which enforce invariants and emit events. Writing the
      tables directly bypasses that logic and will corrupt core state.

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
INNER JOIN sessions s ON v.session_uuid = s.uuid
WHERE v.activated_at IS NOT NULL
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

## Writing Portable SQL

Write standard SQL that does not depend on a specific database engine. The
conventions below keep plugin migrations and queries portable.

### Recommended Types

| Purpose | Type | Go Type | Notes |
|---------|------|---------|-------|
| Auto-increment key | `INTEGER PRIMARY KEY` | `int64` | Avoid `SERIAL` |
| Integer | `INTEGER` | `int64` | |
| Short text | `VARCHAR(n)` | `string` | |
| Long text | `TEXT` | `string` | |
| Decimal | `DECIMAL(p,s)` | `float64` | |
| Boolean | `BOOLEAN` | `bool` | May be stored as 0/1 |
| Timestamp | `TIMESTAMP` | `time.Time` | |
| UUID / unique id | `VARCHAR(36)` | `string` | Avoid a native `UUID` type |
| JSON | `TEXT` | `[]byte` | Parse in Go; avoid `JSONB` |

### Timestamps and Date Math

- Use `CURRENT_TIMESTAMP` for default / "now" values in migrations and simple queries.
- Store all timestamps in **UTC**.
- Do **not** use database-specific date functions. Compute time bounds in Go and
  pass them as parameters:

```go
// Compute in Go instead of in SQL:
expiry := startedAt.AddDate(0, 0, expDays)
elapsedSecs := int64(time.Since(startedAt).Seconds())
```

```sql
-- name: FindActiveSessions :many
SELECT * FROM sessions WHERE started_at IS NOT NULL AND @now < @expiry;
```

### Type Casting

Use the standard `CAST(value AS INTEGER)` / `CAST(value AS REAL)` form rather than
engine-specific cast operators.

---

## sqlc Configuration

### Plugin sqlc Configuration

Plugins can define custom type overrides in an `sqlc.yml` in the plugin root. The
most common need is forcing the Go type for specific columns — for example, making
a nullable `SUM()` result a `sql.NullFloat64`:

```yaml
version: '2'
sql:
  - queries: [resources/queries]
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
./scripts/sqlc-gen.sh <plugin_directory>
# e.g.
./scripts/sqlc-gen.sh ./data/plugins/local/my-plugin
```

> **Note:** The script needs core's `sqlc.yml` for foreign-key resolution, so run
> it from the repository root (where the `core/` directory lives). If the plugin
> has no `resources/queries/` directory, generation is **skipped** (the script
> exits successfully without producing any output) — that is expected for plugins
> with no SQL queries.

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
    db, _ := sql.Open(driverName, dataSourceName)
    q := queries.New(db)
    
    // Create
    err := q.CreateVoucher(context.Background(), queries.CreateVoucherParams{
        Code:          "ABC123",
        SessionType:   "time",
        TimeSecs:      3600, // 1 hour
        DownSpeedMbps: 10,
        UpSpeedMbps:   10,
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
        SessionType:   "time",
        TimeSecs:      7200, // 2 hours
        DataMb:        0,
        DownSpeedMbps: 20,
        UpSpeedMbps:   20,
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
| ✅ Write portable SQL (avoid engine-specific syntax) | |
| ✅ Use `COALESCE` for nullable aggregations | |

### Performance

| Do |
|----|
| ✅ Add indexes on columns used in WHERE clauses |
| ✅ Add indexes on foreign key columns |
| ✅ Use pagination for list queries (`LIMIT`/`OFFSET`) |
| ✅ Use appropriate data types to minimize storage |
| ✅ Test queries against the target database |

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
