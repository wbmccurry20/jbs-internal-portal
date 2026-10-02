# Testing guide

This project keeps tests local-first. The backend must run in a clean dev environment without production credentials, and any DB-backed test must opt in explicitly.

## Backend

Run the default suite:

```bash
cd backend
go test ./...
```

This must pass on a clean machine without Railway secrets or a production `DATABASE_URL`.

For a database-backed run, start local Postgres and opt in with a test variable:

```bash
docker compose up -d postgres
export TEST_DATABASE_URL="postgresql://jbs_user:jbs_password@localhost:5433/jbs_portal?sslmode=disable"
cd backend && go test ./...
```

Rules:
- Unit tests must not require the production `DATABASE_URL`.
- Integration tests that need Postgres should skip when `TEST_DATABASE_URL` is unset.
- No test may write to production data.
- Coverage reports are optional and not a release gate.

## Frontend

If the frontend test script is present, run it from the repo root:

```bash
cd frontend
npm test -- --run
```

The project should not ship a broken test script that only works in a watch terminal.

## A short checklist for contributors

- Do not hardcode developer paths or machine-specific env files.
- Do not depend on production secrets in unit tests.
- Prefer skip logic in tests that require Postgres.
- Keep test commands simple enough for a fresh clone to run.
- If a test expects a local DB, require `TEST_DATABASE_URL` or Docker instead of a hidden laptop setup.

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
