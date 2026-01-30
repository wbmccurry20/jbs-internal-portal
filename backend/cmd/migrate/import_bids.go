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

	// Import active bids
	if err := importBids(db, "../Bids-Table 1.csv", false); err != nil {
		log.Fatalf("Failed to import active bids: %v", err)
	}

	// Import completed bids
	if err := importBids(db, "../Completed Bids-Table 1.csv", true); err != nil {
		log.Fatalf("Failed to import completed bids: %v", err)
	}

	log.Println("✅ Import complete!")
}

func importBids(db *gorm.DB, filename string, archived bool) error {
	file, err := os.Open(filename)
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

	// Load existing clients for matching
	var clients []models.Client
	if err := db.Find(&clients).Error; err != nil {
		return fmt.Errorf("failed to load clients: %w", err)
	}

	clientMap := make(map[string]int)
	for _, client := range clients {
		clientMap[strings.ToLower(client.Name)] = client.ID
	}

	bidsImported := 0
	bidsSkipped := 0

	status := "active"
	if archived {
		status = "archived"
	}

	log.Printf("\n📊 Importing %s bids from %s...\n", status, filename)

	// Determine column indices based on header
	header := records[0]
	var clientIdx, locationIdx, dueDateIdx, assignedToIdx, statusIdx, bcIdx, phIdx, awardedIdx int
	
	for i, col := range header {
		col = strings.TrimSpace(col)
		switch col {
		case "Bid Client", "Client":
			clientIdx = i
		case "Location":
			locationIdx = i
		case "Due Date":
			dueDateIdx = i
		case "Assigned To":
			assignedToIdx = i
		case "Status":
			statusIdx = i
		case "Building Connected":
			bcIdx = i
		case "Plan Hub":
			phIdx = i
		case "Awarded":
			awardedIdx = i
		}
	}

	for i, record := range records[1:] {
		if len(record) < 4 {
			log.Printf("⚠️  Row %d: Invalid record length, skipping", i+2)
			bidsSkipped++
			continue
		}

		// Parse fields
		clientName := strings.TrimSpace(record[clientIdx])
		location := strings.TrimSpace(record[locationIdx])
		dueDateStr := strings.TrimSpace(record[dueDateIdx])
		assignedTo := strings.TrimSpace(record[assignedToIdx])
		bidStatus := strings.TrimSpace(record[statusIdx])
		bcDateStr := ""
		phDateStr := ""
		awarded := ""
		
		if bcIdx < len(record) {
			bcDateStr = strings.TrimSpace(record[bcIdx])
		}
		if phIdx < len(record) {
			phDateStr = strings.TrimSpace(record[phIdx])
		}
		if awardedIdx < len(record) {
			awarded = strings.TrimSpace(record[awardedIdx])
		}

		// Skip if no location
		if location == "" {
			log.Printf("⚠️  Row %d: No location, skipping", i+2)
			bidsSkipped++
			continue
		}

		// Parse location (city, state)
		city, state := parseLocation(location)

		// Match client
		var clientID *int
		if clientName != "" && clientName != "Misc" {
			if id, found := clientMap[strings.ToLower(clientName)]; found {
				clientID = &id
			}
		}

		// Parse dates
		var dueDate, bcDate, phDate *time.Time
		if dueDateStr != "" {
			if t, err := parseDate(dueDateStr); err == nil {
				dueDate = &t
			}
		}
		if bcDateStr != "" {
			if t, err := parseDate(bcDateStr); err == nil {
				bcDate = &t
			}
		}
		if phDateStr != "" {
			if t, err := parseDate(phDateStr); err == nil {
				phDate = &t
			}
		}

		// Normalize status
		bidStatus = normalizeStatus(bidStatus)

		// Create bid record using raw SQL
		query := `
			INSERT INTO bids (
				client_id, location, city, state, due_date,
				assigned_to_name, status, building_connected_date,
				plan_hub_date, awarded, archived,
				created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
			RETURNING id
		`

		var bidID int
		err := db.Raw(query,
			clientID, location, city, state, dueDate,
			nullString(assignedTo), bidStatus, bcDate,
			phDate, nullString(awarded), archived,
		).Scan(&bidID).Error

		if err != nil {
			log.Printf("⚠️  Row %d (%s): Failed to insert bid: %v", i+2, location, err)
			bidsSkipped++
			continue
		}

		bidsImported++
		awardedLabel := ""
		if awarded != "" {
			awardedLabel = fmt.Sprintf(" [%s]", awarded)
		}
		log.Printf("✓ Imported: %s - %s (%s)%s", clientName, location, bidStatus, awardedLabel)
	}

	log.Printf("\n📊 Import Summary (%s):", status)
	log.Printf("   ✅ Bids imported: %d", bidsImported)
	log.Printf("   ⚠️  Bids skipped: %d", bidsSkipped)

	return nil
}

// parseLocation extracts city and state from location strings
func parseLocation(location string) (city, state string) {
	// Handle special cases with parentheses or dashes
	// e.g., "Jacksonville, FL (Normandy)" -> "Jacksonville", "FL"
	// e.g., "Ft Lauderdale, FL-DEMO" -> "Ft Lauderdale", "FL"
	// e.g., "Miami, FL 8th st" -> "Miami", "FL"
	
	parts := strings.Split(location, ",")
	if len(parts) >= 2 {
		city = strings.TrimSpace(parts[0])
		statePart := strings.TrimSpace(parts[1])
		
		// Extract just the state code (first 2 uppercase letters)
		words := strings.Fields(statePart)
		if len(words) > 0 {
			firstWord := words[0]
			// Remove any trailing punctuation or parentheses
			firstWord = strings.TrimRight(firstWord, "-()")
			if len(firstWord) >= 2 {
				state = firstWord[:2]
			}
		}
		return
	}

	// Try to extract state code at the end
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

// parseDate parses dates in MM/DD/YY format
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

// normalizeStatus converts various status strings to standard values
func normalizeStatus(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	switch s {
	case "complete", "completed":
		return "complete"
	case "in progress", "in_progress":
		return "in_progress"
	case "not started", "not_started":
		return "not_started"
	case "no access", "no_access":
		return "no_access"
	case "did not bid", "did_not_bid":
		return "did_not_bid"
	case "not bidding", "not_bidding":
		return "not_bidding"
	default:
		if s == "" {
			return "in_progress"
		}
		return s
	}
}

// nullString returns nil if string is empty
func nullString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
