package ticker

import (
	"geo-observers-blockchain/core/network/external"
	"time"
)

// EventTimeFrameStarted is emitted each time when next block frame is closing.
type EventTimeFrameStarted struct {
	// Index of the current time frame.
	// Minimal value = 0
	// Maximal value = max. observers count - 1.
	Index uint16

	ObserversConfiguration *external.Configuration

	// Timestamp when validation stage must be finalized.
	// WARN: stage finalizing takes some time, so this stage should be ~3 seconds earlier than time frame end.
	ValidationStageEndTimestamp time.Time

	// Timestamp when block generation stage must be finalized.
	// WARN: after this stage newly generated block broadcasting occur,
	// so this stage should be shorter than ValidationStage,
	// to ensure committed blocks transferring through the network.
	BlockGenerationStageEndTimestamp time.Time
}

// eventTickerStarted is emitted each time when internal ticker is started,
// for example when synchronisation is finished.
type eventTickerStarted struct{}

// eventSyncRequested is emitted each time when internal ticker synchronisation is needed.
type eventSyncRequested struct{}
