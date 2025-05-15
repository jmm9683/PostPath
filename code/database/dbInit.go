package database

import (
	"database/sql"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func InitDB(dataSourceName string) {
	var err error
	db, err = sql.Open("sqlite3", dataSourceName)
	if err != nil {
		log.Fatal(err)
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
		username TEXT NOT NULL,
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
		title TEXT NOT NULL
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
		FOREIGN KEY(page_id) REFERENCES pages(id),
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
}

func modifyPageTextTable() {
	_, err := db.Exec(`ALTER TABLE pagetext ADD COLUMN source INTEGER`)
	if err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		log.Fatal(err)
	}
}
