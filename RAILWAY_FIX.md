# Railway Quick Fix - JBS Internal Portal

## The Issue
Railway was trying to build from the root directory and detecting npm workspace commands that the npm version doesn't support.

## The Solution

### For Frontend Service:

1. **In Railway Dashboard**:
   - Go to your frontend service
   - Settings → General → **Root Directory**
   - Set to: `frontend`
   - Save changes

2. **Redeploy**:
   - Click "Deploy" or it will auto-deploy

### Configuration Files (Already Updated)

✅ `frontend/railway.toml` - Uses `npm install --legacy-peer-deps`
✅ `frontend/nixpacks.toml` - Specifies Node 20
✅ `frontend/.node-version` - Pins Node to 20.11.0

### For Backend Service:

No changes needed - backend configuration is correct:
- Root Directory: `backend`
- Uses Go build (no npm issues)

## Verification

After deploying, you should see:
```
✓ Building from directory: frontend/
✓ Installing dependencies with npm install --legacy-peer-deps
✓ Building with npm run build
✓ Starting with npm run start
```

## Common Errors Fixed

❌ `npm error [--include <prod|dev|optional|peer>...]` 
✅ Fixed by setting Root Directory to `frontend`

❌ `npm ci did not complete successfully`
✅ Fixed by using `npm install --legacy-peer-deps` instead

## Next Steps

1. Set Root Directory to `frontend` in Railway
2. Redeploy
3. Check deployment logs
4. Update `PUBLIC_API_URL` environment variable with your backend URL
