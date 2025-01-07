package handlers

import (
	"fmt"
	"github.com/halushko/kino-cat-core-go/nats_helper"
	"kino-cat-file-go/database"
	"log"
	"os"
	"strconv"
	"strings"
)

const StartTorrentMessageToUser = "Торент: %s\nРозміром: %.2f Gb передано до торент клієнта"
const StartTorrentsFolder = "/root/torrents_to_download"

func MoveTorrentFileToDownloads() {
	processor := func(data []byte) {
		userId, args, err := nats_helper.ParseNatsBotCommand(data)
		if err != nil {
			log.Printf("[MoveTorrentFileToDownloads] ERROR: %v", err)
			return
		}
		log.Printf("[MoveTorrentFileToDownloads] Отримано команду: \"%v\" з NATS від користувача %d", args, userId)
		fileIdStr := args[0]
		fileId, err := strconv.ParseInt(fileIdStr, 10, 64)
		if err != nil {
			fmt.Printf("[MoveTorrentFileToDownloads] Помилка конвертації ID %s в число: %v", fileIdStr, err)
			return
		}

		filePath, torrentName, torrentLength, err := database.GetFileInfo(fileId)
		if err != nil {
			fmt.Printf("[MoveTorrentFileToDownloads] Помилка отримання інформації по торенту з ID=%s: %v", fileIdStr, err)
			return
		}

		if _, err := os.Stat(StartTorrentsFolder); os.IsNotExist(err) {
			fmt.Printf("[MoveTorrentFileToDownloads] Директорія %s не існує", StartTorrentsFolder)
			return
		}

		fileName := filePath[strings.LastIndex(filePath, "/")+1:]

		destinationPath := fmt.Sprintf("%s/%s", StartTorrentsFolder, fileName)

		err = os.Rename(filePath, destinationPath)
		if err != nil {
			fmt.Printf("[MoveTorrentFileToDownloads] Помилка переміщення файлу %s : %v", StartTorrentsFolder, err)
			return
		}

		fmt.Printf("[MoveTorrentFileToDownloads] Файл успішно переміщено в %s", StartTorrentsFolder)

		message := fmt.Sprintf(StartTorrentMessageToUser, torrentName, torrentLength)
		nats_helper.SendMessageToUser(userId, message)
	}

	listener := &nats_helper.NatsListenerHandler{
		Function: processor,
	}

	if err := nats_helper.StartNatsListener("TELEGRAM_INPUT_FILE_QUEUE", listener); err != nil {
		log.Printf("[StartGetHelpCommandListener] Не вдалося почати роботу над обробкою торент файлів")
	}
}
