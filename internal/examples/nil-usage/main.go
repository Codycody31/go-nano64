package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"go.codycody31.dev/nano64"
	_ "modernc.org/sqlite"
)

type User struct {
	ID       nano64.Nano64
	Name     string
	ParentID nano64.NullNano64 // Optional parent reference
}

func main() {
	fmt.Println("=== Nano64 Nil Usage Examples ===")

	// Example 1: Using the Nil constant
	fmt.Println("1. Nil Constant:")
	fmt.Printf("   Nil value: %d\n", nano64.Nil.Uint64Value())
	fmt.Printf("   Nil hex: %s\n", nano64.Nil.ToHex())
	fmt.Printf("   Is nil: %v\n\n", nano64.Nil.IsNil())

	// Example 2: Checking if an ID is nil
	fmt.Println("2. Checking for Nil IDs:")
	var uninitializedID nano64.Nano64
	generatedID, _ := nano64.GenerateDefault()

	fmt.Printf("   Uninitialized ID is nil: %v\n", uninitializedID.IsNil())
	fmt.Printf("   Generated ID is nil: %v\n\n", generatedID.IsNil())

	// Example 3: NullNano64 for optional fields
	fmt.Println("3. NullNano64 for Optional Fields:")

	// Root user with no parent
	rootUser := User{
		ID:       generatedID,
		Name:     "Root User",
		ParentID: nano64.NullNano64{Valid: false}, // No parent
	}
	fmt.Printf("   Root user: %s (Parent: %v)\n", rootUser.Name, rootUser.ParentID.Valid)

	// Child user with a parent
	childID, _ := nano64.GenerateDefault()
	childUser := User{
		ID:       childID,
		Name:     "Child User",
		ParentID: nano64.NullNano64{ID: generatedID, Valid: true},
	}
	fmt.Printf("   Child user: %s (Parent: %v)\n\n", childUser.Name, childUser.ParentID.Valid)

	// Example 4: JSON marshaling with null values
	fmt.Println("4. JSON Marshaling:")

	rootJSON, _ := json.MarshalIndent(rootUser, "   ", "  ")
	fmt.Printf("   Root user JSON:\n   %s\n\n", string(rootJSON))

	childJSON, _ := json.MarshalIndent(childUser, "   ", "  ")
	fmt.Printf("   Child user JSON:\n   %s\n\n", string(childJSON))

	// Example 5: Database usage with null values
	fmt.Println("5. Database Usage with Null Values:")
	if err := databaseExample(); err != nil {
		log.Printf("   Database example error: %v\n", err)
	}
}

func databaseExample() error {
	// Create temporary database
	tempDir, err := os.MkdirTemp("", "nano64-example-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	dbPath := filepath.Join(tempDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// Create table with nullable parent_id
	_, err = db.Exec(`
		CREATE TABLE users (
			id BLOB PRIMARY KEY,
			name TEXT NOT NULL,
			parent_id BLOB
		)
	`)
	if err != nil {
		return err
	}

	// Insert root user (no parent)
	rootID, _ := nano64.GenerateDefault()
	_, err = db.Exec(
		"INSERT INTO users (id, name, parent_id) VALUES (?, ?, ?)",
		rootID,
		"Root User",
		nano64.NullNano64{Valid: false},
	)
	if err != nil {
		return err
	}
	fmt.Println("   ✓ Inserted root user with NULL parent")

	// Insert child user (with parent)
	childID, _ := nano64.GenerateDefault()
	_, err = db.Exec(
		"INSERT INTO users (id, name, parent_id) VALUES (?, ?, ?)",
		childID,
		"Child User",
		nano64.NullNano64{ID: rootID, Valid: true},
	)
	if err != nil {
		return err
	}
	fmt.Println("   ✓ Inserted child user with parent reference")

	// Query and display results
	rows, err := db.Query("SELECT id, name, parent_id FROM users")
	if err != nil {
		return err
	}
	defer rows.Close()

	fmt.Println("\n   Retrieved users:")
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Name, &user.ParentID); err != nil {
			return err
		}

		if user.ParentID.Valid {
			fmt.Printf("     - %s (ID: %s, Parent: %s)\n",
				user.Name,
				user.ID.ToHex(),
				user.ParentID.ID.ToHex())
		} else {
			fmt.Printf("     - %s (ID: %s, Parent: NULL)\n",
				user.Name,
				user.ID.ToHex())
		}
	}

	return rows.Err()
}
