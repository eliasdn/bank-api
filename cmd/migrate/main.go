package main

import (
	"bank-api/internal/db"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	var action string
	flag.StringVar(&action, "action", "up", "Migration action: up, down, reset")
	flag.Parse()

	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	dbPath := "bank.db"
	if envPath := os.Getenv("DB_PATH"); envPath != "" {
		dbPath = envPath
	}

	switch action {
	case "up":
		fmt.Println("Running migrations...")
		if err := db.RunMigrations(dbPath); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations completed successfully")
	case "down":
		fmt.Println("Rolling back migrations...")
		// For now, we'll just drop all tables as SQLite doesn't support full down migrations
		if err := resetDatabase(dbPath); err != nil {
			log.Fatalf("Failed to reset database: %v", err)
		}
		fmt.Println("Database reset completed")
	case "reset":
		fmt.Println("Resetting database...")
		if err := resetDatabase(dbPath); err != nil {
			log.Fatalf("Failed to reset database: %v", err)
		}
		if err := db.RunMigrations(dbPath); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Database reset and migrations completed")
	default:
		fmt.Printf("Unknown action: %s\n", action)
		fmt.Println("Usage: migrate [up|down|reset]")
		os.Exit(1)
	}
}

func resetDatabase(dbPath string) error {
	// Remove the database file to reset
	if _, err := os.Stat(dbPath); err == nil {
		return os.Remove(dbPath)
	}
	return nil
}
