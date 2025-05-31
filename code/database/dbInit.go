package database

import (
	"context"
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB(dataSourceName string) {

	var err error
	db, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Fatal(err)
	}
	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err = db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	createUserTable()
	createPageTables()
	modifyPageTextTable()
}

func DB() *sql.DB {
	return db
}

func createUserTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);`
	if _, err := db.Exec(query); err != nil {
		log.Fatal(err)
	}
}

func createPageTables() {
	pageTable := `
	CREATE TABLE IF NOT EXISTS pages (
		id INTEGER PRIMARY KEY,
		title TEXT NOT NULL UNIQUE
	);`
	if _, err := db.Exec(pageTable); err != nil {
		log.Fatal(err)
	}

	pageTextTable := `
	CREATE TABLE IF NOT EXISTS pagetext (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		page_id INTEGER NOT NULL,
		user_id INTEGER NOT NULL,
		text TEXT NOT NULL,
		is_link INTEGER DEFAULT 0,
		link_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		is_edited INTEGER DEFAULT 0,
		path TEXT,
		source INTEGER,
		FOREIGN KEY(page_id) REFERENCES pages(id),
		FOREIGN KEY(link_id) REFERENCES pages(id),
		FOREIGN KEY(source) REFERENCES pages(id),
		FOREIGN KEY(user_id) REFERENCES users(id)
	);`
	if _, err := db.Exec(pageTextTable); err != nil {
		log.Fatal(err)
	}

	// Insert Home page if it doesn't exist
	_, err := db.Exec(`INSERT OR IGNORE INTO pages (id, title) VALUES (0, 'Home')`)
	if err != nil {
		log.Fatal(err)
	}
	// Insert Home page if it doesn't exist
	_, err = db.Exec(`INSERT OR IGNORE INTO pages (id, title) VALUES (1, 'Profile')`)
	if err != nil {
		log.Fatal(err)
	}
}

func modifyPageTextTable() {
	_, err := db.Exec(`ALTER TABLE pagetext ADD COLUMN source INTEGER`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		log.Fatal(err)
	}
}

// Add this function to handle timeouts
func QueryWithTimeout(query string, args ...interface{}) (*sql.Rows, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		cancel() // safe to cancel if there's an error
		return nil, nil, err
	}
	return rows, cancel, nil
}

// Returns sql.Row and a cancel function the caller must defer
func QueryRowWithTimeout(query string, args ...interface{}) (*sql.Row, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	return db.QueryRowContext(ctx, query, args...), cancel
}

// Returns sql.Result and a cancel function the caller must defer
func ExecWithTimeout(query string, args ...interface{}) (sql.Result, context.CancelFunc, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		cancel() // cancel early if failed
		return nil, nil, err
	}
	return result, cancel, nil
}
