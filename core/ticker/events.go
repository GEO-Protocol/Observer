package ticker

import (
	"geo-observers-blockchain/core/network/external"
	"time"
)

// EventTimeFrameEnd is emitted each time when next block frame is closing.
type EventTimeFrameEnd struct {
	// Index of the current time frame.
	// Minimal value = 0
	// Maximal value = max. observers count - 1.
	Index uint16

	Conf *external.Configuration

	// See documentation for constants.BlockGenerationSilencePeriod.
	FinalStageTimestamp time.Time
}

// EventTickerStarted is emitted each time when internal ticker ticker is started,
// for example (when synchronisation is finished).
type EventTickerStarted struct{}
