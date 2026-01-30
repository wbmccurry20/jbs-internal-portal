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

	// Import completed jobs
	if err := importCompletedJobs(db); err != nil {
		log.Fatalf("Failed to import completed jobs: %v", err)
	}

	log.Println("✅ Import complete!")
}

func importCompletedJobs(db *gorm.DB) error {
	file, err := os.Open("../Completed Jobs-Table 1.csv")
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

	// Load existing clients and superintendents for matching
	var clients []struct {
		ID   int
		Name string
	}
	if err := db.Raw("SELECT id, name FROM clients").Scan(&clients).Error; err != nil {
		return fmt.Errorf("failed to load clients: %w", err)
	}

	var superintendents []struct {
		ID   int
		Name string
	}
	if err := db.Raw("SELECT id, name FROM superintendents").Scan(&superintendents).Error; err != nil {
		return fmt.Errorf("failed to load superintendents: %w", err)
	}

	clientMap := make(map[string]int)
	for _, client := range clients {
		clientMap[strings.ToLower(client.Name)] = client.ID
	}

	superMap := make(map[string]int)
	for _, super := range superintendents {
		superMap[strings.ToLower(super.Name)] = super.ID
	}

	jobsImported := 0
	jobsSkipped := 0
	notesAdded := 0

	log.Println("\n📊 Importing completed jobs...")

	for i, record := range records[1:] {
		if len(record) < 12 {
			log.Printf("⚠️  Row %d: Invalid record length, skipping", i+2)
			jobsSkipped++
			continue
		}

		// Parse CSV columns (same as Active Jobs)
		jobNumber := strings.TrimSpace(record[0])
		clientName := strings.TrimSpace(record[1])
		status := strings.TrimSpace(record[2])
		_ = strings.TrimSpace(record[3]) // projectManager
		_ = strings.TrimSpace(record[4]) // apm
		superintendent := strings.TrimSpace(record[5])
		contractValue := strings.TrimSpace(record[6])
		revisedContract := strings.TrimSpace(record[7])
		startDate := strings.TrimSpace(record[8])
		projectedEndDate := strings.TrimSpace(record[9])
		actualEndDate := strings.TrimSpace(record[10])
		meetingMinutes := strings.TrimSpace(record[11])

		// Skip if no job number or duplicate
		if jobNumber == "" {
			log.Printf("⚠️  Row %d: No job number, skipping", i+2)
			jobsSkipped++
			continue
		}

		// Check if job already exists
		var existingID int
		db.Raw("SELECT id FROM jobs WHERE job_number = ?", strings.Split(jobNumber, " ")[0]).Scan(&existingID)
		if existingID > 0 {
			log.Printf("⚠️  Row %d (%s): Job already exists, skipping", i+2, jobNumber)
			jobsSkipped++
			continue
		}

		// Parse location from job number
		location := ""
		city := ""
		state := ""

		parts := strings.SplitN(jobNumber, " ", 2)
		if len(parts) > 1 {
			location = strings.TrimSpace(parts[1])
			city, state = parseLocation(location)
		}

		jobNumberClean := parts[0]

		// Match client
		var clientID *int
		if clientName != "" && clientName != "Misc" {
			if id, found := clientMap[strings.ToLower(clientName)]; found {
				clientID = &id
			}
		}

		// Match superintendent
		var superID *int
		if superintendent != "" && superintendent != "None" && superintendent != "NEED" && superintendent != "TBD" && !strings.Contains(strings.ToLower(superintendent), "fired") {
			if id, found := superMap[strings.ToLower(superintendent)]; found {
				superID = &id
			}
		}

		// Parse contract values
		contractValueFloat := parseMoneyToFloat(contractValue)
		revisedContractFloat := parseMoneyToFloat(revisedContract)

		var contractValuePtr *float64
		var revisedContractPtr *float64
		if contractValueFloat > 0 {
			contractValuePtr = &contractValueFloat
		}
		if revisedContractFloat > 0 {
			revisedContractPtr = &revisedContractFloat
		}

		// Parse dates
		var startDateParsed *time.Time
		var projectedEndDateParsed *time.Time
		var actualEndDateParsed *time.Time

		if startDate != "" {
			if t, err := parseDate(startDate); err == nil {
				startDateParsed = &t
			}
		}
		if projectedEndDate != "" {
			if t, err := parseDate(projectedEndDate); err == nil {
				projectedEndDateParsed = &t
			}
		}
		if actualEndDate != "" {
			if t, err := parseDate(actualEndDate); err == nil {
				actualEndDateParsed = &t
			}
		}

		// Insert job using raw SQL
		query := `
			INSERT INTO jobs (
				job_number, job_name, location, city, state,
				client_id, superintendent_id, status,
				contract_value, revised_contract_value,
				start_date, projected_end_date, actual_end_date,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW(), NOW())
			RETURNING id
		`

		var jobID int
		err := db.Raw(query,
			jobNumberClean, location, location, city, state,
			clientID, superID, normalizeStatus(status),
			contractValuePtr, revisedContractPtr,
			startDateParsed, projectedEndDateParsed, actualEndDateParsed,
		).Scan(&jobID).Error

		if err != nil {
			log.Printf("⚠️  Row %d (%s): Failed to insert job: %v", i+2, jobNumber, err)
			jobsSkipped++
			continue
		}

		jobsImported++
		log.Printf("✓ Imported: %s - %s (%s)", jobNumberClean, location, status)

		// Add meeting minutes as job update if present
		if meetingMinutes != "" {
			db.Exec(`
				INSERT INTO job_updates (job_id, update_text, author_name, created_at)
				VALUES ($1, $2, $3, NOW())
			`, jobID, meetingMinutes, "Smartsheet Import")
			notesAdded++
		}
	}

	log.Printf("\n📊 Import Summary:")
	log.Printf("   ✅ Jobs imported: %d", jobsImported)
	log.Printf("   📝 Meeting notes added: %d", notesAdded)
	log.Printf("   ⚠️  Jobs skipped: %d", jobsSkipped)

	return nil
}

