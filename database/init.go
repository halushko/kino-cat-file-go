package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

var DB *sql.DB

func InitDB(filepath string) {
	var err error
	DB, err = sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("[InitDB] Помилка при підключенні до БД: %v", err)
	}

	createTable(`
		CREATE TABLE IF NOT EXISTS torrent_files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			file_id TEXT NOT NULL UNIQUE,
			file_name TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		)`,
	)
	createTable(`
		CREATE TABLE IF NOT EXISTS files_info (
			id INTEGER PRIMARY KEY,
			path TEXT NOT NULL,
			name TEXT NOT NULL,
			length REAL NOT NULL,
		)`,
	)
}

func createTable(query string) {
	_, err := DB.Exec(query)
	if err != nil {
		log.Fatalf("[InitDB] Помилка при створенні таблиці torrent_files: %v", err)
	}
	log.Println("[InitDB] База даних ініційована.")
}
