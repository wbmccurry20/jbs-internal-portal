# Architecture Decision Record: Payment Application Backend

**Status:** Accepted  
**Date:** 2026-05-29  
**Ticket:** Architecture Check — Should JBS Site Use Internal Portal Backend for Payment Applications?  
**Decider:** Engineering  
**Affects:** `jbs-site` (public frontend), `jbs-internal-portal` (backend + admin portal)

---

## Context

The JBS payment application generator is a public-facing tool on `buildwithjbs.com` that lets
subcontractors submit AIA G702/G703-style payment applications online. The form prototype was
built in `jbs-site` (Astro, Vercel). The backend schema and Go models were started in
`jbs-internal-portal` (Go/Gin, PostgreSQL, Railway).

These are **separate deployed projects.** Before wiring them together, this document validates
whether `jbs-internal-portal` is the correct backend home, or whether a different approach
should be used.

---

## Project Inventory

### `jbs-site` — Public marketing and client portal website

| Property | Value |
|---|---|
| Framework | Astro 5, React 19, Tailwind CSS v4 |
| Deployment | Vercel (static build) |
| Domain | buildwithjbs.com |
| SSR adapter | **None** — purely static output |
| Database clients | None — no database dependencies in `package.json` |
| API routes | None — no Vercel functions, no `src/pages/api/` directory |
| Auth | None |
| Stripe | None |

**Key finding:** `jbs-site` is a **100% static site**. It has no server runtime. There is no
mechanism in the current project to run server-side code, connect to a database, or handle
webhooks.

---

### `jbs-internal-portal` — Go/Gin backend + internal React portal

| Property | Value |
|---|---|
| Backend | Go 1.24, Gin |
| Deployment | Railway — backend at `api.buildwithjbs.com` (port 8080) |
| Database | PostgreSQL (Railway-managed) |
| Auth | JWT (`golang-jwt/jwt/v5`), RBAC (executive, hr_admin, finance, project_manager, construction_admin, support) |
| CORS | `ALLOWED_ORIGINS` env var — currently set to internal portal frontend origin |
| Public routes (no JWT) | `/health`, `/api/login`, `/api/invite/validate/:token`, `/api/invite/accept`, `/api/sharepoint/callback` |
| All other routes | Protected by `AuthMiddleware()` |
| Migration infrastructure | Yes — `internal/database/migrate.go`, `migrations/` directory |
| Payment application schema | Migrations 020–023 created (not yet applied to production) |
| Payment application Go models | `internal/models/payment_application.go` created |

**Key finding:** The backend infrastructure is mature and ready. It already has a database, a
migration runner, auth, rate limiting, CORS handling, and a deployed Railway service. The
payment application schema work is already under way here.

---

## Options Evaluated

### Option 1 — Use `jbs-internal-portal` Go/PostgreSQL backend (extend existing service)

**How it works:**
- Add public routes to `main.go` — outside the `api.Use(middleware.AuthMiddleware())` block:
  - `POST /api/v1/payment-applications` — public submission (no JWT)
  - `GET /api/v1/payment-applications/:token` — public status lookup by submission token (no JWT)
- Add protected admin routes inside the JWT block:
  - `GET /api/v1/admin/payment-applications` — admin list (executive, finance, support)
  - `GET /api/v1/admin/payment-applications/:id` — admin detail
  - `PATCH /api/v1/admin/payment-applications/:id/review` — mark reviewed
- Add `https://buildwithjbs.com` to `ALLOWED_ORIGINS` in Railway environment variables
- Apply migrations 020–023 to Railway PostgreSQL
- `jbs-site` React form POSTs JSON to `https://api.buildwithjbs.com/api/v1/payment-applications`

**Database/storage ownership:** One Railway PostgreSQL instance. Payment applications live in
the same DB as jobs, users, licenses, training. No data silo. Cross-referencing (e.g., linking
a PA to a job record) is possible in a future ticket without a join-across-service problem.

**Stripe webhook handling:** Add `POST /api/v1/webhooks/stripe` as a public route in Go. Gin
handles the raw body needed for HMAC signature verification. Stripe webhooks require a stable
HTTPS endpoint — Railway provides this out of the box.

**PDF generation:** Go service generates PDF in-process (using a Go PDF library such as
`github.com/jung-kurt/gofpdf` or `github.com/signintech/gopdf`). PDF stored in object storage
(S3/Cloudflare R2) or as a Railway volume, referenced by `pdf_storage_key` in the schema.

**AP email delivery:** Go backend calls Resend (or SendGrid) HTTP API after submission.
`ap_email` is already stored per tenant in `pa_tenants`. The `email_status` and
`ap_email_status` columns track delivery state.

