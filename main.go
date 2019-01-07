package main

import (
	"geo-observers-blockchain/core"
	"geo-observers-blockchain/core/logger"
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
)

func main() {
	PrintLogo()
	PrintVersionDigest()

	conf, err := settings.LoadSettings()
	if err != nil {
		log.Fatal(err)
	}

	logfile := logger.InitLogger(conf)
	defer logfile.Close()

	c, err := core.New(conf)
	if err != nil {
		log.Fatal(err)
	}

	c.Run()
}
