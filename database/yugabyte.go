package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

func ConnectYugabyte() *sql.DB {
	host := os.Getenv("YUGABYTE_HOST")
	port := os.Getenv("YUGABYTE_PORT")
	user := os.Getenv("YUGABYTE_USER")
	password := os.Getenv("YUGABYTE_PASSWORD")
	dbname := os.Getenv("YUGABYTE_DB")
	sslmode := os.Getenv("YUGABYTE_SSLMODE")
	sslrootcert := os.Getenv("YUGABYTE_SSLROOTCERT")

	// Check required configuration
	if host == "" || port == "" || user == "" ||
		dbname == "" || sslmode == "" {
		fmt.Println("Yugabyte connection error: missing environment variables")
		return nil
	}

	// Build connection string
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s sslmode=%s connect_timeout=10",
		host,
		port,
		user,
		dbname,
		sslmode,
	)

	// Add password only if provided
	if password != "" {
		connStr += fmt.Sprintf(" password=%s", password)
	}

	// Add SSL certificate only if provided
	if sslrootcert != "" {
		connStr += fmt.Sprintf(" sslrootcert=%s", sslrootcert)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Yugabyte connection error:", err)
		return nil
	}

	// Test connection with a timeout
	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		fmt.Printf("Yugabyte ping failed: %T: %v\n", err, err)

		if closeErr := db.Close(); closeErr != nil {
			fmt.Println("Error closing Yugabyte connection:", closeErr)
		}

		return nil
	}

	fmt.Println("YugabyteDB connected successfully")
	return db
}