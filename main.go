package main

import (
	"github.com/halushko/kino-cat-core-go/logger_helper"
	"kino-cat-file-go/database"
	"kino-cat-file-go/handlers"
)

const dbPath = "/data/kino_cat.db"

//goland:noinspection ALL
func main() {
	logFile := logger_helper.SoftPrepareLogFile()

	database.InitDB(dbPath)

	go handlers.StartGetTorrentFileListener()
	go handlers.MoveTorrentFileToDownloads()

	select {}
	defer logger_helper.SoftLogClose(logFile)
}
