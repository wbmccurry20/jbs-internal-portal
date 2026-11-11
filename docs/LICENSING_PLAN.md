# Licensing & SharePoint Integration Plan

## License tracking and alerts

The licensing page supports state and city records. Enter a city and check
**City License** for each actual city license; leave both blank for a state
license. Link the actual license document from the SharePoint browser after
creating the record. SharePoint must be configured and connected, and users
must have access to the document.

Use **Find licenses** to search by state, city, license type, number, or entity,
and filter by status or jurisdiction. Results are ordered by expiration date.
The map includes both state and city records; a city's expired or expiring
license can affect the state's displayed status. A state with no records has
not necessarily been reviewed or determined to require no license.

For records that are not explicitly pending, status is computed each time
licenses are fetched using the America/New_York calendar date: expired before
today, expiring from today through 60 days out, active beyond 60 days, and
needs review when no expiration date is recorded. Pending records remain
pending and are excluded from expiration alerts until changed.

### Email alert configuration

Email alerts are disabled until `LICENSE_ALERT_RECIPIENTS` is configured on the
backend. Set it to a comma-separated list of recipient addresses. Configure
`SMTP_HOST`, `SMTP_USER`, `SMTP_PASSWORD`, and optionally `SMTP_PORT` (default
587) and `SMTP_FROM` (default `SMTP_USER`). The SMTP server must support
STARTTLS. Check backend logs for configuration or delivery errors.

Once enabled, the worker runs at startup and hourly while the backend is
running. It sends reminders at 60 and 30 calendar days before expiration,
using America/New_York. If an active record enters the system inside either
window, or the backend was unavailable on the scheduled day, it sends one
catch-up reminder for the most recent threshold; a later 30-day reminder still
applies when appropriate. Already expired, pending, and undated records do not
generate reminders. The database records delivery per license, expiration date,
and threshold, so a new expiration date starts a new reminder cycle. If SMTP
delivery fails, the worker logs the error and retries on a later run.

Alerts are at-least-once: if SMTP accepts a message but the database commit
fails, a retry can send a duplicate. The worker must be running for alerts to
be delivered. Before enabling alerts in production, verify SMTP configuration,
recipient addresses, backend uptime, and logs; enabling can immediately send
catch-up reminders for existing licenses.

## Phased Approach

The JBS SharePoint site (`netorg4205680.sharepoint.com/sites/JBS`) contains licensing documents organized by state and city. The portal manages structured license data while SharePoint remains the document store.

### Phase 1: License Records ✅ DONE
- Manual CRUD for license data (state, city, type, number, dates, status)
- Interactive US map with per-state status
- All 50 states in dropdown, city license support
- Auto-calculated expiration status

### Phase 2: SharePoint Folder Browser
- Browse the `Licensing/` folder from inside the portal
- Navigate state → city → document hierarchy
- Click a document → opens in SharePoint
- Requires Azure App Registration (Graph API: `Sites.Read.All`, `Files.Read.All`)

### Phase 3: Document Linking
- Attach SharePoint document URLs to license records
- `onedrive_file_path` column already exists in `state_licenses`
- Click "View Document" on a license → opens linked PDF in SharePoint
- Link multiple docs per license (application, bond, COI, certificate)

### Phase 4: Auto-Suggest
- Scan SharePoint folder hierarchy to propose new license records
- Parse "Alabama (AL)" → state=AL, "Birmingham" → city
- Admin reviews suggestions before anything enters the DB
- Never auto-imports — human always confirms

## SharePoint Folder Structure (observed)
```
Licensing/
  {State Name} ({Abbr})/
    .Application/           ← state-level application docs
    .Renewal-{YYYY}/        ← renewal paperwork
    {City Name}/            ← city-level license
      Application/          ← city application docs
    License_Copy.pdf        ← actual state license
```

## Key Principle
**Files stay on SharePoint. Structured data lives in the portal DB. The portal links to SharePoint for documents.**
