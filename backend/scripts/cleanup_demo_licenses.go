package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// Clean up demo/seed license data
// Use this before importing real data from OneDrive
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

	fmt.Println("🧹 Cleaning up demo license data...")
	fmt.Println("⚠️  This will delete ALL licenses in the database.")
	fmt.Print("Continue? (yes/no): ")

	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("❌ Cleanup cancelled")
		return
	}

	// Delete all licenses
	result, err := db.Exec("DELETE FROM state_licenses")
	if err != nil {
		log.Fatal("Failed to delete licenses:", err)
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("✅ Deleted %d license records\n", rowsAffected)

	// Reset the sequence
	_, err = db.Exec("ALTER SEQUENCE state_licenses_id_seq RESTART WITH 1")
	if err != nil {
		log.Println("Warning: Could not reset ID sequence:", err)
	} else {
		fmt.Println("✅ Reset ID sequence")
	}

	fmt.Println("\n✨ Database is clean and ready for OneDrive import!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Configure Azure credentials in .env")
	fmt.Println("2. Run OneDrive sync script to import real license files")
	fmt.Println("3. Licenses will auto-populate from OneDrive folder")
}
