package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/halushko/kino-cat-core-go/nats_helper"
	"io"
	"log"
	"net/http"
	"os"
)

const TgBotApiGetFile = "https://api.telegram.org/bot%s/getFile?file_id=%s"
const TgBotApiDownload = "https://api.telegram.org/file/bot%s/%s"
const TorrentFilesPath = "/app/torrent_files/%s"

type getFileResponse struct {
	Ok     bool `json:"ok"`
	Result struct {
		FilePath string `json:"file_path"`
	} `json:"result"`
}

func StartGetTorrentFileListener() {
	processor := func(data []byte) {
		userId, fileId, fileName, size, mimeType, err := nats_helper.ParseNatsBotFile(data)
		if err != nil {
			log.Printf("[StartGetTorrentFileListener] ERROR: %v", err)
			return
		}
		log.Printf("[StartGetTorrentFileListener] Отримано файл: \"%s\" (%s) з NATS від користувача %d розміром %d", fileName, fileId, userId, size)

		if userId != 0 && checkMimeType(mimeType) {
			botToken := os.Getenv("BOT_TOKEN")
			if botToken == "" {
				log.Printf("[StartGetTorrentFileListener] BOT_TOKEN не задано")
				return
			}

			filePath, err := getFilePathFromTelegram(botToken, fileId)
			if err != nil {
				log.Printf("[StartGetTorrentFileListener] Помилка отримання шляху до файлу: %v", err)
				return
			}

			fileURL := fmt.Sprintf(TgBotApiDownload, botToken, filePath)
			savePath := fmt.Sprintf(TorrentFilesPath, fileName)

			if err := downloadFile(fileURL, savePath); err != nil {
				log.Printf("[StartGetTorrentFileListener] Помилка скачування файлу: %v", err)
				return
			}

			log.Printf("[StartGetTorrentFileListener] Файл \"%s\" вдало збережено у \"%s\"", fileName, savePath)
		}
	}

	listener := &nats_helper.NatsListenerHandler{
		Function: processor,
	}

	if err := nats_helper.StartNatsListener("TELEGRAM_INPUT_FILE_QUEUE", listener); err != nil {
		log.Printf("[StartGetHelpCommandListener] Не вдалося почати роботу над обробкою торент файлів")
	}
}

func getFilePathFromTelegram(botToken, fileId string) (string, error) {
	url := fmt.Sprintf(TgBotApiGetFile, botToken, fileId)

	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("Помилка виконання запиту getFile: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Неправильний статус відповіді getFile: %s", resp.Status)
	}

	var result getFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("Помилка розбору відповіді getFile: %w", err)
	}

	if !result.Ok {
		return "", fmt.Errorf("getFile повернув помилку")
	}

	return result.Result.FilePath, nil
}

func downloadFile(url string, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("[downloadFile] Помилка завантаження файлу: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[downloadFile] Не вдалося завантажити файл: %s", resp.Status)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("[downloadFile] Помилка створення файлу: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("[downloadFile] Помилка запису файлу: %w", err)
	}

	return nil
}

func checkMimeType(mimeType string) bool {
	if mimeType != "application/x-bittorrent" {
		log.Printf("[StartGetTorrentFileListener] Невідомий MIME-тип: %s", mimeType)
		return false
	}
	return true
}
