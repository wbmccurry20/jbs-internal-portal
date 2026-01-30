# Deployment Guide

## Production Deployment

### Prerequisites
- Railway CLI installed: `npm install -g @railway/cli`
- Logged into Railway: `railway login`

### Recommended: Disable Auto-Deploy
1. Go to Railway dashboard
2. For each service (backend, frontend):
   - Settings → Auto Deploy → Toggle OFF
3. This prevents production outages from every git push

### Manual Deployment

Deploy both services:
```bash
./deploy.sh both
```

Deploy backend only:
```bash
./deploy.sh backend
```

Deploy frontend only:
```bash
./deploy.sh frontend
```

### Current Configuration

**Backend** ([railway.toml](railway.toml)):
- Healthcheck: `/health` endpoint
- Restart policy: On failure, max 10 retries
- Watch pattern: `backend/**` (only rebuilds on backend changes)

**Frontend** ([railway.frontend.toml](railway.frontend.toml)):
- Node/Astro SSR build
- Should watch `frontend/**` only

### Troubleshooting

**502 Errors**: Service crashed during startup
- Check Railway logs for errors
- Verify environment variables are set
- Manually redeploy from Railway dashboard

**Slow deploys**: Build taking too long
- Check Railway build logs
- May need to optimize dependencies

**Database migrations**: Run automatically on backend startup
- Check logs to ensure migrations complete
- New migrations added in `backend/internal/database/migrations/`
