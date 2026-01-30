package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load(".env")
	
	db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("📊 License Summary by State:\n")
	
	rows, err := db.Query("SELECT state, COUNT(*) as count FROM state_licenses GROUP BY state ORDER BY state")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	total := 0
	for rows.Next() {
		var state string
		var count int
		rows.Scan(&state, &count)
		fmt.Printf("%s: %d licenses\n", state, count)
		total += count
	}
	
	fmt.Printf("\n✅ Total: %d licenses across all states\n", total)
}
