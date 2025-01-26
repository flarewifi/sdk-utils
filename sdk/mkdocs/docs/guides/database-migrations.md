# Database Migrations

Database migrations are a way to manage changes to your database schema over time. They are a way to keep track of changes to your database schema and apply them in a consistent way. This is important because as your application evolves, the database schema will need to change to reflect that.

## 1. Creating Migration Files {#creating-migration-files}

To create migration files for your database, you can use the `db-migrate` command:

In Windows:

```title="PowerShell"
.\scripts\flare.bat create-migration
```

In Linux/Mac:

```title="Terminal"
./scripts/flare.sh create-migration
```

This will create new migration files in the `resources/migrations` directory of your plugin. The file will be named with a timestamp and a description of the migration. For example, `20210101000000_create_users_table.up.sql` and `20210101000000_create_users_table.down.sql`.

The file with the `.up.sql` extension contains the SQL commands to apply the migration, and the file with the `.down.sql` extension contains the SQL commands to revert the migration.

SQL commands must be written for MySQL database since we are using MySQL as the database for Flare Hotspot.

Below is an example of an **up** migration file:

```sql title="resources/migrations/20210101000000_create_users_table.up.sql"
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    email VARCHAR(255) NOT NULL,
    password VARCHAR(20) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

Below is an example of the **down** migration file:

```sql title="resources/migrations/20210101000000_create_users_table.down.sql"
DROP TABLE IF EXISTS users;
```

## 2. Running Migration Files {#running-migration-files}

The `up` migration files are run automatically during plugin installation.
The `down` migrations are also run automatically when the plugin is being uninstalled.
