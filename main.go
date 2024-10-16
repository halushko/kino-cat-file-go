package kino_cat_file_go

import (
	"github.com/halushko/kino-cat-core-go/logger_helper"
)

//goland:noinspection ALL
func main() {
	logFile := logger_helper.SoftPrepareLogFile()

	defer logger_helper.SoftLogClose(logFile)
}
