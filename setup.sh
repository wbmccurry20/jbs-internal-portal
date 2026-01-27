#!/bin/bash

echo "🚀 JBS Internal Portal - Setup Script"
echo "======================================"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Docker is not running. Please start Docker Desktop and try again."
    exit 1
fi

# Start PostgreSQL
echo ""
echo "📦 Starting PostgreSQL..."
docker-compose up -d postgres

# Wait for PostgreSQL to be ready
echo "⏳ Waiting for PostgreSQL to be ready..."
sleep 5

# Copy .env.example to .env if it doesn't exist
if [ ! -f backend/.env ]; then
    echo "📝 Creating backend/.env file..."
    cp backend/.env.example backend/.env
    echo "✅ Created backend/.env - Update JWT_SECRET before deploying to production!"
fi

# Install Go dependencies
echo ""
echo "📦 Installing Go dependencies..."
cd backend
go mod download
go mod tidy
cd ..

# Create uploads directory
mkdir -p backend/uploads

# Create a test user (password: password123)
echo ""
echo "👤 Creating test user..."
PGPASSWORD=jbs_password psql -h localhost -U jbs_user -d jbs_portal -c "
INSERT INTO users (email, password, name, role)
VALUES (
    'admin@jbs.com',
    '\$2a\$14\$XZEj7VxKzC0hMVVvXrJ9.eTqzPk5LxvLvLvXuVsYGzE9TvBL8LPqG',
    'Admin User',
    'admin'
)
ON CONFLICT (email) DO NOTHING;
" 2>/dev/null || echo "Note: Run 'cd backend && go run cmd/server/main.go' first to create tables"

echo ""
echo "✅ Setup complete!"
echo ""
echo "Next steps:"
echo "1. Start the backend:"
echo "   cd backend/cmd/server"
echo "   go run main.go"
echo ""
echo "2. Start the frontend (in a new terminal):"
echo "   cd frontend"
echo "   npm install"
echo "   npm run dev"
echo ""
echo "3. Login with:"
echo "   Email: admin@jbs.com"
echo "   Password: password123"
echo ""
echo "4. Access the portal at: http://localhost:4321"
echo ""
