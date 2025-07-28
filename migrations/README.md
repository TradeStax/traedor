# Database Setup

## Prerequisites

1. Install PostgreSQL (version 12 or higher)
2. Create a database and user:

```sql
CREATE DATABASE traedor;
CREATE USER traedor WITH PASSWORD 'your_secure_password';
GRANT ALL PRIVILEGES ON DATABASE traedor TO traedor;
```

## Running Migrations

1. Connect to the database:
```bash
psql -U traedor -d traedor
```

2. Run the initial schema:
```bash
psql -U traedor -d traedor < migrations/001_initial_schema.sql
```

## Configuration

Update your `config.yaml` with the database connection string:

```yaml
Database:
  ConnectionString: "postgres://traedor:your_secure_password@localhost:5432/traedor?sslmode=disable"
  MaxConnections: 10
  MaxIdleTime: "30m"
```

## Partitioning

The tick_data table is partitioned by month for better performance. Add new partitions as needed:

```sql
CREATE TABLE IF NOT EXISTS tick_data_2024_01 PARTITION OF tick_data
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');
```