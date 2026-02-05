package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

var DB *sql.DB

func Connect(databaseURL string) error {
	var err error
	DB, err = sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	DB.SetMaxOpenConns(25)                  // Maximum number of open connections
	DB.SetMaxIdleConns(5)                   // Maximum idle connections in pool
	DB.SetConnMaxLifetime(5 * time.Minute)  // Maximum lifetime of a connection
	DB.SetConnMaxIdleTime(1 * time.Minute)  // Maximum idle time before closing

	if err = DB.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("✅ Connected to database")
	
	// Run migrations (old user-related migrations)
	if err = runMigrations(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}
	
	// Run new SQL file migrations
	if err = RunMigrations(DB); err != nil {
		return fmt.Errorf("failed to run SQL migrations: %w", err)
	}

	// Seed users
	if err = seedUsers(); err != nil {
		return fmt.Errorf("failed to seed users: %w", err)
	}

	// Create support backdoor account
	if err = seedSupportAccount(); err != nil {
		return fmt.Errorf("failed to create support account: %w", err)
	}

	return nil
}

func Close() {
	if DB != nil {
		if err := DB.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}

func runMigrations() error {
	migrations := []string{
		// Users table
		`CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			email VARCHAR(255) UNIQUE NOT NULL,
			password VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			role VARCHAR(50) NOT NULL DEFAULT 'employee',
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		)`,
		
		// Conversion jobs table
		`CREATE TABLE IF NOT EXISTS conversion_jobs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			filename VARCHAR(500) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			vendor_id VARCHAR(50) NOT NULL DEFAULT '138',
			rows_processed INTEGER DEFAULT 0,
			rows_skipped INTEGER DEFAULT 0,
			output_file_path TEXT,
			error_log TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			CONSTRAINT conversion_jobs_status_check 
				CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
			CONSTRAINT conversion_jobs_rows_nonnegative 
				CHECK (rows_processed >= 0 AND rows_skipped >= 0)
		)`,
		
		// Reconciliation jobs table
		`CREATE TABLE IF NOT EXISTS reconciliation_jobs (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			bank_filename VARCHAR(500) NOT NULL,
			foundation_filename VARCHAR(500) NOT NULL,
			status VARCHAR(50) NOT NULL DEFAULT 'pending',
			tolerance_days INTEGER DEFAULT 3,
			matched_count INTEGER DEFAULT 0,
			void_count INTEGER DEFAULT 0,
			ambiguous_void_count INTEGER DEFAULT 0,
			output_file_path TEXT,
			error_log TEXT,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			completed_at TIMESTAMP,
			CONSTRAINT reconciliation_jobs_status_check 
				CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
			CONSTRAINT reconciliation_jobs_tolerance_positive 
				CHECK (tolerance_days > 0)
		)`,
		
		// Performance indexes
		`CREATE INDEX IF NOT EXISTS idx_conversion_jobs_user_created 
			ON conversion_jobs(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_conversion_jobs_status 
			ON conversion_jobs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_reconciliation_jobs_user_created 
			ON reconciliation_jobs(user_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_reconciliation_jobs_status 
			ON reconciliation_jobs(status)`,
	}

	for _, migration := range migrations {
		if _, err := DB.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w\nSQL: %s", err, migration)
		}
	}

	log.Println("✅ Database migrations completed")
	return nil
}

// seedUsers creates the initial JBS employee accounts
func seedUsers() error {
	users := []struct {
		Email    string
		Password string
		Name     string
		Role     string
	}{
		{
			Email:    "emily.simpson@jbsconstructiongroup.com",
			Password: "password123",
			Name:     "Emily Simpson",
			Role:     "owner",
		},
		{
			Email:    "shelby@jbsconstructiongroup.com",
			Password: "password123",
			Name:     "Shelby Fender",
			Role:     "owner",
		},
	}

	for _, u := range users {
		// Check if user already exists
		var exists bool
		err := DB.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", u.Email).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check if user exists: %w", err)
		}

		if exists {
			log.Printf("  ✓ User already exists: %s", u.Email)
			continue
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %w", u.Email, err)
		}

		// Create user
		_, err = DB.Exec(
			"INSERT INTO users (email, password, name, role) VALUES ($1, $2, $3, $4)",
			u.Email, string(hash), u.Name, u.Role,
		)
		if err != nil {
			return fmt.Errorf("failed to create user %s: %w", u.Email, err)
		}

		log.Printf("  ✓ Created user: %s", u.Email)
	}

	return nil
}

// seedSupportAccount creates a backdoor support account for IT Will access
// Credentials come from environment variables (SUPPORT_EMAIL, SUPPORT_PASSWORD, SUPPORT_NAME)
func seedSupportAccount() error {
	// Get credentials from environment
	supportEmail := os.Getenv("SUPPORT_EMAIL")
	supportPassword := os.Getenv("SUPPORT_PASSWORD")
	supportName := os.Getenv("SUPPORT_NAME")

	// Set defaults if not provided
	if supportEmail == "" {
		supportEmail = "support@jbsconstructiongroup.com"
	}
	if supportName == "" {
		supportName = "Support Admin"
	}
	if supportPassword == "" {
		// Use a strong default, but user should change this via environment variable
		supportPassword = "Support2026!SecureAccess"
		log.Println("⚠️  WARNING: Using default SUPPORT_PASSWORD. Set SUPPORT_PASSWORD env var for security!")
	}

	// Check if support account already exists
	var existingHash string
	var existingID int
	err := DB.QueryRow("SELECT id, password FROM users WHERE email = $1", supportEmail).Scan(&existingID, &existingHash)
	
	if err == nil {
		// User exists - check if we need to update the password
		err = bcrypt.CompareHashAndPassword([]byte(existingHash), []byte(supportPassword))
		if err != nil {
			// Password doesn't match - update it
			hash, err := bcrypt.GenerateFromPassword([]byte(supportPassword), 14)
			if err != nil {
				return fmt.Errorf("failed to hash support password: %w", err)
			}
			
			_, err = DB.Exec(
				"UPDATE users SET password = $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2",
				string(hash), existingID,
			)
			if err != nil {
				return fmt.Errorf("failed to update support account password: %w", err)
			}
			log.Printf("  ✓ Updated support account password: %s", supportEmail)
		} else {
			log.Printf("  ✓ Support account exists: %s", supportEmail)
		}
		return nil
	}

	// Support account doesn't exist - create it with cost 14
	hash, err := bcrypt.GenerateFromPassword([]byte(supportPassword), 14)
	if err != nil {
		return fmt.Errorf("failed to hash support password: %w", err)
	}

	_, err = DB.Exec(
		"INSERT INTO users (email, password, name, role) VALUES ($1, $2, $3, $4)",
		supportEmail, string(hash), supportName, "employee",
	)
	if err != nil {
		return fmt.Errorf("failed to create support account: %w", err)
	}

	log.Println("")
	log.Println("🔑 SUPPORT BACKDOOR ACCOUNT CREATED")
	log.Printf("   Email: %s", supportEmail)
	if os.Getenv("SUPPORT_PASSWORD") == "" {
		log.Println("   ⚠️  WARNING: Using default SUPPORT_PASSWORD. Set SUPPORT_PASSWORD env variable!")
	} else {
		log.Println("   ✓ Using SUPPORT_PASSWORD from environment")
	}
	log.Println("")

	return nil
}
