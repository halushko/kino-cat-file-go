package handlers

import (
	"fmt"
	"github.com/halushko/kino-cat-core-go/nats_helper"
	"github.com/zeebo/bencode"
	"io"
	"kino-cat-file-go/database"
	"log"
	"net/http"
	"os"
)

type Torrent struct {
	Info struct {
		Name   string `bencode:"name"`
		Length int64  `bencode:"length"`
		Files  []struct {
			Length int64  `bencode:"length"`
			Path   string `bencode:"path"`
		} `bencode:"files"`
	} `bencode:"info"`
}

const TorrentFileSpath = "/root/torrents_to_process/%s_%d"
const TorrentMessageToUser = "(%.2f Gb) \"%s\"\nВи дійсно хочете завантажити цей торент?\nТак: /start_%d"

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
			var id int64
			id = 0

			switch {
			case isTorrent(mimeType):
				id, err = database.GetOrCreateId(fileId, fileName)
				if err != nil {
					log.Printf("[StartGetTorrentFileListener] ERROR: %v", err)
					return
				}
				savePath = fmt.Sprintf(TorrentFileSpath, id, fileName)
			}

			if err := downloadFile(fileUrl, savePath); err != nil {
				log.Printf("[StartGetTorrentFileListener] Помилка скачування файлу: %v", err)
				return
			}
			log.Printf("[StartGetTorrentFileListener] Файл \"%s\" вдало збережено у \"%s\"", fileName, savePath)

			switch {
			case isTorrent(mimeType):
				contentSize, contentName, manyFiles, err := getTorrentContentInfo(savePath)
				if err != nil {
					log.Printf("[StartGetTorrentFileListener] ERROR Не вдалося отримати інформацію по торент файлу: %v", err)
				}
				messageToUser = fmt.Sprintf(TorrentMessageToUser, contentSize, contentName, id)
				if manyFiles {
					messageToUser = messageToUser + fmt.Sprintf("\nПереглянути файли: /expand_%d", id)
				}
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
	if !isTorrent(mimeType) {
		log.Printf("[StartGetTorrentFileListener] Невідомий MIME-тип: %s", mimeType)
		return false
	}
	return true
}

func isTorrent(mimeType string) bool {
	return mimeType == "application/x-bittorrent"
}

func getTorrentContentInfo(pathToTorrentFile string) (float64, string, bool, error) {
	file, err := os.Open(pathToTorrentFile)
	if err != nil {
		fmt.Printf("[getTorrentContentInfo] Помилка відкриття файлу: %v\n", err)
		return 0, "", false, err
	}
	defer file.Close()

	var torrent Torrent
	if err := bencode.NewDecoder(file).Decode(&torrent); err != nil {
		fmt.Printf("[getTorrentContentInfo] Помилка розбору файлу: %v\n", err)
		return 0, "", false, err
	}

	flag := false
	size := -1.0
	name := "noname"
	fmt.Printf("[getTorrentContentInfo] INFO: %v", torrent)

	if len(torrent.Info.Files) > 0 {
		flag = true
		for _, f := range torrent.Info.Files {
			fmt.Printf("[getTorrentContentInfo] Файл: %s, Розмір: %d байт\n", f.Path, f.Length)
		}
	} else {
		fmt.Printf("[getTorrentContentInfo] Файл: %s, Розмір: %d байт\n", torrent.Info.Name, torrent.Info.Length)
		size = float64(torrent.Info.Length) / (1024 * 1024 * 1024)
		name = torrent.Info.Name
	}

	return size, name, flag, nil
}
