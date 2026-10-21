package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env: ", err)
	}

	host := os.Getenv("YUGABYTE_HOST")
	port := os.Getenv("YUGABYTE_PORT")
	user := os.Getenv("YUGABYTE_USER")
	password := os.Getenv("YUGABYTE_PASSWORD")
	dbname := os.Getenv("YUGABYTE_DB")
	sslmode := os.Getenv("YUGABYTE_SSLMODE")
	rootCert := os.Getenv("YUGABYTE_SSLROOTCERT")

	if host == "" || port == "" || user == "" ||
		dbname == "" || sslmode == "" {
		log.Fatal("Missing Yugabyte environment variables")
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s sslmode=%s connect_timeout=10",
		host, port, user, dbname, sslmode,
	)

	if password != "" {
		connStr += fmt.Sprintf(" password=%s", password)
	}

	if rootCert != "" {
		connStr += fmt.Sprintf(" sslrootcert=%s", rootCert)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal("Open error: ", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Fatalf("Connection test failed: %T: %v", err, err)
	}

	fmt.Println("Successfully connected to YugabyteDB!")

	var version string
	if err := db.QueryRowContext(ctx, "SELECT version()").Scan(&version); err != nil {
		log.Fatal("Version query failed: ", err)
	}

	fmt.Println("Database version:", version)

	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
		log.Fatal("SELECT 1 failed: ", err)
	}

	fmt.Println("Test query result:", result)
}
