package logger

import (
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
	"github.com/x-cray/logrus-prefixed-formatter"
	"os"
)

func InitLogger() *os.File {
	logfile, err := os.OpenFile("operations.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Fatal(err)
	}

	if !settings.Conf.Debug {
		log.SetOutput(logfile)
		log.SetFormatter(&log.JSONFormatter{})

	} else {
		log.SetLevel(log.TraceLevel)
		log.SetFormatter(&prefixed.TextFormatter{
			FullTimestamp: true,
		})
	}
	return logfile
}
