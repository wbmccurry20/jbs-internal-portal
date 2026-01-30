#!/bin/bash
# Manual deployment script for Railway
# Usage: ./deploy.sh [backend|frontend|both]

set -e

SERVICE=${1:-both}

echo "🚀 Deploying JBS Internal Portal to Railway..."

if [ "$SERVICE" = "backend" ] || [ "$SERVICE" = "both" ]; then
    echo "📦 Deploying backend..."
    railway up --service backend
fi

if [ "$SERVICE" = "frontend" ] || [ "$SERVICE" = "both" ]; then
    echo "🎨 Deploying frontend..."
    railway up --service frontend
fi

echo "✅ Deployment complete!"
echo "🔗 Frontend: https://portal.buildwithjbs.com"
echo "🔗 Backend: https://jbs-internal-portal-production.up.railway.app"
