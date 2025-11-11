# SQLC Generation Scripts

Shell-based scripts for managing sqlc code generation for both core and plugins, with support for plugin-specific sqlc type overrides.

## Scripts

### `sqlc-gen.sh`
Main script for generating sqlc queries.

**Usage:**
```bash
./scripts/sqlc-gen.sh <plugin_directory> [driver]
```

**Examples:**
```bash
# Generate queries for core with postgres
./scripts/sqlc-gen.sh ./core postgres

# Generate queries for core with sqlite
./scripts/sqlc-gen.sh ./core sqlite

# Generate queries for a plugin
./scripts/sqlc-gen.sh ./data/plugins/local/my-plugin postgres
```

**What it does:**
1. Creates a temporary directory
2. Copies core migrations (if not generating for core itself)
3. Copies plugin migrations and queries
4. Merges core and plugin sqlc configurations (if plugin has custom overrides)
5. Runs `sqlc generate`
6. Copies generated files back to plugin
7. Cleans up temporary directory

### `copy-sql.sh`
Utility script for copying SQL resources (migrations and queries) and merging sqlc configs.

**Usage:**
```bash
./scripts/copy-sql.sh <source_directory> <dest_directory> [driver]
```

**Example:**
```bash
./scripts/copy-sql.sh ./core /tmp/build postgres
```

## Plugin SQLC Overrides

Plugins can define their own sqlc type overrides that will be merged with core overrides.

### How to use:

1. Create `sqlc.postgres.yml` and/or `sqlc.sqlite.yml` in your plugin root directory
2. Define your plugin-specific overrides under `sql[0].gen.go.overrides`
3. Run sqlc generation as usual

### Example plugin sqlc.postgres.yml:

```yaml
---
version: '2'
sql:
  - engine: postgresql
    queries: [resources/queries]
    schema: [resources/migrations]
    gen:
      go:
        package: queries
        out: db/queries
        sql_package: "database/sql"
        overrides:
          # Plugin-specific overrides
          - column: "my_plugin_table.custom_field"
            go_type:
              type: "[]byte"
          - column: "my_plugin_table.rate"
            go_type:
              type: "sql.NullFloat64"
```

See example files:
- `data/plugins/local/com.flarego.basic-wifi-hotspot/sqlc.postgres.yml.example`
- `data/plugins/local/com.flarego.basic-wifi-hotspot/sqlc.sqlite.yml.example`

### Merge behavior:

- Core overrides are always included
- Plugin overrides are appended after core overrides
- If plugin sqlc file is the same as core sqlc file (i.e., plugin is core), merge is skipped
- If a plugin doesn't have sqlc config files, only core overrides are used
- Both postgres and sqlite can have separate plugin overrides

### Implementation:

The merge is done using shell `awk` commands to extract and append the overrides section from the plugin sqlc file to the core sqlc file. This approach:
- ✅ No external dependencies required (no Python, Node.js, or yq needed)
- ✅ Works on all systems with standard POSIX shell
- ✅ Fast and efficient
- ✅ Preserves YAML formatting

## Benefits

1. **Pure shell implementation** - No external dependencies
2. **Plugin customization** - Plugins can define their own type overrides
3. **Backward compatible** - Existing plugins work without changes
4. **Automatic detection** - Automatically detects if plugin is core to avoid double-merging
5. **Database flexibility** - Supports both postgres and sqlite configurations

## How It Works

The `copy-sql.sh` script uses `awk` to:
1. Extract the `overrides:` section from the plugin's sqlc YAML file
2. Find the end of the `overrides:` section in the core's sqlc YAML
3. Append the plugin overrides to the core overrides
4. Write the merged config to the temporary directory

This happens automatically when you run `sqlc-gen.sh` for a plugin that has a custom sqlc configuration file.
