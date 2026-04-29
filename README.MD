# Start Postgresql and redis
- brew services start postgresql@16
- brew services start redis

# Connect to PostgreSQL (Homebrew creates a default user matching your macOS username)
psql postgres

# Inside the psql shell, run these SQL commands:
CREATE USER edusaas WITH PASSWORD 'edusaas_dev';
CREATE DATABASE edusaas OWNER edusaas;
GRANT ALL PRIVILEGES ON DATABASE edusaas TO edusaas;

-- Required for uuid-ossp extension
\c edusaas
GRANT CREATE ON SCHEMA public TO edusaas;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

\q

# Run the migration
cd edu-saas
psql -U edusaas -d edusaas -f backend/migrations/001_foundation.sql

# If it asks for a password, enter edusaas_dev.
psql -h localhost -U edusaas -d edusaas -f backend/migrations/001_foundation.sql

# Verify the tables were created:
psql -h localhost -U edusaas -d edusaas -c "\dt"

# Set up and run the Go backend
cd backend

# Download dependencies
go mod tidy

# Create your env file (optional — defaults work for local dev)
cp ../.env.example .env

# Now run the server:
go run ./cmd/server

# This drops everything and recreates — you'll lose existing test data
psql -h localhost -U edusaas -d edusaas -c "
  DROP SCHEMA public CASCADE;
  CREATE SCHEMA public;
  CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";
  CREATE EXTENSION IF NOT EXISTS \"pgcrypto\";
"