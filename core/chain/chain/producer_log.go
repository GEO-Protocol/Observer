package chain

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	log "github.com/sirupsen/logrus"
)

func (p *Producer) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Producer"})
}

func (p *Producer) logErrorIfAny(e errors.E) {
	if e == nil {
		return
	}

	message := e.Error().Error()

	p.log().WithFields(log.Fields{
		"StackTrace": e.StackTrace(),
	}).Error(message)
}

func (p *Producer) logCurrentTimeFrameIndex(frame *ticker.EventTimeFrameStarted) {
	if settings.Conf.Debug {
		p.log().WithFields(
			log.Fields{"Index": frame.Index}).Debug("Time frame changed")
	}
}
