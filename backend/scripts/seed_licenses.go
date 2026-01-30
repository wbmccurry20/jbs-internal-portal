package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// TEMPORARY DEMO SEED SCRIPT
// This creates sample license data for testing/demo purposes only
// 
// IMPORTANT: Before importing real OneDrive data, run cleanup_demo_licenses.go
// to remove this demo data and prepare the database for production sync
//
// Future: This will be replaced by OneDrive sync that auto-imports
// license PDFs from Azure and parses metadata

func main() {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Connect to database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Sample licenses
	licenses := []struct {
		state          string
		licenseType    string
		licenseNumber  string
		entityName     string
		issueDate      string
		expirationDate string
		status         string
		renewalFee     float64
		notes          string
	}{
		{
			state:          "CA",
			licenseType:    "General Contractor",
			licenseNumber:  "CA-GC-123456",
			entityName:     "JBS Construction Group LLC",
			issueDate:      "2023-01-15",
			expirationDate: "2027-01-15",
			status:         "active",
			renewalFee:     450.00,
			notes:          "Primary license for West Coast operations",
		},
		{
			state:          "TX",
			licenseType:    "Commercial Builder",
			licenseNumber:  "TX-CB-789012",
			entityName:     "JBS Construction Group LLC",
			issueDate:      "2024-03-01",
			expirationDate: "2026-05-15",
			status:         "expiring",
			renewalFee:     350.00,
			notes:          "Renewal due soon - critical for AutoZone projects",
		},
		{
			state:          "FL",
			licenseType:    "General Contractor",
			licenseNumber:  "FL-GC-345678",
			entityName:     "JBS Construction Group LLC",
			issueDate:      "2022-06-10",
			expirationDate: "2025-12-31",
			status:         "expiring",
			renewalFee:     500.00,
			notes:          "Used for Panda Express and Driven Brands locations",
		},
		{
			state:          "AZ",
			licenseType:    "Commercial Contractor",
			licenseNumber:  "AZ-CC-901234",
			entityName:     "JBS Construction Group LLC",
			issueDate:      "2023-09-20",
			expirationDate: "2026-09-20",
			status:         "active",
			renewalFee:     400.00,
			notes:          "Recently renewed",
		},
		{
			state:          "NV",
			licenseType:    "General Building",
			licenseNumber:  "NV-GB-567890",
			entityName:     "JBS Construction Group LLC",
			issueDate:      "2021-11-05",
			expirationDate: "2024-06-30",
			status:         "expired",
			renewalFee:     425.00,
			notes:          "EXPIRED - Renewal needed for Las Vegas projects",
		},
	}

	fmt.Println("🌱 Seeding license data...")

	for _, lic := range licenses {
		// Check if already exists
		var count int
		err := db.QueryRow(
			"SELECT COUNT(*) FROM state_licenses WHERE state = $1",
			lic.state,
		).Scan(&count)
		
		if err != nil {
			log.Printf("Error checking for existing license in %s: %v", lic.state, err)
			continue
		}
		
		if count > 0 {
			fmt.Printf("  ⏭️  %s - already exists, skipping\n", lic.state)
			continue
		}

		// Insert license
		_, err = db.Exec(`
			INSERT INTO state_licenses 
				(state, license_type, license_number, entity_name, issue_date, 
				 expiration_date, status, renewal_fee, notes, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`,
			lic.state, lic.licenseType, lic.licenseNumber, lic.entityName, lic.issueDate,
			lic.expirationDate, lic.status, lic.renewalFee, lic.notes,
			time.Now(), time.Now(),
		)

		if err != nil {
			log.Printf("Error inserting license for %s: %v", lic.state, err)
			continue
		}

		fmt.Printf("  ✅ %s - %s (%s)\n", lic.state, lic.licenseType, lic.status)
	}

	fmt.Println("\n✨ License seeding complete!")
}
