package main

import (
	"fmt"
	"net/http"
	"os"

	"data-sync/database"
	"data-sync/routes"

	"github.com/joho/godotenv"

	"context"
	"os/signal"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Println("Error loading .env:", err)
		return
	}

	// Connect to PostgreSQL
	postgresDB := database.Connect()
	if postgresDB == nil {
		fmt.Println("Failed to connect to PostgreSQL")
		return
	}
	defer postgresDB.Close()

	// Connect to local YugabyteDB
	yugabyteDB := database.ConnectYugabyte()
	if yugabyteDB == nil {
		fmt.Println("Failed to connect to YugabyteDB")
		return
	}
	defer yugabyteDB.Close()

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
	)
	defer cancel()

	go database.StartOutboxWorker(
		ctx,
		postgresDB,
		yugabyteDB,
	)
	// Configure API routes
	// Currently, routes use PostgreSQL only.
	routes.SetupRoutes(postgresDB, yugabyteDB)

	// Start HTTP server
	fmt.Println("Data Sync Backend Started")
	fmt.Println("Both databases connected successfully")
	fmt.Println("Server running on http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
