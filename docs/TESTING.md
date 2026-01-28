# Testing Guide - JBS Internal Portal

## Overview
This document explains how to run and maintain the test suite for the JBS Internal Portal. The test suite covers authentication, reconciliation logic, file handling, and API endpoints.

## Quick Start

### Backend Tests
```bash
cd backend
go test ./...
```

**Test Coverage Goals**:
- Auth module: 95%+ ✅
- Handlers: 80%+
- Services (reconciliation): 90%+
- Middleware: 80%+
- Utils: 80%+ ✅

### Run Specific Test Package
```bash
# Auth tests
go test -v ./internal/auth/...

# Handler tests
go test -v ./internal/handlers/...

# Service tests
go test -v ./internal/services/...

# Utils tests
go test -v ./internal/utils/...

# Middleware tests
go test -v ./internal/middleware/...
```

### Run with Coverage
```bash
# All tests with coverage
go test -cover ./...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# View coverage by package
go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out
```

---

## Test Structure

```
backend/
├── internal/
│   ├── auth/
│   │   ├── jwt.go
│   │   └── jwt_test.go              # ✅ JWT & password hashing tests
│   ├── handlers/
│   │   ├── auth_handler.go
│   │   ├── auth_handler_test.go     # ✅ Authentication endpoint tests
│   │   ├── reconciliation_handler.go
│   │   └── health_test.go           # ✅ Health endpoint tests
│   ├── middleware/
│   │   ├── middleware.go
│   │   └── middleware_test.go       # ✅ Auth middleware tests
│   ├── services/
│   │   ├── reconciliation.go
│   │   └── reconciliation_test.go   # ✅ Reconciliation logic tests
│   └── utils/
│       ├── file_validation.go
│       └── file_validation_test.go  # ✅ File validation tests
```

---

## Test Categories

### 1. Unit Tests

**Authentication & Security**
- ✅ Password hashing and verification
- ✅ JWT token generation
- ✅ JWT token validation
- ✅ Token expiration handling
- ✅ 'None' algorithm attack prevention
- ✅ Different user tokens
- ✅ Special characters in email

**File Validation**
- ✅ Excel file validation (.xlsx, .xls)
- ✅ File size limits (10MB)
- ✅ Invalid file extensions
- ✅ Path traversal prevention
- ✅ Filename sanitization

**Reconciliation Engine**
- ✅ Exact amount/date matching
- ✅ Date tolerance (±N days)
- ✅ Amount tolerance (±%)
- ✅ Void transaction detection
- ✅ Match rate calculation
- ✅ Empty input handling
- ✅ Multiple matches

### 2. Integration Tests

**Authentication Endpoints**
- Login with valid credentials
- Login with invalid password
- Login with invalid JSON
- User not found
- Get current user
- Unauthorized access

**Reconciliation Endpoints**
- File upload
- Processing jobs
- Job history
- Download results
- Error handling

### 3. Security Tests

**Input Validation**
- SQL injection prevention
- Path traversal prevention
- File type validation
- Size limit enforcement

**Authentication**
- Token expiration
- Invalid token handling
- Missing JWT secret
- Algorithm validation

---

## Running Tests

### All Tests
```bash
cd backend
go test ./...
```

### Verbose Output
```bash
go test -v ./...
```

### Specific Test
```bash
go test -v -run TestGenerateToken ./internal/auth/...
go test -v -run TestReconciliationEngine_ExactMatch ./internal/services/...
```

### With Race Detection
```bash
go test -race ./...
```

### Benchmark Tests
```bash
go test -bench=. ./...
```

---

## Test Database

For integration tests that require a database:

**Option 1: In-Memory SQLite (Fast)**
```go
database.DB, _ = sql.Open("sqlite3", ":memory:")
```

**Option 2: Test PostgreSQL Container**
```bash
# Using Docker
docker run -d \
  --name jbs-test-db \
  -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=jbs_test \
  -p 5433:5432 \
  postgres:15-alpine

# Run tests with test database
export DATABASE_URL="postgresql://postgres:test@localhost:5433/jbs_test?sslmode=disable"
go test ./...

# Cleanup
docker stop jbs-test-db && docker rm jbs-test-db
```

