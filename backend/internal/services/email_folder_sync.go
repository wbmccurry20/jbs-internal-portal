package services

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strings"
)

// TODO: Confirm the exact Outlook folder name for bid-submitted emails with JBS
var bidSubmittedFolderName = ".Bid Submitted"

// folderEntry is an intermediate record built during folder traversal before upsert.
type folderEntry struct {
	folder            MailFolder
	parentFolderName  string
	folderPath        string
	isBidSubmittedSub bool
}

// multiSpaceRE collapses runs of whitespace to a single space.
var multiSpaceRE = regexp.MustCompile(`\s+`)

// EmailFolderMapping mirrors a row in the email_folder_mappings table.
// Defined here (Ticket 6); reused by the classifier in Ticket 5 — do not redefine there.
type EmailFolderMapping struct {
	ID                int
	ProjectName       string
	FolderID          string
	FolderName        string
	FolderPath        string
	ParentFolderName  string
	IsBidSubmittedSub bool
	BidID             *int
}

// TODO: Confirm with JBS — system folders to skip (these are never project folders)
var skipFolderNames = map[string]bool{
	"sent items":           true,
	"deleted items":        true,
	"drafts":               true,
	"outbox":               true,
	"junk email":           true,
	"archive":              true,
	"conversation history": true,
}

// maxFolderDepth is the maximum recursion depth for folder traversal.
// root(0) → Inbox(1) → .Bid Submitted(2) → job folder(3) is enough; the cap guards against loops.
const maxFolderDepth = 4

// walkFolder appends folder and, if it has children, all descendants up to maxFolderDepth.
// underBidSubmitted is true when any ancestor was named bidSubmittedFolderName.
func walkFolder(
	graphClient *GraphClient,
	mailbox string,
	folder MailFolder,
	parentName, parentPath string,
	underBidSubmitted bool,
	depth int,
	entries *[]folderEntry,
) {
	// Skip well-known system folders; never skip Inbox.
	if skipFolderNames[NormalizeFolderName(folder.DisplayName)] {
		return
	}

	path := folder.DisplayName
	if parentPath != "" {
		path = parentPath + "/" + folder.DisplayName
	}

	*entries = append(*entries, folderEntry{
		folder:            folder,
		parentFolderName:  parentName,
		folderPath:        path,
		isBidSubmittedSub: underBidSubmitted,
	})

	if folder.ChildFolderCount == 0 || depth >= maxFolderDepth {
		return
	}

	children, err := graphClient.ListSubFolders(mailbox, folder.ID)
	if err != nil {
		log.Printf("[EmailSync] Warning: could not list children of %q: %v", folder.DisplayName, err)
		return
	}

	isThisBidSubmitted := strings.EqualFold(folder.DisplayName, bidSubmittedFolderName)
	for _, child := range children {
		walkFolder(
			graphClient, mailbox, child,
			folder.DisplayName, path,
			underBidSubmitted || isThisBidSubmitted,
			depth+1, entries,
		)
	}
}

// trims leading/trailing whitespace, and collapses runs of spaces to one.
// Used both during upsert and by the bid cross-reference matcher.
func NormalizeFolderName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.TrimSpace(s)
	s = multiSpaceRE.ReplaceAllString(s, " ")
	return s
}

