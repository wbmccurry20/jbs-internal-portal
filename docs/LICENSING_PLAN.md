# Licensing & SharePoint Integration Plan

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
