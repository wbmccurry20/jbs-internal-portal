# JBS Internal Portal

Internal operations and finance portal for JBS Construction Group. The app centralizes authenticated tools for jobs, bids, superintendent lookup, licensing, training, reimbursement support, and the Concur/reconciliation workflows used by the finance team.

## Stack

- Frontend: Astro 5 + Tailwind CSS
- Backend: Go + Gin
- Database: PostgreSQL
- Local tooling: Docker Compose for Postgres, Node for frontend, Go for backend

## Prerequisites

- Docker Desktop or Docker Engine
- Go 1.24+
- Node.js 20+
- Git

## Local run

1. Start Postgres:

```bash
cd /path/to/jbs-internal-portal
docker compose up -d postgres
```

2. Create the backend local env if it does not exist:

```bash
cp backend/.env.example backend/.env
```

3. Install the root XLS helper used by reconciliation and its tests:

```bash
npm ci
```

4. Start the backend:

```bash
cd backend
go run ./cmd/server
```

5. Start the frontend in a second terminal:

```bash
cd frontend
npm install
npm run dev
```

The app is available at:
- Frontend: http://localhost:4321
- Backend API: http://localhost:8080

## Local-only login note

For local development, the bootstrap admin is a seed account only and should not be used in production. Example local-only access:

- Email: admin@jbs.com
- Password: password123

Do not document or reuse production credentials in this repo.

## Primary pages

- Login
- Dashboard
- Jobs
- Bids
- Superintendents
- Licensing
- Training
- Users
- Concur
- Reconciliation
- Invite flow
- Change password

## API at a glance

The app exposes a small set of authenticated and public API groups:

- Auth: login, logout-style session validation, current user, change-password
- Concur: upload and history for expense conversion jobs
- Health: backend health checks
- Additional operational routes exist for jobs, bids, users, training, licensing, and support tooling, but the core local-dev flow is centered on auth + Concur + health

## Tests

See [docs/TESTING.md](docs/TESTING.md) for the supported local workflow. The short version is:

```bash
npm ci
cd backend && go test ./...
cd ../frontend && npm ci && npm test -- --run
```

## Deployment

Use the production guidance in [DEPLOYMENT.md](DEPLOYMENT.md). Keep Railway secrets in the deployment platform, not in git.

## License

Internal JBS use only.
