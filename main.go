package kino_cat_file_go

import (
	"github.com/halushko/kino-cat-core-go/logger_helper"
	"kino-cat-file-go/handlers"
)

//goland:noinspection ALL
func main() {
	logFile := logger_helper.SoftPrepareLogFile()

	go handlers.StartGetTorrentFileListener()

	select {}
	defer logger_helper.SoftLogClose(logFile)
}
