package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("✓ Connected to database")

	// Create owner accounts
	owners := []struct {
		Email    string
		Name     string
		Password string
	}{
		{"kelsey@jbsconstructiongroup.com", "Kelsey", "JBSOwner2026!"},
		{"joe@jbsconstructiongroup.com", "Joe", "JBSOwner2026!"},
		{"alex@jbsconstructiongroup.com", "Alex", "JBSOwner2026!"},
		{"kevin@jbsconstructiongroup.com", "Kevin", "JBSOwner2026!"},
	}

	for _, owner := range owners {
		// Check if user exists
		var exists bool
		err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", owner.Email).Scan(&exists)
		if err != nil {
			log.Printf("❌ Error checking %s: %v", owner.Email, err)
			continue
		}

		if exists {
			log.Printf("  ⚠️  User already exists: %s", owner.Email)
			continue
		}

		// Hash password
		hash, err := bcrypt.GenerateFromPassword([]byte(owner.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("❌ Failed to hash password for %s: %v", owner.Email, err)
			continue
		}

		// Insert user
		_, err = db.Exec(
			"INSERT INTO users (email, password, name, role) VALUES ($1, $2, $3, $4)",
			owner.Email, string(hash), owner.Name, "owner",
		)
		if err != nil {
			log.Printf("❌ Failed to create %s: %v", owner.Email, err)
			continue
		}

		log.Printf("  ✅ Created owner account: %s (%s)", owner.Name, owner.Email)
	}

	log.Println("\n✅ All owner accounts created!")
	fmt.Println("\nTemporary password for all owners: JBSOwner2026!")
	fmt.Println("They should change this on first login.")
}