// SyncFolderMappings discovers Outlook folders from the shared mailbox via the Graph API,
// upserts them into email_folder_mappings, and attempts to cross-reference each unlinked
// mapping to a bid by matching city + state tokens in the project name.
// Returns the count of folders upserted and any error.
func SyncFolderMappings(db *sql.DB, graphClient *GraphClient, mailbox string) (int, error) {
	if mailbox == "" {
		return 0, fmt.Errorf("mailbox_address not configured — set in email automation settings")
	}

	// -----------------------------------------------------------------------
	// Step 1: Discover folder tree (recursive up to maxFolderDepth)
	// -----------------------------------------------------------------------
	topFolders, err := graphClient.ListMailFolders(mailbox)
	if err != nil {
		return 0, fmt.Errorf("SyncFolderMappings: list top-level folders: %w", err)
	}

	var entries []folderEntry

	for _, tf := range topFolders {
		walkFolder(graphClient, mailbox, tf, "", "", false, 1, &entries)
	}

	// -----------------------------------------------------------------------
	// Step 2: Upsert all discovered folders
	// -----------------------------------------------------------------------
	// project_name and bid_id are NOT overwritten on conflict — staff edits are preserved.
	// On first insert, project_name defaults to folder_name.
	const upsertSQL = `
		INSERT INTO email_folder_mappings
		  (folder_id, folder_name, project_name, folder_path,
		   parent_folder_id, parent_folder_name, is_bid_submitted_sub, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (folder_id) DO UPDATE SET
		  folder_name          = EXCLUDED.folder_name,
		  folder_path          = EXCLUDED.folder_path,
		  parent_folder_id     = EXCLUDED.parent_folder_id,
		  parent_folder_name   = EXCLUDED.parent_folder_name,
		  is_bid_submitted_sub = EXCLUDED.is_bid_submitted_sub,
		  updated_at           = NOW()
		-- project_name and bid_id are intentionally excluded: preserve manual edits`

	count := 0
	for _, e := range entries {
		_, err := db.Exec(upsertSQL,
			e.folder.ID,           // $1 folder_id
			e.folder.DisplayName,  // $2 folder_name
			e.folder.DisplayName,  // $3 project_name (first insert only; conflict leaves it untouched)
			e.folderPath,          // $4 folder_path
			e.folder.ParentFolderID, // $5 parent_folder_id
			e.parentFolderName,    // $6 parent_folder_name
			e.isBidSubmittedSub,   // $7 is_bid_submitted_sub
		)
		if err != nil {
			return count, fmt.Errorf("SyncFolderMappings: upsert folder %q: %w", e.folder.DisplayName, err)
		}
		count++
	}

	// -----------------------------------------------------------------------
	// Step 3: Cross-reference unlinked mappings against active bids
	// -----------------------------------------------------------------------
	type bidRow struct {
		id       int
		location string
		city     string
		state    string
	}

	rows, err := db.Query(`
		SELECT id, location, city, state
		FROM bids
		WHERE archived = false AND city <> '' AND state <> ''`)
	if err != nil {
		return count, fmt.Errorf("SyncFolderMappings: query bids: %w", err)
	}
	defer rows.Close()

	var bids []bidRow
	for rows.Next() {
		var b bidRow
		var city, state sql.NullString
		if err := rows.Scan(&b.id, &b.location, &city, &state); err != nil {
			return count, fmt.Errorf("SyncFolderMappings: scan bid: %w", err)
		}
		b.city = city.String
		b.state = state.String
		if b.city != "" && b.state != "" {
			bids = append(bids, b)
		}
	}
	if err := rows.Err(); err != nil {
		return count, fmt.Errorf("SyncFolderMappings: iterate bids: %w", err)
	}

	// Load mappings that still have no bid_id.
	type unmatchedRow struct {
		id          int
		projectName string
	}
	mRows, err := db.Query(`SELECT id, project_name FROM email_folder_mappings WHERE bid_id IS NULL`)
	if err != nil {
		return count, fmt.Errorf("SyncFolderMappings: query unmatched mappings: %w", err)
	}
	defer mRows.Close()

	var unmatched []unmatchedRow
	for mRows.Next() {
		var m unmatchedRow
		if err := mRows.Scan(&m.id, &m.projectName); err != nil {
			return count, fmt.Errorf("SyncFolderMappings: scan mapping: %w", err)
		}
		unmatched = append(unmatched, m)
	}
	if err := mRows.Err(); err != nil {
		return count, fmt.Errorf("SyncFolderMappings: iterate mappings: %w", err)
	}

	// Match: BOTH the city token AND state token must appear in the normalized project name.
	for _, m := range unmatched {
		normProject := NormalizeFolderName(m.projectName)
		for _, b := range bids {
			normCity := NormalizeFolderName(b.city)
			normState := NormalizeFolderName(b.state)
			if normCity == "" || normState == "" {
				continue
			}
			if strings.Contains(normProject, normCity) && strings.Contains(normProject, normState) {
				_, err := db.Exec(
					`UPDATE email_folder_mappings SET bid_id = $1 WHERE id = $2 AND bid_id IS NULL`,
					b.id, m.id,
				)
				if err != nil {
					log.Printf("[EmailSync] Warning: could not link mapping %d to bid %d: %v", m.id, b.id, err)
				}
				break // use the first matching bid; avoid double-assignment
			}
		}
	}

	log.Printf("[EmailSync] Synced %d folders for mailbox [REDACTED]", count)
	return count, nil
}