// Helper functions (same as import_smartsheet.go)

func parseLocation(location string) (city, state string) {
	parts := strings.Split(location, ",")
	if len(parts) >= 2 {
		city = strings.TrimSpace(parts[0])
		statePart := strings.TrimSpace(parts[1])
		words := strings.Fields(statePart)
		if len(words) > 0 {
			firstWord := words[0]
			firstWord = strings.TrimRight(firstWord, "-()")
			if len(firstWord) >= 2 {
				state = firstWord[:2]
			}
		}
		return
	}

	words := strings.Fields(location)
	if len(words) > 0 {
		lastWord := words[len(words)-1]
		if len(lastWord) == 2 && strings.ToUpper(lastWord) == lastWord {
			state = lastWord
			city = strings.TrimSpace(strings.TrimSuffix(location, lastWord))
			return
		}
	}

	city = location
	return
}

func parseMoneyToFloat(s string) float64 {
	if s == "" {
		return 0
	}
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	var value float64
	fmt.Sscanf(s, "%f", &value)
	return value
}

func parseDate(s string) (time.Time, error) {
	if s == "" || s == "Delayed" {
		return time.Time{}, fmt.Errorf("empty or invalid date")
	}

	t, err := time.Parse("01/02/06", s)
	if err == nil {
		return t, nil
	}

	t, err = time.Parse("01/02/2006", s)
	if err == nil {
		return t, nil
	}

	return time.Time{}, err
}

func normalizeStatus(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "in progress":
		return "in_progress"
	case "not started":
		return "not_started"
	case "close out", "closeout":
		return "closeout"
	case "permitting":
		return "permitting"
	case "completed", "complete":
		return "completed"
	default:
		if s == "" {
			return "not_started"
		}
		return s
	}
}
