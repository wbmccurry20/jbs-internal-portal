# OneDrive License Sync Integration

## Current Status: PENDING AZURE CREDENTIALS

The license management feature is currently in **portal-first mode** with manual CRUD operations. OneDrive integration is ready to implement once Azure credentials are available.

---

## Production Data Management

### Current Setup
- License management via portal UI (CRUD operations)
- Manual data entry through licensing page
- Interactive US map visualization

---

## OneDrive Integration Plan

### Prerequisites

1. **Azure App Registration**
   - Go to Azure Portal → App Registrations
   - Create new registration for "JBS Portal OneDrive Sync"
   - Note: Application (client) ID
   - Note: Directory (tenant) ID
   - Create client secret, note the value

2. **API Permissions**
   - Add Microsoft Graph API permissions:
     - `Files.Read.All` (read OneDrive files)
     - `Files.ReadWrite.All` (for bi-directional sync)
   - Grant admin consent

3. **OneDrive Folder Structure**
   Recommended folder organization:
   ```
   /JBS Construction/
     /Licenses/
       /Active/
         CA-General-Contractor-123456.pdf
         TX-Commercial-Builder-789012.pdf
       /Expired/
         NV-General-Building-567890.pdf
   ```

4. **Environment Variables**
   Add to `.env`:
   ```bash
   # Azure OneDrive Sync
   AZURE_CLIENT_ID=your-client-id
   AZURE_CLIENT_SECRET=your-client-secret
   AZURE_TENANT_ID=your-tenant-id
   ONEDRIVE_FOLDER_PATH=/JBS Construction/Licenses
   ONEDRIVE_SYNC_ENABLED=true
   ```

---

## Implementation Steps

### Phase 1: Read-Only Sync (Week 1)
- [ ] Implement OAuth 2.0 authentication with Azure
- [ ] Connect to OneDrive API
- [ ] List files in configured folder
- [ ] Parse metadata from filenames
- [ ] Import to database (create/update records)
- [ ] Store `onedrive_file_path` reference
- [ ] Track sync in `onedrive_sync_log` table

### Phase 2: Bi-Directional Sync (Week 2)
- [ ] Portal uploads → save to OneDrive
- [ ] OneDrive changes → update portal
- [ ] Conflict resolution strategy
- [ ] File versioning

### Phase 3: Advanced Features (Week 3)
- [ ] OCR for extracting data from PDFs
- [ ] Automated expiration email alerts
- [ ] Scheduled sync (cron job)
- [ ] Webhook notifications on OneDrive changes

---

## File Naming Convention

For automatic metadata parsing, use this format:
```
{STATE}-{LICENSE_TYPE}-{LICENSE_NUMBER}.pdf

Examples:
CA-GeneralContractor-GC123456.pdf
TX-CommercialBuilder-CB789012.pdf
FL-GeneralContractor-GC345678.pdf
```

Alternatively, use a metadata JSON sidecar:
```
CA-GeneralContractor-GC123456.pdf
CA-GeneralContractor-GC123456.json  ← contains structured data
```

---

## Database Schema Reference

The `state_licenses` table already has OneDrive fields:
- `onedrive_file_path TEXT` - Full path to file in OneDrive
- `last_synced_at TIMESTAMP` - Last sync timestamp

The `onedrive_sync_log` table tracks sync history:
- Sync start/end times
- Files processed
- Errors encountered
- Records added/updated

---

## API Endpoints (Future)

### Sync Operations
- `POST /api/licenses/sync/onedrive` - Trigger manual sync
- `GET /api/licenses/sync/status` - Get last sync status
- `GET /api/licenses/sync/history` - View sync log

### File Operations
- `GET /api/licenses/:id/download` - Download PDF from OneDrive
- `POST /api/licenses/:id/upload` - Upload/replace PDF in OneDrive
- `GET /api/licenses/:id/preview` - PDF preview URL

---

## Testing Plan

1. **Local Testing**
   - Use OneDrive Personal account
   - Create test folder with sample PDFs
   - Test authentication flow
   - Verify metadata parsing

2. **Staging**
   - Use company OneDrive
   - Import subset of real licenses
   - Test bi-directional sync
   - Monitor for errors

3. **Production**
   - Schedule initial full sync during off-hours
   - Monitor sync logs
   - Set up automated daily sync at 2 AM

---

## Rollback Plan

If OneDrive sync has issues:
1. Disable sync: `ONEDRIVE_SYNC_ENABLED=false`
2. Portal continues working with manual CRUD
3. Database remains authoritative source
4. Fix sync issues offline
5. Re-enable once stable

---

## Contact

**Azure Admin Access Needed From:**
- IT Department or Azure tenant admin
- Provide app registration details
- Grant API permissions

**Current Owner:** Will McCurry (ITWill)  
**Implementation Timeline:** TBD based on credential availability
