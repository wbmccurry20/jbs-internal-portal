package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
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

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	log.Println("✓ Connected to database")

	// Import lodging
	if err := importLodging(db); err != nil {
		log.Fatalf("Failed to import lodging: %v", err)
	}

	log.Println("✅ Import complete!")
}

func importLodging(db *gorm.DB) error {
	file, err := os.Open("../AirBnbs-Table 1.csv")
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty")
	}

	// Load existing superintendents and jobs for matching
	var superintendents []struct {
		ID   int
		Name string
	}
	if err := db.Raw("SELECT id, name FROM superintendents").Scan(&superintendents).Error; err != nil {
		return fmt.Errorf("failed to load superintendents: %w", err)
	}

	superMap := make(map[string]int)
	for _, super := range superintendents {
		superMap[strings.ToLower(super.Name)] = super.ID
	}

	// Load jobs for matching by jobsite address
	var jobs []struct {
		ID      int
		Address string
		City    string
		State   string
	}
	if err := db.Raw("SELECT id, location as address, city, state FROM jobs").Scan(&jobs).Error; err != nil {
		return fmt.Errorf("failed to load jobs: %w", err)
	}

	lodgingImported := 0
	lodgingSkipped := 0

	log.Println("\n📊 Importing superintendent lodging...")

	// CSV columns: Superintendent,AirBnb,Check In Date,Check Out Date,Property Link,Address,Jobsite Address
	for i, record := range records[1:] {
		if len(record) < 7 {
			log.Printf("⚠️  Row %d: Invalid record length, skipping", i+2)
			lodgingSkipped++
			continue
		}

		superintendentName := strings.TrimSpace(record[0])
		airbnbLocation := strings.TrimSpace(record[1])
		checkInStr := strings.TrimSpace(record[2])
		checkOutStr := strings.TrimSpace(record[3])
		propertyLink := strings.TrimSpace(record[4])
		address := strings.TrimSpace(record[5])
		jobsiteAddress := strings.TrimSpace(record[6])

		// Skip if no superintendent or no location
		if superintendentName == "" || airbnbLocation == "" {
			log.Printf("⚠️  Row %d: No superintendent or location, skipping", i+2)
			lodgingSkipped++
			continue
		}

		// Match superintendent
		superID, found := superMap[strings.ToLower(superintendentName)]
		if !found {
			log.Printf("⚠️  Row %d (%s): Superintendent not found, skipping", i+2, superintendentName)
			lodgingSkipped++
			continue
		}

		// Try to match job by jobsite address
		var jobID *int
		if jobsiteAddress != "" {
			for _, job := range jobs {
				// Try exact match or partial match
				if strings.Contains(strings.ToLower(jobsiteAddress), strings.ToLower(job.City)) ||
					strings.Contains(strings.ToLower(job.Address), strings.ToLower(jobsiteAddress)) {
					jobID = &job.ID
					break
				}
			}
		}

		// Parse dates
		var checkIn, checkOut *time.Time
		if checkInStr != "" {
			if t, err := parseDate(checkInStr); err == nil {
				checkIn = &t
			}
		}
		if checkOutStr != "" {
			if t, err := parseDate(checkOutStr); err == nil {
				checkOut = &t
			}
		}

		// Insert lodging
		query := `
			INSERT INTO superintendent_lodging (
				superintendent_id, job_id, location,
				check_in_date, check_out_date, property_link,
				address, jobsite_address,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
			RETURNING id
		`

		var lodgingID int
		err := db.Raw(query,
			superID, jobID, airbnbLocation,
			checkIn, checkOut, nullString(propertyLink),
			nullString(address), nullString(jobsiteAddress),
		).Scan(&lodgingID).Error

		if err != nil {
			log.Printf("⚠️  Row %d (%s - %s): Failed to insert lodging: %v", i+2, superintendentName, airbnbLocation, err)
			lodgingSkipped++
			continue
		}

		lodgingImported++
		jobLabel := ""
		if jobID != nil {
			jobLabel = fmt.Sprintf(" [Job #%d]", *jobID)
		}
		datesLabel := ""
		if checkIn != nil || checkOut != nil {
			if checkIn != nil && checkOut != nil {
				datesLabel = fmt.Sprintf(" (%s - %s)", checkIn.Format("01/02"), checkOut.Format("01/02"))
			} else if checkIn != nil {
				datesLabel = fmt.Sprintf(" (from %s)", checkIn.Format("01/02"))
			} else {
				datesLabel = fmt.Sprintf(" (until %s)", checkOut.Format("01/02"))
			}
		}
		log.Printf("✓ Imported: %s - %s%s%s", superintendentName, airbnbLocation, datesLabel, jobLabel)
	}

	log.Printf("\n📊 Import Summary:")
	log.Printf("   ✅ Lodging imported: %d", lodgingImported)
	log.Printf("   ⚠️  Lodging skipped: %d", lodgingSkipped)

	return nil
}

// Helper functions

func parseDate(s string) (time.Time, error) {
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}

	// Try MM/DD/YY format
	t, err := time.Parse("01/02/06", s)
	if err == nil {
		return t, nil
	}

	// Try MM/DD/YYYY format
	t, err = time.Parse("01/02/2006", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, err
}

func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