**Admin review workflow:** The existing internal portal frontend (Railway) gets a new
"Payment Applications" page. JBS staff log in with existing credentials and review submissions
in the same tool they already use for jobs, licenses, and training. No new admin tool to build.

**CORS/security implications:**
- `buildwithjbs.com` is added to `ALLOWED_ORIGINS`. This is low risk — CORS is origin-scoped,
  not credential-scoped. Public endpoints require no JWT, so there are no internal credentials
  at risk.
- Public submission endpoint must be rate-limited. The existing `middleware.RateLimitGeneral()`
  already applies at the router level. A more specific `RateLimitSubmission()` middleware can
  be added for the submission route (e.g., 5 requests/IP/hour).
- `submission_token` is a cryptographically random UUID — no sequential ID guessing attack is
  possible on the status lookup route.
- `IPAddress` and `UserAgent` are stored in the DB for abuse tracking but are `json:"-"` — they
  are never returned to callers.
- Public endpoints expose only public-facing fields (no internal user IDs, no AP email address
  in the submission response, no internal review notes).

**Deployment implications:** No new services. One Railway env var change (`ALLOWED_ORIGINS`).
Database migrations are applied on the existing Railway PostgreSQL.

**White-label portability:** The `pa_tenants` table (migration 020) is the white-label
abstraction. Each GC has a slug, AP email, and brand config. The public submission endpoint
accepts a `tenant` slug in the URL (e.g., `POST /api/v1/pa/jbs/payment-applications`). A new
GC is a new row — no new service deployment required.

**Schema/model work already done:**
- Migrations 020–023: correct, complete, ready to apply
- `internal/models/payment_application.go`: correct, passes `go build ./...` and `go vet ./...`
- **This work should continue. Nothing needs to be paused or moved.**

**Complexity/risk:**
- Low. The backend already exists and is in production. The pattern (public routes + protected
  routes in the same Gin server) is standard and already in use (`/api/invite/validate/:token`
  is already a public route with no JWT).
- The only new runtime dependency is a PDF library (deferred to the PDF ticket).
- CORS change is one env var update.

---

### Option 2 — Vercel/serverless API routes inside `jbs-site`

**How it works:**
- Add `@astrojs/vercel` adapter and switch `jbs-site` from static to SSR mode
- Create `src/pages/api/*.ts` Astro API routes (or Vercel Edge/Serverless functions)
- Connect to a separate database (Neon, Supabase, or Vercel Postgres) — or tunnel to Railway
  PostgreSQL (cross-cloud latency, credential exposure risk)

**Assessment:**

| Concern | Problem |
|---|---|
| Static-to-SSR migration | `jbs-site` has no SSR adapter. Adding `@astrojs/vercel` is non-trivial: all pages must be audited for compatibility; build pipeline changes; `client:load` islands behave differently under SSR. High risk of regressions on a production public site. |
| Runtime | Node.js/TypeScript, not Go. The existing backend team knows Go. TypeScript serverless functions are a different runtime with different operational concerns (cold starts, 10s execution limits on Vercel Hobby plan). |
| Database | Requires a second PostgreSQL instance (Neon/Vercel Postgres) or cross-cloud access to Railway — the latter exposes Railway DB credentials in Vercel env vars. If a second DB, admin review data is isolated from the internal portal. |
| Stripe webhooks | Vercel serverless functions can handle webhooks but require careful raw body handling in Next.js/Astro API routes. Not impossible, but less clean than a persistent Go server. |
| PDF generation | Node.js serverless PDFs: `pdfkit` or `puppeteer` (headless Chrome). `puppeteer` exceeds Vercel function bundle size limits (250 MB). `pdfkit` is manageable but Vercel functions have a 10-second timeout — PDF generation for large continuation sheets could approach this. |
| Admin review | Disconnected from internal portal. JBS staff would need to log into a separate admin interface, or the Vercel app would need its own auth system (new JWT implementation, new user table, new invite flow). This duplicates work already done in `jbs-internal-portal`. |
| White-label | Each new GC would require a new Vercel project or complex multi-tenant routing in `jbs-site`. |
| Deployment | Requires restructuring `jbs-site` build pipeline, a new database service, new env vars in Vercel, and testing the SSR migration. |

**Verdict:** Very high complexity, high regression risk on a production site, and the outcome
is inferior to Option 1 on every meaningful dimension. **Not recommended.**

---

### Option 3 — Separate standalone payment application service

**How it works:**
- New Go service (or Node.js) deployed as its own Railway/Fly.io project
- Own database, own domain/URL (e.g., `pa-api.buildwithjbs.com`)
- Purpose-built for payment applications only
- `jbs-site` calls this service; internal portal would call it or share data via API

**Assessment:**

