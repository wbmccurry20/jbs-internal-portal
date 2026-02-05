# JBS Internal Portal - Railway Deployment Guide

## Prerequisites
- Railway account (https://railway.app)
- GitHub repository for this project

## Deployment Steps

### 1. Database Setup (PostgreSQL)

1. Go to Railway dashboard
2. Click "New Project" → "Provision PostgreSQL"
3. Name it: `jbs-postgres`
4. Railway will auto-generate:
   - `DATABASE_URL` (copy this for backend)
   - Connection credentials

### 2. Backend Deployment (Go API)

1. Click "New Service" → "GitHub Repo"
2. Select your `jbs-internal-portal` repository
3. Configure:
   - **Root Directory**: `backend`
   - **Build Command**: (Railway auto-detects Go)
   - **Start Command**: `./cmd/server/main`

4. **Environment Variables** (Settings → Variables):
   ```
   DATABASE_URL=${{Postgres.DATABASE_URL}}
   JWT_SECRET=<generate-strong-secret-here>
   GIN_MODE=release
   PORT=8080
   ALLOWED_ORIGINS=https://your-frontend-url.railway.app
   UPLOAD_MAX_SIZE=10485760
   UPLOAD_DIR=/app/uploads
   
   # Owner Email
   OWNER_EMAIL=emily.simpson@jbsconstructiongroup.com
   
   # Support Backdoor (IT Will Access)
   SUPPORT_EMAIL=hello@itwill.dev
   SUPPORT_PASSWORD=<your-secure-password>
   SUPPORT_NAME=Will McCurry
   ```

5. **Connect Database**:
   - In Variables tab, click "+" → "Reference" → Select PostgreSQL service
   - Variable: `DATABASE_URL`

6. Click "Deploy" - backend will be available at: `https://jbs-backend-XXXX.railway.app`

### 3. Frontend Deployment (Astro)

1. Click "New Service" → "GitHub Repo"
2. Select same repository
3. **CRITICAL**: Configure Root Directory:
   - Go to Settings → General
   - Set **Root Directory**: `frontend`
   - This tells Railway to build from the frontend folder only

4. The build will use the `railway.toml` file in the frontend directory automatically

5. **Environment Variables** (Settings → Variables):
   ```
   PUBLIC_API_URL=https://jbs-backend-XXXX.railway.app/api
   NODE_ENV=production
   ```

6. Update `frontend/src/lib/api.ts`:
   ```typescript
   const API_BASE_URL = import.meta.env.PUBLIC_API_URL || 'http://localhost:8080/api';
   ```

7. Click "Deploy" - frontend will be available at: `https://jbs-portal-XXXX.railway.app`

**Important**: If you get npm workspace errors, ensure:
- Root Directory is set to `frontend` (not root)
- Node version is 20+ (specified in `.node-version`)
- Using `npm install --legacy-peer-deps` in build command

### 4. Update CORS Settings

After frontend deploys, update backend `ALLOWED_ORIGINS`:
```
ALLOWED_ORIGINS=https://jbs-portal-XXXX.railway.app,https://portal.jbsconstructiongroup.com
```

### 5. Custom Domain (Optional)

**Frontend**:
1. Settings → Networking → Custom Domain
2. Add: `portal.jbsconstructiongroup.com`
3. Update DNS:
   - Type: CNAME
   - Name: `portal`
   - Value: `jbs-portal-XXXX.railway.app`

**Backend**:
1. Settings → Networking → Custom Domain
2. Add: `api.jbsconstructiongroup.com`
3. Update DNS:
   - Type: CNAME
   - Name: `api`
   - Value: `jbs-backend-XXXX.railway.app`

## Post-Deployment

### Test Endpoints

```bash
# Backend health check
curl https://jbs-backend-XXXX.railway.app/health

# Test login
curl -X POST https://jbs-backend-XXXX.railway.app/api/login \
  -H "Content-Type: application/json" \
  -d '{"email":"emily.simpson@jbsconstructiongroup.com","password":"password123"}'
```

### User Accounts

After first deployment, 3 users will be automatically created:

1. **emily.simpson@jbsconstructiongroup.com** / password123
2. **shelby@jbsconstructiongroup.com** / password123
3. **hello@itwill.dev** / (from SUPPORT_PASSWORD env var) - Backdoor access

### Database Migrations

Migrations run automatically on startup. Check logs:
```
✅ Connected to database
✅ Database migrations completed
✓ User already exists: emily.simpson@jbsconstructiongroup.com
✓ Support account exists: hello@itwill.dev
```

## Monitoring

- **Logs**: Railway dashboard → Service → Deployments tab
- **Metrics**: CPU, Memory, Network usage visible in dashboard
- **Health Check**: Backend `/health` endpoint

## Troubleshooting

### NPM Workspace/Install Errors (Frontend)
**Error**: `npm error [--include <prod|dev|optional|peer> [--include <prod|dev|optional|peer> ...]]`

**Cause**: Railway is building from root directory instead of frontend directory, or npm version mismatch.

**Fix**:
1. Go to Railway service Settings → General
2. Set **Root Directory** to `frontend` (NOT root or empty)
3. Redeploy the service
4. If still failing, check that `frontend/.node-version` exists with `20.11.0`

### Backend won't start
- Check `DATABASE_URL` is connected to PostgreSQL service
- Verify all required env vars are set
- Check logs for migration errors

### Frontend can't connect to backend
- Verify `API_BASE_URL` points to backend Railway URL
- Check backend `ALLOWED_ORIGINS` includes frontend URL
- Ensure backend is deployed and running

### Database connection issues
- Verify PostgreSQL service is running
- Check `DATABASE_URL` format matches PostgreSQL connection string
- Review backend logs for connection errors

## Cost Estimate

Railway Hobby Plan ($5/month):
- PostgreSQL: ~$5/month
- Backend: ~$5/month  
- Frontend: ~$5/month

Total: ~$15/month

## Security Notes

1. **Never commit** `.env` file to git
2. **Rotate** JWT_SECRET in production
3. **Change** default user passwords after first login
4. **Secure** SUPPORT_PASSWORD - don't use default
5. **Limit** ALLOWED_ORIGINS to your actual domains

## Updates & Redeployment

Railway auto-deploys on git push to main branch. To manually redeploy:
1. Go to service in Railway
2. Click "Deploy" → "Redeploy"
