# JBS Portal - Complete Implementation Plan

**Project Start**: January 29, 2026  
**Client**: JBS Construction Group (Kelsey)  
**Status**: 🟢 Active Development  
**Last Updated**: January 29, 2026

---

## 📋 Executive Summary

Building a comprehensive internal portal for JBS Construction Group to replace Smartsheet and centralize operations:

1. **Job & Superintendent Management** - Replace Smartsheet with custom database and Gantt chart scheduling
2. **State Licensing Dashboard** - Sync OneDrive files and display interactive US map
3. **Password Manager** - Secure credential vault with audit trails

**Timeline**: 4-5 weeks  
**Tech Stack**: Go + PostgreSQL + Astro + React

---

## 🎯 Core Features (EXPANDED SCOPE)

### Feature 1: Complete Job Lifecycle Management
**Replaces**: Smartsheet (Active Jobs + Completed Jobs)  
**Purpose**: Track 40+ construction jobs from start to completion

**Key Capabilities**:
- Job database with full CRUD operations
- Superintendent assignment and tracking
- Gantt chart timeline visualization
- Project status updates and notes
- Financial tracking (contract values, change orders)
- Executive and PM dashboards
- **NEW**: Archive completed jobs automatically
- **NEW**: Full project history and analytics

### Feature 2: Bid Pipeline & Sales Management
**Replaces**: Smartsheet (Active Bids + Completed Bids)  
**Purpose**: Track 50+ bid opportunities, estimator workload, win/loss analytics

**Key Capabilities**:
- **NEW**: Bid tracking (50+ active opportunities)
- **NEW**: Estimator assignment (Steph Perez, Maria Siebenaller, Lindsay Caruso)
- **NEW**: Status progression (in_progress → complete → awarded)
- **NEW**: Building Connected & Plan Hub integration tracking
- **NEW**: Win/Loss analytics and reporting
- **NEW**: Automatic job creation when bid is won
- **NEW**: Historical bid data for performance metrics

### Feature 3: Superintendent Housing Management
**Replaces**: Smartsheet (AirBnbs)  
**Purpose**: Track temporary lodging for traveling superintendents

**Key Capabilities**:
- **NEW**: AirBnb booking tracking
- **NEW**: Link lodging to specific jobs
- **NEW**: Check-in/out date management
- **NEW**: Cost tracking per job
- **NEW**: Calendar view of who's where
- **NEW**: Budget roll-up (job contract + lodging costs)

### Feature 4: State Licensing Management
**Data Source**: OneDrive folder (not Smartsheet)  
**Purpose**: Track licenses across multiple states with visual dashboard

**Key Capabilities**:
- Interactive US map (click state to see licenses)
- Color-coded status (active/expiring/expired)
- Automatic sync from OneDrive Excel/CSV files
- Expiration alerts and renewal tracking
- State-by-state drill-down views

### Feature 5: Password Manager
**Purpose**: Secure team credential storage with role-based access

**Key Capabilities**:
- AES-256 encrypted password storage
- Category organization
- Sharing with team members
- Complete audit trail (who viewed what, when)
- Copy-to-clipboard functionality

---

## 📊 Complete Smartsheet Replacement Analysis

JBS currently manages **5 Smartsheets** manually that need to be consolidated into the portal:

### 1. Active Jobs (Main Sheet) ✅ IMPORTED
**Records**: 39 jobs  
**Status Distribution**:
- In Progress: 20 jobs
- Close out: 12 jobs
- Permitting: 4 jobs
- Not Started: 3 jobs

**Fields**: Job Number, Location, Client, Status, PM, APM, Superintendent, Contract Value, Dates, Meeting Minutes

### 2. Completed Jobs (Historical Archive)
**Records**: 13 completed jobs  
**Purpose**: Historical record of finished projects  
**Status**: Complete, Close out  
**Same Fields as Active Jobs**

**Issue**: Manual process - jobs must be copy/pasted from Active to Completed when done

### 3. Active Bids (Sales Pipeline)
**Records**: 51 active bid opportunities  
**Clients**: Auto Zone (majority), Driven Brands, Mister Car Wash, Panda, Misc  
**Fields**:
- Bid Client
- Location
- Due Date
- Assigned To (estimator: Steph Perez, Maria Siebenaller, Lindsay Caruso)
- Status (Complete, no access, did not bid)
- Building Connected (date uploaded)
- Plan Hub (date uploaded)
- Awarded (Yes/No/blank)

