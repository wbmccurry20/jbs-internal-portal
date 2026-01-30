#!/bin/bash
# Test Job API endpoints

BASE_URL="http://localhost:8080/api"

echo "=== Testing Job API Endpoints ==="
echo ""

# 1. Login to get token
echo "1. Getting auth token..."
LOGIN_RESPONSE=$(curl -s -X POST "$BASE_URL/login" \
  -H "Content-Type: application/json" \
  -d '{"email":"hello@itwill.dev","password":"your-secure-password-here"}')

TOKEN=$(echo $LOGIN_RESPONSE | grep -o '"token":"[^"]*' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
  echo "❌ Failed to get auth token"
  echo "Response: $LOGIN_RESPONSE"
  exit 1
fi

echo "✅ Got token: ${TOKEN:0:20}..."
echo ""

# 2. List all jobs
echo "2. Listing all jobs..."
curl -s -X GET "$BASE_URL/jobs" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool | head -50
echo ""

# 3. Get first job details
echo "3. Getting job #1 details..."
curl -s -X GET "$BASE_URL/jobs/1" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool
echo ""

# 4. Search jobs by city
echo "4. Searching jobs by 'Miami'..."
curl -s -X GET "$BASE_URL/jobs?search=Miami" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool | head -30
echo ""

# 5. Filter by status
echo "5. Filtering by status='In Progress'..."
curl -s -X GET "$BASE_URL/jobs?status=In%20Progress" \
  -H "Authorization: Bearer $TOKEN" | python3 -m json.tool | head -30
echo ""

echo "=== Test Complete ==="
