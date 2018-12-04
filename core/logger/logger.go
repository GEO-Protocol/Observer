package logger

import (
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
	"os"
)

func InitLogger(conf *settings.Settings) *os.File {
	logfile, err := os.OpenFile("operations.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	if !conf.Debug {
		log.SetOutput(logfile)
		log.SetFormatter(&log.JSONFormatter{})

	} else {
		log.SetFormatter(&log.TextFormatter{
			FullTimestamp: true,
		})
	}
	return logfile
}