**Issue**: No automatic status progression; must manually update and move to Completed Bids

### 4. Completed Bids (Historical Win/Loss)
**Records**: 35 historical bids  
**Purpose**: Track win rate, estimator performance, client relationships  
**Awarded Field**: Shows if bid was won (becomes a job) or lost

**Issue**: Manual tracking - no analytics on win rate, no connection to jobs that were won

### 5. AirBnbs (Superintendent Housing)
**Records**: 11 superintendents with lodging assignments  
**Fields**:
- Superintendent Name
- AirBnb Location (city/state)
- Check In Date
- Check Out Date
- Property Link (Airbnb URL)
- Address
- Jobsite Address

**Purpose**: Track temporary housing when superintendents travel for jobs  
**Issue**: No connection to jobs table; hard to see who's where and budget for lodging

**Superintendents** (10+ active):
- Alan Tyminski, James Elliott, Martin Wade, Enrique McFarlane
- Graham Dickson, Wolf Ishcomer, Cody Goatley, Jason Buchanan
- Sam Clark, Wally Logan, Aaron Vess

**Geographic Spread**: FL, AZ, OH, MI, VA, NM, KY, WI, TN, ID, AL, CO

---

## 🔄 Complete Workflow & Data Relationships

```
📝 BID SUBMITTED
  ├─> Active Bids table
  │
  ├─> Status: Complete (bid submitted)
  │
  ├─> Awarded = "Yes" ──────┐
  │                         │
  │                    ✅ WON
  │                         │
  └─> Awarded = "No"        ├─> Manual copy to Active Jobs
      Awarded = blank       │   (currently manual)
          │                 │
          └─> Move to       └─> ACTIVE JOB
              Completed         ├─> Assign Superintendent
              Bids              ├─> Book AirBnb (if needed)
              (manual)          ├─> Track progress
                                ├─> Update contract values
                                ├─> Meeting notes
                                │
                                └─> Status = Complete
                                    │
                                    └─> Move to Completed Jobs
                                        (currently manual)
```

