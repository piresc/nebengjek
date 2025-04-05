# Database Migrations

This directory contains database migration files for the User service. Each migration file is versioned with a sequential number prefix (e.g., `001_`, `002_`) to ensure they are applied in the correct order.

## Migration Files

- `001_initial_schema.sql`: Initial database schema for the User service

## Best Practices

1. **Never modify existing migration files** that have been applied to any environment
2. **Always create new migration files** for schema changes
3. Use descriptive names for migration files (e.g., `002_add_user_preferences.sql`)
4. Include both `up` (apply) and `down` (rollback) migrations when possible
5. Test migrations thoroughly in development before applying to production

## Running Migrations

Migrations should be applied using a migration tool such as:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- [atlas](https://atlasgo.io/)

Example command using golang-migrate:
```
migrate -path ./migrations -database "postgresql://username:password@localhost:5432/database?sslmode=disable" up
```