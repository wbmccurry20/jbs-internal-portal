# Testing guide

Tests are local-first. No test may use Railway credentials or write to production data.

## Backend

Install the repository-level `xlsx` helper used by the legacy `.xls` conversion tests, then run the suite:

```bash
npm ci
cd backend
go test ./...
```

The default suite requires no production environment variables or database credentials. Database-backed integration tests must use local Docker Postgres and `TEST_DATABASE_URL`; they must skip when it is unset.

```bash
docker compose up -d postgres
cd backend
TEST_DATABASE_URL="postgresql://jbs_user:jbs_password@localhost:5433/jbs_portal?sslmode=disable" go test ./...
```

Never point `TEST_DATABASE_URL` or `DATABASE_URL` at production.

## Frontend

```bash
cd frontend
npm ci
npm test -- --run
```

Coverage percentages are not release gates. Keep tests independent of developer-specific paths, local secrets, and production services.
