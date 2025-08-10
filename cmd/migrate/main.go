package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/daccred/sorobangraph.attest.so/db"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run cmd/migrate/main.go <up|down|status>")
	}

	command := os.Args[1]
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgres://user:password@localhost/stellar_ingester?sslmode=disable"
	}

	dbConn, err := db.Connect(databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbConn.Close()

	switch command {
	case "up":
		if err := runMigrations(dbConn); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully!")
	case "status":
		if err := dbConn.Ping(); err != nil {
			log.Fatalf("Database connection failed: %v", err)
		}
		fmt.Println("Database connection successful!")
	default:
		log.Fatal("Unknown command. Use 'up' or 'status'")
	}
}

func runMigrations(dbConn *sql.DB) error {
	migrationsDir := "migrations"

	// Get all SQL files in migrations directory
	files, err := filepath.Glob(filepath.Join(migrationsDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	for _, file := range files {
		fmt.Printf("Running migration: %s\n", file)

		content, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", file, err)
		}

		if _, err := dbConn.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", file, err)
		}
	}

	return nil
}
