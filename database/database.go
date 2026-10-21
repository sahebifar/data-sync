package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Connect با استفاده از متغیرهای محیطی به PostgreSQL وصل می‌شود.
// مقادیر از فایل .env خوانده می‌شوند (نه داخل کد).
func Connect() *sql.DB {
	host := os.Getenv("POSTGRES_HOST")
	port := os.Getenv("POSTGRES_PORT")
	user := os.Getenv("POSTGRES_USER")
	password := os.Getenv("POSTGRES_PASSWORD")
	dbname := os.Getenv("POSTGRES_DB")

	if host == "" || port == "" || user == "" || dbname == "" {
		fmt.Println("PostgreSQL connection error: missing environment variables")
		return nil
	}

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s dbname=%s sslmode=disable connect_timeout=10",
		host, port, user, dbname,
	)

	if password != "" {
		connStr += fmt.Sprintf(" password=%s", password)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("PostgreSQL connection error:", err)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		fmt.Printf("PostgreSQL ping failed: %T: %v\n", err, err)
		if closeErr := db.Close(); closeErr != nil {
			fmt.Println("Error closing PostgreSQL connection:", closeErr)
		}
		return nil
	}

	fmt.Println("PostgreSQL connected successfully")
	return db
}
