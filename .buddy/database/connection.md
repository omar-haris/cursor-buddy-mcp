# Database Connection
- category: database
- type: postgresql

## Local Development
```
DATABASE_URL=postgresql://user:password@localhost:5432/myapp
```

## Production
Use environment variables:
- `DATABASE_URL` - Full connection string
- `DB_HOST` - Database host
- `DB_PORT` - Database port (default: 5432)
- `DB_NAME` - Database name
- `DB_USER` - Database user
- `DB_PASSWORD` - Database password

## Connection Pool
- Max connections: 10
- Connection timeout: 30s
- Idle timeout: 10m 