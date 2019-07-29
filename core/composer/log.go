package composer

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
)

func (c *Composer) logError(err errors.E, message string) {
	if settings.Conf.Debug {
		c.log().WithFields(log.Fields{
			"Message":    err.Error(),
			"StackTrace": err.StackTrace(),
		}).Error(message)

	} else {
		c.log().Error(message, err.Error())

	}
}

func (c *Composer) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Composer"})
}
