package database

import "log"

func AddFileLocation(id int64, path string, name string, length float64) error {
	query := `INSERT INTO files_info (id, path, name, length)
				VALUES (?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET
					path = excluded.path
				    name = excluded.name
				    length = excluded.length;`
	_, err := DB.Exec(query, id, path, name, length)
	if err != nil {
		log.Printf("[AddFileLocation] Помилка при додаванні запису: %v", err)
		return err
	}

	log.Printf("[AddFileLocation] Створено новий запис: ID %d для Name %s , Path: %s розміром %f", id, name, path, length)
	return nil
}

func GetFileInfo(id int64) (string, string, float64, error) {
	query := `SELECT path FROM files_info WHERE id = ?;`
	row := DB.QueryRow(query, id)

	var path string
	var name string
	var length float64

	err := row.Scan(&path, &name)
	if err == nil {
		log.Printf("[GetFileLocation] Знайдено існуючий ID: %d зі шляхом: %s на ім'я: %s розміром: %f", id, path, name, length)
		return path, name, length, nil
	} else {
		log.Printf("[GetFileLocation] Помилка під час пошуку шляху для файла ID: %d: %v", id, err)
		return "", "", -1.0, err
	}
}
