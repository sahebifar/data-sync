package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"

	"data-sync/database"
	"data-sync/routes"

	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("No .env file found, using environment variables")
	}

	postgresDB := database.Connect()
	if postgresDB == nil {
		fmt.Println("Failed to connect to PostgreSQL")
		return
	}
	defer postgresDB.Close()

	yugabyteDB := database.ConnectYugabyte()
	if yugabyteDB == nil {
		fmt.Println("Failed to connect to YugabyteDB")
		return
	}
	defer yugabyteDB.Close()

	// Run migrations on both databases.
	if err := database.RunPostgresMigrations(postgresDB); err != nil {
		fmt.Println("PostgreSQL migrations failed:", err)
		return
	}
	if err := database.RunYugabyteMigrations(yugabyteDB); err != nil {
		fmt.Println("YugabyteDB migrations failed:", err)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go database.StartOutboxWorker(ctx, postgresDB, yugabyteDB)

	routes.SetupRoutes(postgresDB, yugabyteDB)

	fmt.Println("Data Sync Backend Started")
	fmt.Println("Both databases connected successfully")
	fmt.Println("Server running on http://localhost:8080")

	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Println("Server error:", err)
	}
}