**Option 3: Mocking**
For unit tests, we mock database calls to avoid external dependencies.

---

## Writing New Tests

### Test Template
```go
package yourpackage

import "testing"

func TestYourFunction(t *testing.T) {
	// Arrange
	input := "test data"
	expected := "expected result"
	
	// Act
	result := YourFunction(input)
	
	// Assert
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}
```

### Table-Driven Tests
```go
func TestValidation(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		shouldError bool
	}{
		{"Valid input", "test@jbs.com", false},
		{"Invalid input", "not-an-email", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.input)
			if (err != nil) != tt.shouldError {
				t.Errorf("Unexpected error state")
			}
		})
	}
}
```

---

## Continuous Integration

### GitHub Actions (Recommended)
```yaml
name: Tests
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run tests
        run: |
          cd backend
          go test -v -race -coverprofile=coverage.out ./...
      - name: Upload coverage
        uses: codecov/codecov-action@v3
        with:
          files: ./backend/coverage.out
```

---

## Coverage Goals

| Package | Current | Target | Status |
|---------|---------|--------|--------|
| auth | 95% | 95% | ✅ |
| utils | 51.7% | 80% | 🟡 |
| middleware | 26.3% | 80% | 🔴 |
| handlers | 0.5% | 80% | 🔴 |
| services | 80%+ | 90% | 🟡 |

---

## Best Practices

1. **Always test edge cases**
   - Empty inputs
   - Nil values
   - Maximum sizes
   - Invalid formats

2. **Test security features**
   - SQL injection
   - Path traversal
   - Authentication bypass
   - Token manipulation

3. **Use table-driven tests**
   - Easier to add new cases
   - Clearer test intent
   - Better organization

4. **Mock external dependencies**
   - Database calls
   - File system operations
   - External APIs
   - Time-dependent operations

5. **Clean up resources**
   - Close database connections
   - Delete test files
   - Reset environment variables
   - Use defer for cleanup

---

## Debugging Failed Tests

### Verbose Output
```bash
go test -v ./internal/auth/... 2>&1 | less
```

### Run Single Test
```bash
go test -v -run TestSpecificFunction ./...
```

### Print Debug Info
```go
func TestSomething(t *testing.T) {
	t.Logf("Debug info: %+v", someValue)
	// ... rest of test
}
```

### Check for Race Conditions
```bash
go test -race ./...
```

---

## Adding Tests for New Features

When adding a new feature:

1. **Write tests first** (TDD approach)
2. **Test happy path** - normal operation
3. **Test error cases** - what can go wrong?
4. **Test edge cases** - boundary conditions
5. **Test security** - injection, validation, auth
6. **Update this documentation**

Example workflow:
```bash
# 1. Create test file
touch internal/handlers/new_feature_test.go

# 2. Write failing tests
go test -v ./internal/handlers/...

# 3. Implement feature
# Edit new_feature.go

# 4. Tests should pass
go test -v ./internal/handlers/...

# 5. Check coverage
go test -cover ./internal/handlers/...
```

---

## Common Test Scenarios

### Testing HTTP Handlers
```go
func TestHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()
	router.GET("/endpoint", YourHandler)
	
	req, _ := http.NewRequest("GET", "/endpoint", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}
```

### Testing with Authentication
```go
func TestAuthProtected(t *testing.T) {
	router := gin.Default()
	router.GET("/protected", func(c *gin.Context) {
		c.Set("userID", 1)
		YourProtectedHandler(c)
	})
	// ... test
}
```

### Testing File Operations
```go
func TestFileProcessing(t *testing.T) {
	// Create temp file
	tmpFile, _ := os.CreateTemp("", "test-*.xlsx")
	defer os.Remove(tmpFile.Name())
	
	// ... test file processing
}
```

---

## Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Gin Testing Guide](https://github.com/gin-gonic/gin#testing)
- [Table-Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Go Test Coverage](https://blog.golang.org/cover)

---

## Getting Help

If tests are failing:
1. Check the error message carefully
2. Run with `-v` for verbose output
3. Check if environment variables are set
4. Ensure test database is accessible
5. Review recent code changes
6. Check the test documentation above

For questions or issues, contact the development team.