| Concern | Problem |
|---|---|
| New infrastructure | New Railway project, new PostgreSQL instance, new deployment pipeline, new monitoring, new domain/SSL. All for a feature that fits cleanly in an existing backend. |
| Admin review | Requires building an entire admin UI from scratch — login, user management, review workflow — all of which already exist in `jbs-internal-portal`. |
| Data isolation | Payment applications cannot be cross-referenced with existing JBS jobs, users, or contacts without building a cross-service API or duplicating data. |
| White-label | Same challenge as Option 2 — either a new service per GC, or multi-tenant routing in a new service. The `pa_tenants` model already solves this in Option 1. |
| Complexity | Highest of all three options. |
| When this makes sense | If the payment application tool is extracted as a standalone SaaS product sold to many GCs independently of the JBS internal portal. This is a valid future architectural evolution, not a v1 decision. |

**Verdict:** Appropriate only if the payment application tool becomes a standalone product.
Premature for v1. **Not recommended.**

---

## Decision

**Option 1 is selected: extend the existing `jbs-internal-portal` Go/PostgreSQL backend.**

### Rationale

1. **Zero new infrastructure.** The database, migration runner, deployment pipeline, rate
   limiting, security headers, and CORS handling are already in production.

2. **The public/protected pattern already exists.** `/api/invite/validate/:token` and
   `/api/invite/accept` are already public routes in the same Gin server. Adding payment
   application public routes follows the established pattern.

3. **Admin review is free.** JBS staff already use the internal portal. A new "Payment
   Applications" section in that portal is far cheaper to build than a new admin tool.

4. **Schema/model work is correct.** The four migration files (020–023) and the Go model file
   are exactly right for this architecture. No rework needed.

5. **White-label is already modeled.** `pa_tenants` is the right abstraction for future GCs.
   It lives in the right database.

6. **Option 2 requires a high-risk SSR migration on a live public site for no benefit.**

7. **Option 3 is the right architecture if and when the tool becomes a standalone product.**
   That is not a v1 concern.

---

## Implementation Notes for TICKET-006

These notes are observations only — implementation details belong in the ticket.

**Public routes that must be added (no JWT):**
- `POST /api/v1/payment-applications` — accept submission from `buildwithjbs.com`
- `GET /api/v1/payment-applications/:token` — status/confirmation lookup by `submission_token`

**Protected routes that must be added (JWT + role gate):**
- `GET /api/v1/admin/payment-applications` — admin list
- `GET /api/v1/admin/payment-applications/:id` — admin detail with line items and change orders
- `PATCH /api/v1/admin/payment-applications/:id/review` — mark reviewed (executive, finance, support)

**CORS change required:**
- Add `https://buildwithjbs.com` to the `ALLOWED_ORIGINS` Railway env var (comma-separated,
  alongside the existing internal portal frontend origin). This is the only infrastructure
  change needed before TICKET-006 can be tested end-to-end.

**Rate limiting:**
- The existing `RateLimitGeneral()` middleware applies at the router level and covers the new
  public routes automatically. A dedicated `RateLimitSubmission()` middleware (stricter) is
  recommended for `POST /api/v1/payment-applications` to prevent form spam.

**Migration sequence:**
- Migrations 020–023 must be applied to Railway PostgreSQL before TICKET-006 API handlers go live.
- The existing `RunMigrations()` call in `main.go` will pick them up automatically on next deploy.

---

## Status of Existing Work

| Artifact | Status | Action |
|---|---|---|
| `migrations/020_create_pa_tenants.sql` | ✅ Complete | Continue — apply in TICKET-006 |
| `migrations/021_create_payment_applications.sql` | ✅ Complete | Continue — apply in TICKET-006 |
| `migrations/022_create_payment_application_change_orders.sql` | ✅ Complete | Continue — apply in TICKET-006 |
| `migrations/023_create_payment_application_line_items.sql` | ✅ Complete | Continue — apply in TICKET-006 |
| `internal/models/payment_application.go` | ✅ Complete | Continue — used in TICKET-006 handlers |
| `jbs-site` payment application form (prototype) | ✅ Complete | No change — will wire to backend in TICKET-006 |
| `jbs-site` SSR adapter | Not added | Do not add — `jbs-site` stays static |
| Vercel API routes | Not added | Do not add |
| Separate standalone service | Not started | Do not create |

---

## Future Evolution Path

This decision does not foreclose Option 3. If the payment application tool is later packaged
as a standalone product sold to GCs who do not use the JBS internal portal, the correct path
is:

1. Extract the `pa_tenants`, `payment_applications`, `payment_application_change_orders`, and
   `payment_application_line_items` tables into a new PostgreSQL schema or a new service.
2. Extract the corresponding Go handlers and models into a new Go module.
3. Deploy as an independent service with its own Railway project.

The current schema design (tenant-aware, no foreign keys to JBS-internal tables) was
intentionally kept portable to support this future extraction without a breaking migration.
