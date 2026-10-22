package database

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
)

//go:embed migrations/postgres/*.sql
var postgresMigrations embed.FS

//go:embed migrations/yugabyte/*.sql
var yugabyteMigrations embed.FS

func RunPostgresMigrations(db *sql.DB) error {
	return runMigrations(db, postgresMigrations, "migrations/postgres")
}

func RunYugabyteMigrations(db *sql.DB) error {
	return runMigrations(db, yugabyteMigrations, "migrations/yugabyte")
}

func runMigrations(db *sql.DB, fs embed.FS, dir string) error {
	entries, err := fs.ReadDir(dir)
	if err != nil {
		return err
	}

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)

	for _, name := range names {
		content, err := fs.ReadFile(dir + "/" + name)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("migration %s/%s: %w", dir, name, err)
		}
		fmt.Printf("Migration applied: %s/%s\n", dir, name)
	}
	return nil
}