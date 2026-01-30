package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/wbmccurry20/jbs-internal-portal/internal/models"
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

	// Import jobs
	if err := importJobs(db); err != nil {
		log.Fatalf("Failed to import jobs: %v", err)
	}

	log.Println("✅ Import complete!")
}

func importJobs(db *gorm.DB) error {
	// Open CSV file (in parent directory)
	file, err := os.Open("../Active Jobs-Table 1.csv")
	if err != nil {
		return fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return fmt.Errorf("failed to read CSV: %w", err)
	}

	// Skip header row
	if len(records) < 2 {
		return fmt.Errorf("CSV file is empty")
	}

	// Load existing clients and superintendents for matching
	var clients []models.Client
	if err := db.Find(&clients).Error; err != nil {
		return fmt.Errorf("failed to load clients: %w", err)
	}

	var superintendents []models.Superintendent
	if err := db.Raw("SELECT id, name, email, phone, active FROM superintendents").Scan(&superintendents).Error; err != nil {
		return fmt.Errorf("failed to load superintendents: %w", err)
	}

	// Create lookup maps
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

	for i, record := range records[1:] {
		if len(record) < 12 {
			log.Printf("⚠️  Row %d: Invalid record length, skipping", i+2)
			jobsSkipped++
			continue
		}

		// Parse CSV columns
		jobNumber := strings.TrimSpace(record[0])
		clientName := strings.TrimSpace(record[1])
		status := strings.TrimSpace(record[2])
		_ = strings.TrimSpace(record[3]) // projectManager - not used yet
		_ = strings.TrimSpace(record[4]) // apm - not used yet
		superintendent := strings.TrimSpace(record[5])
		contractValue := strings.TrimSpace(record[6])
		revisedContract := strings.TrimSpace(record[7])
		startDate := strings.TrimSpace(record[8])
		projectedEndDate := strings.TrimSpace(record[9])
		actualEndDate := strings.TrimSpace(record[10])
		meetingMinutes := strings.TrimSpace(record[11])

		// Skip if no job number
		if jobNumber == "" {
			log.Printf("⚠️  Row %d: No job number, skipping", i+2)
			jobsSkipped++
			continue
		}

		// Parse location from job number (e.g., "2025-710 Orange City, FL")
		location := ""
		city := ""
		state := ""

		// Extract location after the job number
		parts := strings.SplitN(jobNumber, " ", 2)
		if len(parts) > 1 {
			location = strings.TrimSpace(parts[1])
			// Parse city and state
			city, state = parseLocation(location)
		}

		// Extract just the job number part
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
		if superintendent != "" && superintendent != "None" && superintendent != "NEED" && superintendent != "TBD" && superintendent != "" {
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

		// Create job record
		job := models.Job{
			JobNumber:            jobNumberClean,
			JobName:              location, // Use location as job name
			Location:             location,
			ClientID:             clientID,
			SuperintendentID:     superID,
			Status:               normalizeStatus(status),
			ContractValue:        contractValuePtr,
			RevisedContractValue: revisedContractPtr,
			StartDate:            startDateParsed,
			ProjectedEndDate:     projectedEndDateParsed,
			ActualEndDate:        actualEndDateParsed,
			City:                 city,
			State:                state,
		}

		// Insert job using raw SQL to avoid GORM relationship loading issues
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
			job.JobNumber, job.JobName, job.Location, job.City, job.State,
			job.ClientID, job.SuperintendentID, job.Status,
			job.ContractValue, job.RevisedContractValue,
			job.StartDate, job.ProjectedEndDate, job.ActualEndDate,
		).Scan(&jobID).Error
		
		if err != nil {
			log.Printf("⚠️  Row %d (%s): Failed to insert job: %v", i+2, jobNumber, err)
			jobsSkipped++
			continue
		}
		
		job.ID = jobID

		jobsImported++
		log.Printf("✓ Imported: %s - %s (%s)", job.JobNumber, job.JobName, status)

		// Add meeting minutes as job update if present
		if meetingMinutes != "" {
			jobUpdate := models.JobUpdate{
				JobID:      job.ID,
				UpdateText: meetingMinutes,
				AuthorName: "Smartsheet Import",
			}

			if err := db.Create(&jobUpdate).Error; err != nil {
				log.Printf("⚠️  Failed to add meeting notes for job %s: %v", jobNumber, err)
			} else {
				notesAdded++
			}
		}
	}

	log.Printf("\n📊 Import Summary:")
	log.Printf("   ✅ Jobs imported: %d", jobsImported)
	log.Printf("   📝 Meeting notes added: %d", notesAdded)
	log.Printf("   ⚠️  Jobs skipped: %d", jobsSkipped)

	return nil
}

// parseLocation extracts city and state from location strings
// Examples: "Orange City, FL" -> "Orange City", "FL"
func parseLocation(location string) (city, state string) {
	// Remove parenthetical info
	re := regexp.MustCompile(`\([^)]*\)`)
	location = re.ReplaceAllString(location, "")
	location = strings.TrimSpace(location)

	// Split by comma
	parts := strings.Split(location, ",")
	if len(parts) >= 2 {
		city = strings.TrimSpace(parts[0])
		stateCandidate := strings.TrimSpace(parts[1])
		
		// Extract state code from strings like "AZ - Durban"
		stateWords := strings.Fields(stateCandidate)
		if len(stateWords) > 0 {
			firstWord := stateWords[0]
			// Check if it's a 2-letter state code
			if len(firstWord) == 2 && strings.ToUpper(firstWord) == firstWord {
				state = firstWord
				return
			}
		}
		
		// If not a 2-letter code, try to extract state abbreviation
		// Common format: "City, State Name" - skip for now
		return
	}

	// If no comma, check if it ends with a state code
	words := strings.Fields(location)
	if len(words) > 0 {
		lastWord := words[len(words)-1]
		if len(lastWord) == 2 && strings.ToUpper(lastWord) == lastWord {
			state = lastWord
			city = strings.TrimSpace(strings.TrimSuffix(location, lastWord))
			return
		}
		
		// Check second to last word (handles "Show Low, AZ")
		if len(words) > 1 {
			secondLast := words[len(words)-2]
			if len(secondLast) == 2 && strings.ToUpper(secondLast) == secondLast {
				state = secondLast
				city = strings.TrimSpace(strings.TrimSuffix(location, secondLast+" "+lastWord))
				return
			}
		}
	}

	// Default: use entire location as city
	city = location
	return
}

// parseMoneyToFloat converts money strings like "$1,190,000.00" to float64
func parseMoneyToFloat(s string) float64 {
	if s == "" {
		return 0
	}

	// Remove $, commas, and spaces
	s = strings.ReplaceAll(s, "$", "")
	s = strings.ReplaceAll(s, ",", "")
	s = strings.TrimSpace(s)

	value, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}

	return value
}

// parseDate parses dates in MM/DD/YY format
func parseDate(s string) (time.Time, error) {
	if s == "" || s == "Delayed" {
		return time.Time{}, fmt.Errorf("empty or invalid date")
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

// normalizeStatus converts various status strings to standard values
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

// nullString returns nil if string is empty, otherwise returns pointer to string
func nullString(s string) *string {
	if s == "" || s == "None" || s == "NEED" || s == "TBD" {
		return nil
	}
	return &s
}
