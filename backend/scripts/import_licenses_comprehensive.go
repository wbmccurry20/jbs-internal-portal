package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type License struct {
	State          string
	City           string
	IsCityLicense  bool
	LicenseType    string
	LicenseNumber  string
	EntityName     string
	IssueDate      *time.Time
	ExpirationDate *time.Time
	Status         string
	IsActive       bool
	Notes          string
	Files          []LicenseFile
}

type LicenseFile struct {
	FileName      string
	FilePath      string
	FileType      string
	YearExtracted int
	IsPrimary     bool
}

func main() {
	godotenv.Load(".env")

	tenantID := os.Getenv("AZURE_TENANT_ID")
	clientID := os.Getenv("AZURE_CLIENT_ID")
	clientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	driveID := os.Getenv("SHAREPOINT_DRIVE_ID")
	dbURL := os.Getenv("DATABASE_URL")

	if tenantID == "" || clientID == "" || clientSecret == "" || dbURL == "" {
		log.Fatal("❌ Missing credentials in .env file")
	}

	fmt.Println("📥 JBS Comprehensive License Import")
	fmt.Println("✓ City licenses supported")
	fmt.Println("✓ Smart deduplication enabled")
	fmt.Println("✓ File tracking enabled")
	fmt.Println()

	// Connect to database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("❌ Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create Azure client
	cred, _ := azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
	client, _ := msgraphsdk.NewGraphServiceClientWithCredentials(cred, []string{"https://graph.microsoft.com/.default"})
	ctx := context.Background()

	// Get Licensing folder
	root, _ := client.Drives().ByDriveId(driveID).Root().Get(ctx, nil)
	rootID := *root.GetId()
	children, _ := client.Drives().ByDriveId(driveID).Items().ByDriveItemId(rootID).Children().Get(ctx, nil)

	var licensingFolderID string
	for _, item := range children.GetValue() {
		if item.GetFolder() != nil && strings.EqualFold(*item.GetName(), "Licensing") {
			licensingFolderID = *item.GetId()
			break
		}
	}

	if licensingFolderID == "" {
		log.Fatal("❌ Licensing folder not found")
	}

	// Get state folders
	licensingChildren, _ := client.Drives().ByDriveId(driveID).Items().ByDriveItemId(licensingFolderID).Children().Get(ctx, nil)
	stateFolders := licensingChildren.GetValue()

	fmt.Printf("📁 Found %d state folders\n\n", len(stateFolders))

	totalLicenses := 0
	totalFiles := 0

	// Process each state folder
	for _, stateFolder := range stateFolders {
		if stateFolder.GetFolder() == nil {
			continue
		}

		folderName := *stateFolder.GetName()
		
		// Skip non-state folders
		if folderName == "0_Documents" {
			continue
		}

		stateCode := extractStateCode(folderName)
		if stateCode == "" {
			log.Printf("⚠️  Skipping: %s", folderName)
			continue
		}

		fmt.Printf("📍 %s (%s)\n", folderName, stateCode)

		// Process state folder
		stateFolderID := *stateFolder.GetId()
		licenses, files := processStateFolder(ctx, client, driveID, stateFolderID, stateCode, folderName)

		// Import licenses with deduplication
		imported := importLicenses(db, licenses)
		totalLicenses += imported
		totalFiles += files

		fmt.Printf("   ✅ %d licenses, %d files\n", imported, files)
	}

	fmt.Println()
	fmt.Printf("🎉 Import complete!\n")
	fmt.Printf("   Licenses: %d\n", totalLicenses)
	fmt.Printf("   Files tracked: %d\n", totalFiles)
}

func processStateFolder(ctx context.Context, client *msgraphsdk.GraphServiceClient, driveID, folderID, stateCode, stateName string) ([]License, int) {
	contents, err := client.Drives().ByDriveId(driveID).Items().ByDriveItemId(folderID).Children().Get(ctx, nil)
	if err != nil {
		log.Printf("   ❌ Error reading folder: %v", err)
		return nil, 0
	}

	licenseMap := make(map[string]*License) // Key: state+city+entity
	totalFiles := 0

	items := contents.GetValue()
	for _, item := range items {
		if item.GetFolder() != nil {
			// City subfolder
			cityName := *item.GetName()
			cityFolderID := *item.GetId()
			
			cityLicenses, cityFiles := processCityFolder(ctx, client, driveID, cityFolderID, stateCode, cityName, stateName)
			totalFiles += cityFiles
			
			// Merge city licenses
			for i := range cityLicenses {
				lic := &cityLicenses[i]
				key := licenseKey(*lic)
				if existing, exists := licenseMap[key]; exists {
					// Merge files
					existing.Files = append(existing.Files, lic.Files...)
					// Keep best data
					mergeLicenseData(existing, lic)
				} else {
					licenseMap[key] = lic
				}
			}
		} else {
			// File in state root
			fileName := *item.GetName()
			ext := strings.ToLower(filepath.Ext(fileName))
			
			if ext == ".pdf" || ext == ".xlsx" || ext == ".xls" {
				totalFiles++
				
				lic := License{
					State:         stateCode,
					IsCityLicense: false,
					LicenseType:   "General Contractor",
					EntityName:    inferEntityName(fileName),
					Status:        "active",
					IsActive:      true,
				}
				
				file := LicenseFile{
					FileName:      fileName,
					FilePath:      fmt.Sprintf("Licensing/%s/%s", stateName, fileName),
					FileType:      inferFileType(fileName),
					YearExtracted: extractYear(fileName),
					IsPrimary:     true,
				}
				
				lic.Files = []LicenseFile{file}
				
				key := licenseKey(lic)
				if existing, exists := licenseMap[key]; exists {
					existing.Files = append(existing.Files, file)
				} else {
					licenseMap[key] = &lic
				}
			}
		}
	}

	// Convert map to slice
	var licenses []License
	for _, lic := range licenseMap {
		licenses = append(licenses, *lic)
	}

	return licenses, totalFiles
}

