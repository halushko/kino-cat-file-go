package handlers

import (
	"fmt"
	"github.com/halushko/kino-cat-core-go/nats_helper"
	"io"
	"kino-cat-file-go/database"
	"log"
	"os"
	"path/filepath"
	"strconv"
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
			log.Printf("[MoveTorrentFileToDownloads] Помилка конвертації ID %s в число: %v", fileIdStr, err)
			return
		}

		filePath, torrentName, torrentLength, err := database.GetFileInfo(fileId)
		if err != nil {
			log.Printf("[MoveTorrentFileToDownloads] Помилка отримання інформації по торенту з ID=%s: %v", fileIdStr, err)
			return
		}

		if _, err := os.Stat(StartTorrentsFolder); os.IsNotExist(err) {
			log.Printf("[MoveTorrentFileToDownloads] Директорія %s не існує", StartTorrentsFolder)
			return
		}

		fileName := filepath.Base(filePath)
		destinationPath := filepath.Join(StartTorrentsFolder, fileName)

		err = moveFile(filePath, destinationPath)
		if err != nil {
			log.Printf("[MoveTorrentFileToDownloads] Помилка переміщення файлу: %v", err)
			return
		}

		log.Printf("[MoveTorrentFileToDownloads] Файл успішно переміщено в %s", destinationPath)

		message := fmt.Sprintf(StartTorrentMessageToUser, torrentName, torrentLength)
		nats_helper.SendMessageToUser(userId, message)
	}

	listener := &nats_helper.NatsListenerHandler{
		Function: processor,
	}

	if err := nats_helper.StartNatsListener("FILE_MOVE_TO_FOLDER", listener); err != nil {
		log.Printf("[StartGetHelpCommandListener] Не вдалося почати роботу над обробкою торент файлів")
	}
}

func moveFile(sourcePath, destinationPath string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("[moveFile] Помилка відкриття файлу: %w", err)
	}
	defer sourceFile.Close()

	destinationFile, err := os.Create(destinationPath)
	if err != nil {
		return fmt.Errorf("[moveFile] Помилка створення файлу: %w", err)
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return fmt.Errorf("[moveFile] Помилка копіювання файлу: %w", err)
	}

	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("[moveFile] Помилка видалення оригінального файлу: %w", err)
	}

	return nil
}
