# JBS Internal Portal - Quick Reference

## Starting the Application

### 1. Start Database
```bash
docker-compose up -d postgres
```

### 2. Start Backend (Terminal 1)
```bash
cd backend/cmd/server
go run main.go
```

### 3. Start Frontend (Terminal 2)
```bash
cd frontend
npm run dev
```

### 4. Access
- Frontend: http://localhost:4321
- Backend API: http://localhost:8080
- Login: `admin@jbs.com` / `password123`

## Common Tasks

### Add a New User
```sql
-- Connect to database
docker exec -it jbs-postgres psql -U jbs_user -d jbs_portal

-- Hash password first with Go:
-- cd backend && go run -c "import auth; print(auth.HashPassword('yourpassword'))"

INSERT INTO users (email, password, name, role)
VALUES ('newuser@jbs.com', '$2a$14$...hashed...', 'Full Name', 'employee');
```

### Check Database
```bash
docker exec -it jbs-postgres psql -U jbs_user -d jbs_portal

# List tables
\dt

# View users
SELECT * FROM users;

# View recent conversions
SELECT * FROM conversion_jobs ORDER BY created_at DESC LIMIT 10;
```

### View Logs
```bash
# Database logs
docker logs jbs-postgres

# Backend logs (in terminal running Go server)

# Frontend logs (in terminal running npm dev)
```

### Reset Database
```bash
docker-compose down -v
docker-compose up -d postgres
# Then restart backend to run migrations
```

## API Testing with cURL

### Login
```bash
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@jbs.com","password":"password123"}'
```

### Upload Concur File
```bash
TOKEN="your-jwt-token"

curl -X POST http://localhost:8080/api/concur/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@/path/to/concur.xlsx" \
  -F "vendor_id=138"
```

### Get History
```bash
curl http://localhost:8080/api/concur/history \
  -H "Authorization: Bearer $TOKEN"
```

## Troubleshooting

### "Database connection failed"
- Check Docker: `docker ps`
- Restart: `docker-compose down && docker-compose up -d`

### "Invalid token" on API calls
- Token expires after 24 hours
- Login again to get new token

### Frontend can't reach backend
- Check backend is running on port 8080
- Check CORS settings in backend/cmd/server/main.go

### File upload fails
- Check uploads directory exists: `mkdir -p backend/uploads`
- Check file size (max 10MB by default)

## File Locations

- **Uploaded files**: `backend/uploads/`
- **Database data**: Docker volume `jbs_portal_postgres_data`
- **Config**: `backend/.env`
- **Logs**: Terminal output

## Next Phase: Reconciliation

To add reconciliation:
1. Create `backend/internal/services/reconciliation.go`
2. Port logic from `jbs-concur-converter/src/core/reconciliation.py`
3. Create `backend/internal/handlers/reconciliation_handler.go`
4. Add routes in `backend/cmd/server/main.go`
5. Build frontend in `frontend/src/pages/reconciliation/`
