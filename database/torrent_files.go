package database

import (
	"log"
)

func AddFileRecord(fileId, fileName string) (int64, error) {
	query := `INSERT INTO torrent_files (file_id, file_name) VALUES (?, ?);`
	result, err := DB.Exec(query, fileId, fileName)
	if err != nil {
		log.Printf("[AddFileRecord] Помилка при додаванні запису: %v", err)
		return 0, err
	}
	newId, lastInsertErr := result.LastInsertId()
	if lastInsertErr != nil {
		log.Printf("[AddFileRecord] Помилка під час отримання ID для file_id %s: %v", fileId, lastInsertErr)
		return 0, lastInsertErr
	}

	log.Printf("[AddFileRecord] Створено новий запис: ID %d для file_id %s", newId, fileId)
	return newId, nil
}

func GetOrCreateId(fileId, fileName string) (int64, error) {
	query := `SELECT id FROM torrent_files WHERE file_id = ?;`
	row := DB.QueryRow(query, fileId)

	var id int64
	err := row.Scan(&id)
	if err == nil {
		log.Printf("[GetOrCreateId] Знайдено існуючий ID: %d для file_id: %s", id, fileId)
		return id, nil
	}

	if err.Error() != "sql: no rows in result set" {
		log.Printf("[GetOrCreateId] Помилка під час пошуку file_id %s: %v", fileId, err)
		return 0, err
	}

	id, err = AddFileRecord(fileId, fileName)
	if err == nil {
		log.Printf("[GetOrCreateId] Створено ID: %d для file_id: %s", id, fileId)
	}

	return id, nil
}