**Problems with Current Manual Process**:
1. ❌ No connection between Bids → Jobs (have to re-enter data)
2. ❌ No automatic progression (Bid Won → Create Job)
3. ❌ No analytics (win rate, estimator performance)
4. ❌ AirBnbs not linked to jobs (can't see lodging costs per project)
5. ❌ Manual archiving (copy/paste to Completed tables)
6. ❌ Can't track full project lifecycle (bid → job → completion)
7. ❌ No budget roll-up (job contract + lodging costs)

---

## 🏗️ Technical Architecture

### Database Schema

```sql
-- ============================================
-- JOBS & SUPERINTENDENTS
-- ============================================

-- Clients table
CREATE TABLE clients (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    contact_name VARCHAR(255),
    contact_email VARCHAR(255),
    contact_phone VARCHAR(50),
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Superintendents table
CREATE TABLE superintendents (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    certifications TEXT[],
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- BIDS & PIPELINE (NEW)
-- ============================================

CREATE TABLE bids (
    id SERIAL PRIMARY KEY,
    client_id INTEGER REFERENCES clients(id),
    location VARCHAR(255) NOT NULL,
    city VARCHAR(100),
    state VARCHAR(2),
    due_date DATE,
    assigned_to_id INTEGER REFERENCES users(id), -- Estimator
    assigned_to_name VARCHAR(255), -- For legacy import
    status VARCHAR(50) DEFAULT 'in_progress', -- in_progress, complete, no_access, did_not_bid, not_started
    building_connected_date DATE, -- Date uploaded to Building Connected
    plan_hub_date DATE, -- Date uploaded to Plan Hub
    awarded VARCHAR(10), -- 'Yes', 'No', NULL
    bid_amount DECIMAL(12, 2),
    notes TEXT,
    job_id INTEGER REFERENCES jobs(id), -- Link to job if bid was won
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    archived BOOLEAN DEFAULT FALSE, -- Move to "Completed Bids"
    INDEX idx_bids_status (status),
    INDEX idx_bids_archived (archived),
    INDEX idx_bids_awarded (awarded)
);

-- ============================================
-- SUPERINTENDENT LODGING (NEW)
-- ============================================

CREATE TABLE superintendent_lodging (
    id SERIAL PRIMARY KEY,
    superintendent_id INTEGER REFERENCES superintendents(id) ON DELETE CASCADE,
    job_id INTEGER REFERENCES jobs(id) ON DELETE SET NULL,
    location VARCHAR(255), -- City, State
    check_in_date DATE,
    check_out_date DATE,
    property_link TEXT, -- AirBnB URL
    address VARCHAR(255),
    jobsite_address VARCHAR(255),
    cost_per_night DECIMAL(10, 2),
    total_cost DECIMAL(10, 2),
    booking_confirmation VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_lodging_super (superintendent_id),
    INDEX idx_lodging_job (job_id),
    INDEX idx_lodging_dates (check_in_date, check_out_date)
);

-- Jobs table
CREATE TABLE jobs (
    id SERIAL PRIMARY KEY,
    job_number VARCHAR(50) UNIQUE NOT NULL,
    job_name VARCHAR(255) NOT NULL,
    location VARCHAR(255),
    city VARCHAR(100),
    state VARCHAR(2),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    client_id INTEGER REFERENCES clients(id),
    status VARCHAR(50),
    contract_value DECIMAL(12, 2),
    revised_contract_value DECIMAL(12, 2),
    start_date DATE,
    projected_end_date DATE,
    actual_end_date DATE,
    project_manager_id INTEGER REFERENCES users(id),
    apm_id INTEGER REFERENCES users(id),
    superintendent_id INTEGER REFERENCES superintendents(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Job updates/notes table
CREATE TABLE job_updates (
    id SERIAL PRIMARY KEY,
    job_id INTEGER REFERENCES jobs(id) ON DELETE CASCADE,
    author_id INTEGER REFERENCES users(id),
    author_name VARCHAR(255),
    update_text TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Change orders table
CREATE TABLE change_orders (
    id SERIAL PRIMARY KEY,
    job_id INTEGER REFERENCES jobs(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    amount DECIMAL(12, 2),
    status VARCHAR(50) DEFAULT 'pending',
    created_by INTEGER REFERENCES users(id),
    approved_by INTEGER REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    approved_at TIMESTAMP
);

-- ============================================
-- STATE LICENSING
-- ============================================

CREATE TABLE state_licenses (
    id SERIAL PRIMARY KEY,
    state VARCHAR(2) NOT NULL,
    license_type VARCHAR(100) NOT NULL,
    license_number VARCHAR(100),
    entity_name VARCHAR(255),
    issue_date DATE,
    expiration_date DATE,
    status VARCHAR(50),
    renewal_fee DECIMAL(10, 2),
    notes TEXT,
    onedrive_file_path TEXT,
    last_synced_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- PASSWORD MANAGER
-- ============================================

CREATE TABLE credentials (
    id SERIAL PRIMARY KEY,
    category VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    username VARCHAR(255),
    encrypted_password TEXT NOT NULL,
    url TEXT,
    notes TEXT,
    created_by INTEGER REFERENCES users(id),
    shared_with INTEGER[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE credential_audit_log (
    id SERIAL PRIMARY KEY,
    credential_id INTEGER REFERENCES credentials(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id),
    action VARCHAR(50) NOT NULL,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================
-- SYNC LOGS
-- ============================================

CREATE TABLE onedrive_sync_log (
    id SERIAL PRIMARY KEY,
    sync_type VARCHAR(50),
    records_processed INTEGER,
    records_created INTEGER,
    records_updated INTEGER,
    errors TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);
```

### Backend Structure (Go)

```
backend/
├── cmd/
│   ├── server/
│   │   └── main.go
│   └── migrate/
│       └── import_smartsheet.go    # CSV import script
├── internal/
│   ├── models/
│   │   ├── job.go
│   │   ├── client.go
│   │   ├── superintendent.go
│   │   ├── license.go
│   │   └── credential.go
│   ├── services/
│   │   ├── jobs/
│   │   │   ├── service.go
│   │   │   └── scheduler.go
│   │   ├── onedrive/
│   │   │   ├── client.go
│   │   │   ├── sync.go
│   │   │   └── file_processor.go
│   │   └── password_manager/
│   │       ├── encryption.go
│   │       ├── vault.go
│   │       └── audit.go
│   └── handlers/
│       ├── job_handler.go
│       ├── superintendent_handler.go
│       ├── license_handler.go
│       └── credential_handler.go
```

### Frontend Structure (Astro + React)

```
frontend/src/
├── pages/
│   ├── jobs/
│   │   ├── index.astro          # Job list/dashboard
│   │   ├── [id].astro           # Job detail
│   │   ├── new.astro            # Create job
│   │   └── schedule.astro       # Gantt chart
│   ├── superintendents/
│   │   ├── index.astro          # Super list
│   │   └── map.astro            # Job map view
│   ├── licensing/
│   │   └── index.astro          # US map + licenses
│   └── passwords/
│       └── index.astro          # Password vault
├── components/
│   ├── JobForm.tsx
│   ├── JobList.tsx
│   ├── JobGantt.tsx
│   ├── SuperMap.tsx             # Leaflet job map
│   ├── StateLicenseMap.tsx      # US state map
│   └── PasswordVault.tsx
```

---

## 📅 Implementation Timeline (UPDATED WITH EXPANDED SCOPE)

**Total Duration**: 5-6 weeks  
**Phased Delivery**: Core features first, then enhancements

### **Week 1: Foundation & Core Job Management** ✅ IN PROGRESS

#### Days 1-2: Database & Models ✅ MOSTLY COMPLETE
- [x] Environment setup (.env file created)
- [x] Create database migrations (9 tables created)
  - [x] Jobs, clients, superintendents tables
  - [x] Change orders, job updates tables
  - [x] State licenses table
  - [x] Credentials tables
  - [ ] **NEW**: Bids table
  - [ ] **NEW**: Superintendent lodging table
- [x] Run migrations on development database
- [x] Create Go models for all entities
- [x] Import existing 39 jobs from CSV (✅ COMPLETED!)
- [ ] **NEW**: Import bids data (51 active + 35 completed)
- [ ] **NEW**: Import AirBnb/lodging data (11 records)

#### Days 3-5: Job & Bid Management Backend
**Jobs API**:
- [ ] Job API endpoints
  - `GET /api/jobs` - List all jobs (with filters)
  - `GET /api/jobs/:id` - Get job details
  - `POST /api/jobs` - Create job
  - `PUT /api/jobs/:id` - Update job
  - `DELETE /api/jobs/:id` - Archive job (soft delete)
  - `POST /api/jobs/:id/archive` - Move to completed
- [ ] Client API endpoints
- [ ] Superintendent API endpoints
- [ ] Job updates/notes API
- [ ] Geocoding service for addresses

**NEW - Bids API**:
- [ ] Bid API endpoints
  - `GET /api/bids` - List all bids (active by default)
  - `GET /api/bids?archived=true` - Completed bids
  - `GET /api/bids/:id` - Get bid details
  - `POST /api/bids` - Create bid
  - `PUT /api/bids/:id` - Update bid
  - `POST /api/bids/:id/award` - Mark as awarded, create job
  - `POST /api/bids/:id/archive` - Move to completed
- [ ] Bid → Job conversion service (when awarded)
- [ ] Estimator workload analytics

**NEW - Lodging API**:
- [ ] Lodging API endpoints
  - `GET /api/lodging` - List all lodging
  - `GET /api/lodging/superintendent/:id` - By super
  - `GET /api/lodging/job/:id` - By job
  - `POST /api/lodging` - Create booking
  - `PUT /api/lodging/:id` - Update booking
  - `DELETE /api/lodging/:id` - Delete booking
- [ ] Cost calculation service (auto-calculate total from dates)

### **Week 2: Frontend - Jobs, Bids, Lodging**

#### Days 6-8: Job Management UI
- [ ] Navigation menu updates (add Bids, Lodging sections)
- [ ] Job list page with filters
  - Filter by status, client, PM, super
  - Search by job number or location
  - Sort by any column
  - **NEW**: Quick archive button
- [ ] Job detail page
  - All job information
  - Timeline of updates/notes
  - Change orders section
  - **NEW**: Linked lodging reservations
  - **NEW**: Source bid (if created from bid)
- [ ] Job form (create/edit)
  - Client dropdown (searchable)
  - Super dropdown
  - PM dropdown
  - Date pickers
  - Currency inputs with validation

#### Days 9-10: Bid Pipeline UI (NEW)
- [ ] Bid list page
  - Active bids tab
  - Completed bids tab
  - Filter by client, estimator, status
  - Search by location
  - Sort by due date
  - Win/loss indicators
- [ ] Bid detail page
  - All bid information
  - Award/reject actions
  - Link to job (if awarded)
  - Building Connected / Plan Hub tracking
- [ ] Bid form (create/edit)
  - Client dropdown
  - Estimator dropdown
  - Location with city/state parsing
  - Due date picker
  - Upload tracking dates
- [ ] **Bid → Job workflow**
  - "Award Bid" button → create job modal
  - Pre-fill job data from bid
  - Link bid and job together
- [ ] Analytics dashboard
  - Win rate by estimator
  - Win rate by client
  - Average bid amount
  - Pipeline value ($)

#### Days 11-12: Lodging Management UI (NEW)
- [ ] Lodging list page
  - Current reservations
  - Upcoming check-ins (next 7 days)
  - Filter by superintendent
  - Calendar view option
- [ ] Lodging detail/form
  - Superintendent dropdown
  - Job dropdown (optional)
  - Location, dates
  - Property link (AirBnb URL)
  - Address fields
  - Cost fields (auto-calculate total)
- [ ] Superintendent detail page enhancement
  - Show current/upcoming lodging
  - Show job history
  - Show available dates
- [ ] Job detail page enhancement
  - Show assigned lodging
  - Total lodging cost for this job

### **Week 3: Gantt Chart & Advanced Features**

#### Days 13-14: Gantt Chart Scheduling
- [ ] Install FullCalendar Timeline library
- [ ] Create Gantt component
  - View by superintendent (rows = supers)
  - View by job (rows = jobs)
  - Color-coded by status
  - Click to see details
- [ ] Gantt chart page with stats
  - Active supers count
  - Total assignments
  - Availability gaps
- [ ] Availability gap detection service

### **Week 3: State Licensing (OneDrive Integration)**

#### Days 11-12: OneDrive Setup & Backend
**Prerequisites**:
- [ ] Get Azure credentials from client
  - AZURE_TENANT_ID
  - AZURE_CLIENT_ID
  - AZURE_CLIENT_SECRET
  - API permissions granted

**OneDrive Folder Structure** (confirmed):
```
/State Licensing/
├── State 1/
│   ├── [subdirectories - TBD]
│   └── license files (Excel/PDF/etc)
├── State 2/
│   ├── [subdirectories - TBD]
│   └── license files
└── ...
```

- [ ] Document exact subdirectory structure
- [ ] Document Excel/CSV file formats per state

**Development**:
- [ ] Microsoft Graph API client
- [ ] OneDrive file listing
- [ ] Excel/CSV file parser
- [ ] License sync service (manual + scheduled)
- [ ] State summary API endpoints
  - `/api/licenses/state-summary` - All states with status
  - `/api/licenses/state/:code` - Detail for one state
  - `/api/licenses/expiring?days=90` - Expiring licenses
- [ ] Background sync job (daily at 2 AM)

#### Days 13-14: Licensing Frontend
- [ ] Install react-simple-maps for US map
- [ ] Interactive US state map component
  - Color-coded by status
  - Hover tooltips
  - Click to drill down
- [ ] State detail modal
- [ ] License table view (toggle from map)
- [ ] Filters and search
- [ ] Sync button (manual trigger)
- [ ] Expiration alerts

### **Week 4: Password Manager**

#### Days 15-17: Password Manager Backend
- [ ] AES-256 encryption service
- [ ] Generate encryption key (add to .env)
- [ ] Vault service
  - Create/read/update/delete credentials
  - Encryption/decryption
  - Access control (sharing)
- [ ] Audit logging service
- [ ] API endpoints
  - `/api/credentials` - List (no passwords)
  - `/api/credentials/:id` - Get with password
  - `POST /api/credentials` - Create
  - `PUT /api/credentials/:id` - Update
  - `POST /api/credentials/:id/share` - Share
  - `/api/credentials/:id/audit` - View audit log

#### Days 18-19: Password Manager Frontend
- [ ] Password vault UI
  - Grid/card layout
  - Category filtering
  - Search functionality
- [ ] Add/edit credential modal
- [ ] View credential modal
  - Show/hide password toggle
  - Copy to clipboard
  - Logs audit event on view
- [ ] Sharing interface
- [ ] Audit log viewer
- [ ] Password strength indicator

### **Week 5: Polish, Testing & Deployment**

#### Days 20-21: Dashboards & Reports
- [ ] Executive dashboard
  - Total jobs, contract value
  - Jobs by status (pie chart)
  - Jobs by client (bar chart)
  - Revenue metrics
- [ ] PM dashboard (filtered to their jobs)
- [ ] Superintendent utilization report
- [ ] Financial reports
  - Contract values
  - Change orders summary

#### Days 22-23: Testing & Bug Fixes
- [ ] End-to-end testing all features
- [ ] Mobile responsiveness check
- [ ] Performance optimization
- [ ] Security audit
- [ ] User acceptance testing with client

#### Day 24: Deployment
- [ ] Production environment setup
- [ ] Database migration on production
- [ ] Import production data
- [ ] SSL certificates
- [ ] Deploy backend to Railway
- [ ] Deploy frontend to Vercel
- [ ] Configure CORS for production
- [ ] Set up monitoring and logs

---

## 🔐 Required Credentials & Setup

### OneDrive / Microsoft Graph API

**Status**: ⏳ Waiting on client

**Steps**:
1. Azure Portal → App Registrations → New registration
2. Name: "JBS Portal OneDrive Integration"
3. Copy Application (client) ID
4. Copy Directory (tenant) ID
5. Certificates & secrets → New client secret → Copy value
6. API permissions → Microsoft Graph → Application permissions:
   - Files.Read.All
   - Sites.Read.All
7. Grant admin consent
8. Add to `.env`:
   ```
   AZURE_TENANT_ID=...
   AZURE_CLIENT_ID=...
   AZURE_CLIENT_SECRET=...
   ONEDRIVE_USER_EMAIL=licensing@jbs.com
   ONEDRIVE_FOLDER_PATH=/State Licensing
   ```

### Encryption Key (Password Manager)

**Status**: ✅ Generated

Already added to `.env`:
```
ENCRYPTION_KEY=VsyE+w/fYHoz3yHyDwFgCqmMW7SoFt2Uq/fGteYYo+4=
```

---

## 🎨 UI/UX Components

### Job List Page

**Features**:
- Table view with sortable columns
- Status badges (color-coded)
- Quick filters (dropdown + search)
- Pagination
- Export to Excel
- "Add New Job" button

**Columns**:
- Job Number
- Location
- Client
- Status
- Superintendent
- PM
- Start Date
- End Date
- Contract Value
- Actions (Edit, View)

### Job Detail Page

**Sections**:
1. Header: Job number, location, status badge
2. Key Info Cards: Contract value, dates, people
3. Notes Timeline: Chronological updates with @ mentions
4. Change Orders: List with status and amounts
5. Documents: File attachments (future)
6. Action Buttons: Edit, Delete, Add Update

### Gantt Chart Page

**Views**:
- Toggle: View by Superintendent vs View by Job
- Timeline: Month / Quarter / Year
- Legend: Color codes for status
- Stats cards: Active supers, assignments, availability gaps

**Features**:
- Zoom in/out on timeline
- Click bar to see job details
- Hover to see quick info
- Print/export timeline

### State Licensing Map

**Features**:
- Interactive US map
- Color-coded states
- Hover tooltips with summary
- Click to drill down
- Quick stats below map
- Toggle to table view

**Color Coding**:
- 🟢 Green = All licenses active
- 🟡 Yellow = Expiring soon (< 90 days)
- 🔴 Red = Expired licenses
- ⚪ Gray = No licenses

### Password Vault

**Features**:
- Card grid layout
- Category tabs/filter
- Search bar
- "Add Credential" button
- Click card to view details
- Copy password to clipboard
- Audit log per credential

**Security**:
- Password hidden by default
- Click to reveal (logs audit)
- Auto-hide after 30 seconds
- Re-auth for sensitive operations

---

## 📦 Dependencies & Libraries

### Backend (Go)

```bash
# Already installed
github.com/gin-gonic/gin          # Web framework
github.com/lib/pq                 # PostgreSQL driver
github.com/golang-jwt/jwt/v5      # JWT auth
github.com/xuri/excelize/v2       # Excel processing
golang.org/x/crypto               # Encryption

# To install
go get github.com/microsoftgraph/msgraph-sdk-go
go get github.com/Azure/azure-sdk-for-go/sdk/azidentity
go get github.com/robfig/cron/v3  # Scheduled jobs
```

### Frontend (Astro + React)

```bash
# To install
npm install leaflet react-leaflet @types/leaflet
npm install @fullcalendar/core @fullcalendar/react @fullcalendar/timeline @fullcalendar/resource-timeline
npm install react-simple-maps d3-geo
npm install recharts  # Charts for dashboards
npm install date-fns  # Date formatting
```

---

## 🧪 Testing Checklist

### Job Management
- [ ] Create new job
- [ ] Edit existing job
- [ ] Delete job (soft delete)
- [ ] Assign superintendent
- [ ] Add job update/note
- [ ] Create change order
- [ ] Filter jobs by status
- [ ] Search jobs
- [ ] View job detail
- [ ] Gantt chart displays correctly
- [ ] Availability gaps calculated correctly

### State Licensing
- [ ] OneDrive connection successful
- [ ] File listing works
- [ ] Excel/CSV parsing accurate
- [ ] Manual sync trigger
- [ ] Scheduled sync runs
- [ ] Map displays all states
- [ ] State click shows details
- [ ] Color coding correct
- [ ] Expiring licenses alert
- [ ] Table view filters work

### Password Manager
- [ ] Create credential
- [ ] View credential (logs audit)
- [ ] Copy password to clipboard
- [ ] Update credential
- [ ] Delete credential
- [ ] Share with team member
- [ ] Category filtering
- [ ] Search credentials
- [ ] Audit log accurate
- [ ] Encryption/decryption works

---

## 🚀 Deployment Configuration

### Backend (Railway)

```toml
# railway.toml
[build]
builder = "NIXPACKS"

[deploy]
startCommand = "cd cmd/server && go run main.go"
healthcheckPath = "/health"
healthcheckTimeout = 100
restartPolicyType = "ON_FAILURE"
```

**Environment Variables** (set in Railway):
- DATABASE_URL
- JWT_SECRET
- AZURE_TENANT_ID
- AZURE_CLIENT_ID
- AZURE_CLIENT_SECRET
- ENCRYPTION_KEY
- ALLOWED_ORIGINS
- GIN_MODE=release

### Frontend (Vercel)

```json
// vercel.json
{
  "buildCommand": "npm run build",
  "outputDirectory": "dist",
  "framework": "astro"
}
```

**Environment Variables** (set in Vercel):
- PUBLIC_API_URL=https://jbs-portal-backend.railway.app

---

## 📊 Success Metrics

### Performance
- Job list load time: < 2 seconds
- Gantt chart render: < 3 seconds
- OneDrive sync: < 2 minutes
- Password retrieval: < 1 second

### Adoption
- **All 39 active jobs migrated successfully** ✅
- **All 51 active bids imported**
- **All 35 completed bids imported**
- **All 13 completed jobs imported**
- **All 11 superintendent lodging records imported**
- All team members have accounts
- 90% of updates done in portal (not Smartsheet)
- **Zero Smartsheet subscription after 30 days**
- **Estimators track bids in portal, not spreadsheets**

### User Satisfaction
- Mobile access works smoothly
- Search finds results quickly
- Reports export correctly
- No data loss incidents
- **Bid → Job conversion is seamless (one click)**
- **Lodging costs automatically roll up into job budgets**

---

## 🎓 Key Insights from Smartsheet Analysis

### Pain Points Identified
1. **Manual Data Entry** - Same data entered multiple times (Bid → Job requires copy/paste)
2. **No Relationships** - Bids, Jobs, and Lodging are separate sheets with no connections
3. **No Automation** - Moving from Active → Completed requires manual copy/paste
4. **No Analytics** - Can't calculate win rate, estimator performance, or profitability
5. **No Budget Roll-up** - Can't see total project cost (job + lodging + change orders)
6. **Limited Search** - Hard to find related records across sheets
7. **No History** - Can't track full lifecycle (Bid → Job → Completion)
8. **Slow Updates** - Team avoids updating because it's tedious

### Portal Advantages
1. ✅ **One-Click Workflows** - Award bid → automatically creates job
2. ✅ **Connected Data** - Bids link to jobs, jobs link to lodging, all queryable
3. ✅ **Automatic Archiving** - Complete a job → moves to archive with one click
4. ✅ **Real-Time Analytics** - Win rate, estimator performance, budget tracking
5. ✅ **Smart Budget** - Job contract + lodging + change orders = total project cost
6. ✅ **Powerful Search** - Find anything across all tables instantly
7. ✅ **Full History** - See complete timeline from bid to completion
8. ✅ **Fast Updates** - Mobile-friendly, quick forms, auto-save

### Data Migration Summary

| Sheet | Records | Status | Import Script |
|-------|---------|--------|---------------|
| Active Jobs | 39 | ✅ Imported | import_smartsheet.go |
| Completed Jobs | 13 | ⏳ Pending | import_completed_jobs.go |
| Active Bids | 51 | ⏳ Pending | import_bids.go |
| Completed Bids | 35 | ⏳ Pending | import_bids.go (archived=true) |
| AirBnbs (Lodging) | 11 | ⏳ Pending | import_lodging.go |
| **Total** | **149** | **26% Complete** | |

---

## 🔄 Future Enhancements (Post-Launch)

### Phase 2 Ideas
1. **Mobile App** - Native iOS/Android
2. **Document Management** - Upload job photos, contracts, permits
3. **Financial Integration** - Link to QuickBooks/accounting
4. **Email Notifications** - Job updates, license expirations
5. **SMS Alerts** - Super assignment notifications
6. **Time Tracking** - Super hours per job
7. **Budget Tracking** - Actual vs projected costs
8. **Subcontractor Management** - Track subs per job
9. **Equipment Tracking** - Assign equipment to jobs
10. **Weather Integration** - Weather delays tracking

### Advanced Features
- AI-powered scheduling optimization
- Predictive analytics for project delays
- Automated invoice generation
- Client portal (read-only access)
- Integration with project management tools

---

## 📝 Change Log

| Date | Change | By |
|------|--------|-----|
| 1/29/26 | Initial plan created | Will |
| 1/29/26 | Analyzed Smartsheet CSV data (Active Jobs) | Will |
| 1/29/26 | Added encryption key to .env | Will |
| 1/29/26 | Consolidated into single living document | Will |
| 1/29/26 | **EXPANDED SCOPE**: Added 4 more Smartsheets (Bids, Completed, Lodging) | Will |
| 1/29/26 | Database migrations created (9 tables) | Will |
| 1/29/26 | ✅ Imported 39 Active Jobs successfully | Will |
| 1/29/26 | Added Bids and Lodging tables to schema | Will |
| 1/29/26 | Updated timeline to reflect expanded scope (5-6 weeks) | Will |

---

## 🎯 Current Status & Next Actions

### ✅ Completed
- [x] Requirements gathering with client (Kelsey)
- [x] Smartsheet data export and analysis (ALL 5 sheets)
- [x] Technical architecture design
- [x] Database schema design (11 tables total)
- [x] .env file setup with encryption key
- [x] Implementation plan documented
- [x] **Database migrations created (9 tables)**
- [x] **Active Jobs imported (39 records)**

### 🔄 In Progress
- [ ] Building API endpoints for jobs
- [ ] Creating import scripts for remaining sheets

### 🎯 Up Next (Immediate)
1. ✅ ~~Import active jobs~~ DONE
2. Build job CRUD API endpoints
3. Import bids data (51 active + 35 completed)
4. Import completed jobs (13 records)
5. Import lodging data (11 records)
6. Build bid management API
7. Build lodging management API

### 🚧 Blocked
- OneDrive integration (waiting on Azure credentials)

---

**Project Owner**: Will McCurry (ITWill)  
**Client Contact**: Kelsey @ JBS Construction Group  
**Documentation**: Living document - update as project evolves  
**Scope**: **EXPANDED** - Now replacing all 5 Smartsheets + OneDrive sync + Password Manager
