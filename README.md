# JBS Internal Portal

Web-based financial tools for JBS including Concur expense conversion and bank reconciliation.

## 🎯 Overview

This replaces the desktop `jbs-concur-converter` with a modern web application that supports:
- Multi-user access with authentication
- Concur to Foundation expense conversion
- Bank reconciliation with intelligent matching
- Processing history and audit trails
- Scalable architecture for future growth

## 🛠 Tech Stack

- **Frontend**: Astro 5 + React + TailwindCSS
- **Backend**: Go (Gin framework)
- **Database**: PostgreSQL
- **File Processing**: Pure Go (excelize for Excel)

## 🚀 Quick Start

### Prerequisites
- Docker Desktop
- Go 1.21+
- Node.js 18+

### 1. Initial Setup

```bash
# Clone and navigate to the project
cd jbs-internal-portal

# Run setup script (starts PostgreSQL, creates .env)
chmod +x setup.sh
./setup.sh
```

### 2. Start the Backend

```bash
cd backend/cmd/server
go run main.go
```

Backend runs at `http://localhost:8080`

### 3. Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

Frontend runs at `http://localhost:4321`

### 4. Login

- **Email**: `admin@jbs.com`
- **Password**: `password123`

## 📁 Project Structure

```
jbs-internal-portal/
├── backend/
│   ├── cmd/server/          # Application entry point
│   ├── internal/
│   │   ├── handlers/        # HTTP request handlers
│   │   ├── services/        # Business logic
│   │   │   └── concur_converter.go  # Pure Go conversion
│   │   ├── models/          # Data structures
│   │   ├── database/        # DB connection & migrations
│   │   ├── auth/            # JWT authentication
│   │   └── middleware/      # Auth & security
│   ├── uploads/             # Temporary file storage
│   └── go.mod
│
├── frontend/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── index.astro         # Login page
│   │   │   ├── dashboard.astro     # Main dashboard
│   │   │   ├── concur/
│   │   │   │   └── index.astro     # Concur converter
│   │   │   └── reconciliation/
│   │   │       └── index.astro     # Reconciliation tool
│   │   ├── layouts/
│   │   │   └── Layout.astro        # Main layout with nav
│   │   └── styles/
│   │       └── global.css
│   └── package.json
│
└── docker-compose.yml       # PostgreSQL setup
```

## 🔐 API Endpoints

### Authentication
- `POST /api/login` - User login
- `GET /api/user` - Get current user (authenticated)

### Concur Conversion
- `POST /api/concur/upload` - Upload and convert Concur file
- `GET /api/concur/history` - Get conversion history
- `GET /api/download/:id` - Download converted CSV

### Reconciliation (Coming Soon)
- `POST /api/reconciliation/upload` - Upload bank & Foundation files
- `GET /api/reconciliation/history` - Get reconciliation history

## 📊 Database Schema

```sql
users (
  id, email, password, name, role, created_at, updated_at
)

conversion_jobs (
  id, user_id, filename, status, vendor_id, 
  rows_processed, rows_skipped, output_file_path, 
  error_log, created_at, completed_at
)

reconciliation_jobs (
  id, user_id, bank_filename, foundation_filename, 
  status, tolerance_days, matched_count, void_count,
  ambiguous_void_count, output_file_path, 
  created_at, completed_at
)

vendor_configs (
  id, vendor_name, vendor_id, is_default, created_at
)
```

## 🔄 Migration from Python Desktop App

### What's Different

**Old (Python Desktop)**:
- Tkinter GUI
- Local file processing
- Windows .exe distribution
- Single user

**New (Go Web App)**:
- Web browser interface
- Server-side processing
- Access from anywhere
- Multi-user with auth
- Processing history
- Role-based permissions

### Conversion Logic

The Concur-to-Foundation conversion has been **completely rewritten in pure Go**:
- Uses `excelize/v2` for Excel reading
- CSV generation with Go's `encoding/csv`
- Same validation rules as Python version
- Same 48-column Foundation template
- Multi-format date parsing
- Cross-platform compatibility

## 🧪 Testing

### Test Concur Conversion

1. Login to portal
2. Navigate to "Concur Converter"
3. Upload a Concur export (.xlsx)
4. Set vendor ID (default: 138)
5. Click "Convert to Foundation Format"
6. Download the resulting CSV

### Verify Output

The output CSV should:
- Have 12 header rows (Foundation template)
- One data row per expense
- Dates formatted as M/D/YYYY
- Vendor ID in column 3
- Amount in columns 6, 15, and 30

## 🚧 Next Steps

1. **Reconciliation Engine** (Priority)
   - Port reconciliation logic to Go
   - Excel report generation
   - Void detection
   - Ambiguous void handling

2. **Enhanced Features**
   - Batch file processing
   - Scheduled imports
   - Email notifications
   - Advanced reporting dashboard

3. **Deployment**
   - Railway.app setup
   - Custom domain
   - Production environment
   - Backup strategy

## 💻 Development

### Add a New User

```bash
# Connect to database
docker exec -it jbs-postgres psql -U jbs_user -d jbs_portal

# Create user (password will be hashed)
INSERT INTO users (email, password, name, role)
VALUES ('user@jbs.com', '$2a$14$...', 'User Name', 'employee');
```

### Environment Variables

```bash
DATABASE_URL=postgresql://jbs_user:jbs_password@localhost:5432/jbs_portal?sslmode=disable
JWT_SECRET=your-secret-key-change-in-production
GIN_MODE=debug
PORT=8080
ALLOWED_ORIGINS=http://localhost:4321
UPLOAD_MAX_SIZE=10485760
UPLOAD_DIR=./uploads
```

## 📝 Notes

- The desktop `jbs-concur-converter` remains available for users who prefer it
- Both applications produce identical Foundation CSV output
- Web version adds user tracking, history, and collaboration features
- Migration can be gradual - no need to force immediate switch

## 🆘 Support

For issues or questions, check:
1. Database is running: `docker ps`
2. Backend logs for errors
3. Browser console for frontend errors
4. JWT_SECRET is set in .env

## 📜 License

Internal JBS use only.
