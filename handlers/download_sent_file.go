package handlers

import (
	"bytes"
	"fmt"
	"github.com/halushko/kino-cat-core-go/nats_helper"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const TorrentFileSpath = "/root/torrents_to_process/%s_%s"
const TorrentMessageToUser = "(%s) \"%s\"\nВи дійсно хочете завантажити цей торент?\nТак: /start_%s"

func StartGetTorrentFileListener() {
	processor := func(data []byte) {
		userId, fileId, fileName, size, mimeType, fileUrl, err := nats_helper.ParseNatsBotFile(data)
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

			savePath := ""
			messageToUser := ""

			switch {
			case isTorrent(mimeType):
				savePath = fmt.Sprintf(TorrentFileSpath, fileId, fileName)
			}

			if err := downloadFile(fileUrl, savePath); err != nil {
				log.Printf("[StartGetTorrentFileListener] Помилка скачування файлу: %v", err)
				return
			}
			log.Printf("[StartGetTorrentFileListener] Файл \"%s\" вдало збережено у \"%s\"", fileName, savePath)

			switch {
			case isTorrent(mimeType):
				contentSize, contentName, err := getTorrentContentInfo(savePath)
				if err != nil {
					log.Printf("[StartGetTorrentFileListener] ERROR Не вдалося отримати інформацію по торент файлу: %v", err)
				}
				messageToUser = fmt.Sprintf(TorrentMessageToUser, contentSize, contentName, fileId)
			}

			if err = nats_helper.PublishTextMessage("TELEGRAM_OUTPUT_TEXT_QUEUE", userId, messageToUser); err != nil {
				log.Printf("[StartGetTorrentFileListener] Не вдалося надіслати повідомлення \"%s\" через Телеграм бот", messageToUser)
				return
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("[downloadFile] Не вдалося завантажити файл: %s", resp.Status)
	}

	out, err := os.Create(savePath)
	if err != nil {
		return fmt.Errorf("[downloadFile] Помилка створення файлу: %w", err)
	}
	defer func(out *os.File) {
		if e := out.Close(); e != nil {
			log.Printf("[downloadFile] Помилка створення файлу: %v", e)
		}
	}(out)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("[downloadFile] Помилка запису файлу: %w", err)
	}

	return nil
}

func checkMimeType(mimeType string) bool {
	if isTorrent(mimeType) {
		log.Printf("[StartGetTorrentFileListener] Невідомий MIME-тип: %s", mimeType)
		return false
	}
	return true
}

func isTorrent(mimeType string) bool {
	return mimeType == "application/x-bittorrent"
}

func getTorrentContentInfo(pathToTorrentFile string) (string, string, error) {
	cmd := exec.Command("transmission-show", pathToTorrentFile)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("[getTorrentContentInfo] Помилка виконання transmission-show: %v, stderr: %s", err, stderr.String())
	}

	output := out.String()
	lines := strings.Split(output, "\n")
	var size string
	var name string

	for _, line := range lines {
		if strings.HasPrefix(line, "Name:") {
			name = strings.TrimSpace(strings.TrimPrefix(line, "Name:"))
		} else if strings.HasPrefix(line, "Total size:") {
			size = strings.TrimSpace(strings.TrimPrefix(line, "Total size:"))
		}
	}

	if name == "" || size == "" {
		return "", "", fmt.Errorf("[getTorrentContentInfo] Не вдалося дістати інформацію з файлу %s", pathToTorrentFile)
	}

	log.Printf("[getTorrentContentInfo] Отримана інформація - Ім'я: %s, Розмів: %s", name, size)

	return size, name, nil
}