func processCityFolder(ctx context.Context, client *msgraphsdk.GraphServiceClient, driveID, folderID, stateCode, cityName, stateName string) ([]License, int) {
	contents, err := client.Drives().ByDriveId(driveID).Items().ByDriveItemId(folderID).Children().Get(ctx, nil)
	if err != nil {
		return nil, 0
	}

	var licenses []License
	fileCount := 0

	lic := License{
		State:         stateCode,
		City:          cityName,
		IsCityLicense: true,
		LicenseType:   "General Contractor",
		EntityName:    "JBS Construction Group LLC",
		Status:        "active",
		IsActive:      true,
		Notes:         fmt.Sprintf("City-specific license for %s", cityName),
	}

	items := contents.GetValue()
	for _, item := range items {
		if item.GetFolder() == nil {
			fileName := *item.GetName()
			ext := strings.ToLower(filepath.Ext(fileName))
			
			if ext == ".pdf" || ext == ".xlsx" || ext == ".xls" {
				fileCount++
				
				file := LicenseFile{
					FileName:      fileName,
					FilePath:      fmt.Sprintf("Licensing/%s/%s/%s", stateName, cityName, fileName),
					FileType:      inferFileType(fileName),
					YearExtracted: extractYear(fileName),
					IsPrimary:     fileCount == 1,
				}
				
				lic.Files = append(lic.Files, file)
			}
		}
	}

	if fileCount > 0 {
		licenses = append(licenses, lic)
	}

	return licenses, fileCount
}

func importLicenses(db *sql.DB, licenses []License) int {
	imported := 0
	
	for _, lic := range licenses {
		// Insert license
		var licenseID int
		err := db.QueryRow(`
			INSERT INTO state_licenses (
				state, city, is_city_license, license_type, license_number,
				entity_name, issue_date, expiration_date, status, is_active,
				notes, created_at, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
			RETURNING id
		`, lic.State, lic.City, lic.IsCityLicense, lic.LicenseType, lic.LicenseNumber,
			lic.EntityName, lic.IssueDate, lic.ExpirationDate, lic.Status, lic.IsActive, lic.Notes).Scan(&licenseID)
		
		if err != nil {
			log.Printf("   ❌ Failed to insert license: %v", err)
			continue
		}

		// Insert files
		for _, file := range lic.Files {
			_, err := db.Exec(`
				INSERT INTO license_files (
					license_id, file_name, file_path, file_type, year_extracted, is_primary, created_at, updated_at
				) VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			`, licenseID, file.FileName, file.FilePath, file.FileType, file.YearExtracted, file.IsPrimary)
			
			if err != nil {
				log.Printf("   ❌ Failed to insert file: %v", err)
			}
		}

		imported++
	}

	return imported
}

func licenseKey(lic License) string {
	return fmt.Sprintf("%s|%s|%s", lic.State, lic.City, lic.EntityName)
}

func mergeLicenseData(existing, new *License) {
	if new.LicenseNumber != "" && existing.LicenseNumber == "" {
		existing.LicenseNumber = new.LicenseNumber
	}
	if new.ExpirationDate != nil && existing.ExpirationDate == nil {
		existing.ExpirationDate = new.ExpirationDate
	}
	if new.IssueDate != nil && existing.IssueDate == nil {
		existing.IssueDate = new.IssueDate
	}
}

func extractStateCode(folderName string) string {
	re := regexp.MustCompile(`\(([A-Z]{2})\)`)
	if matches := re.FindStringSubmatch(folderName); len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func inferEntityName(filename string) string {
	lower := strings.ToLower(filename)
	if strings.Contains(lower, "management") {
		return "JBS Management Group LLC"
	}
	return "JBS Construction Group LLC"
}

func inferFileType(filename string) string {
	lower := strings.ToLower(filename)
	if strings.Contains(lower, "application") || strings.Contains(lower, "app") {
		return "application"
	}
	if strings.Contains(lower, "renewal") || strings.Contains(lower, "renew") {
		return "renewal"
	}
	if strings.Contains(lower, "final") || strings.Contains(lower, "current") || strings.Contains(lower, "active") {
		return "license"
	}
	return "supporting_doc"
}

func extractYear(filename string) int {
	re := regexp.MustCompile(`20\d{2}`)
	if match := re.FindString(filename); match != "" {
		var year int
		fmt.Sscanf(match, "%d", &year)
		return year
	}
	return 0
}
