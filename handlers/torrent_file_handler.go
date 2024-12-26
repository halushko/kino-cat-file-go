package handlers

import (
	"fmt"
	"github.com/halushko/kino-cat-core-go/nats_helper"
	"io"
	"log"
	"net/http"
	"os"
)

const TgBotApi = "https://api.telegram.org/file/bot%s/%s"
const TorrentFilesPath = "/app/torrent_files/%s"

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
			fileURL := fmt.Sprintf(TgBotApi, botToken, fileId)
			savePath := ""

			switch {
			case mimeType == "application/x-bittorrent":
				savePath = fmt.Sprintf(TorrentFilesPath, fileName)
			case mimeType == "application/pdf":
				savePath = ""
			}

			if savePath != "" {
				if err := downloadFile(fileURL, savePath); err != nil {
					log.Printf("[StartGetTorrentFileListener] Помилка скачування файлу: %v", err)
					return
				}

				log.Printf("[StartGetTorrentFileListener] Файл \"%s\" вдало збережено у \"%s\"", fileName, savePath)
			} else {
				log.Printf("[StartGetTorrentFileListener] Директорію для файлу типу \"%s\" не задано", mimeType)
			}
		}
	}

	listener := &nats_helper.NatsListenerHandler{
		Function: processor,
	}

	if err := nats_helper.StartNatsListener("TELEGRAM_INPUT_FILE_QUEUE", listener); err != nil {
		log.Printf("[StartGetHelpCommandListener] Не вдалося почати роботу над обробкою торент файлів")
	}
}

func downloadFile(url string, savePath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("[downloadFile] Помилка завантаження файлу: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[downloadFile] Не вдалося завантажити файл: %s", resp.Status)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("[downloadFile] Помилка створення файлу: %w", err)
	}
	//goland:noinspection GoUnhandledErrorResult
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
