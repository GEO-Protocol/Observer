package common

import "time"

// todo: move configuration to the settings file
const (
	// Recommended value for production usage: 1024.
	// Recommended value for development usage: 7.
	ObserversMaxCount = 7

	// Recommended value for production usage: 768.
	// Recommended value for development usage: 5.
	ObserversConsensusCount = 5

	// This period of time is delegated to the observer for block generation and distribution.
	// It is expected, that block generation would be almost timeless,
	// and the rest of time would be spent for the block distribution.
	//
	// Recommended value for production usage: 10 minutes.
	// Recommended value for development usage: 5 seconds.
	AverageBlockGenerationTimeRange = time.Second * 5

	// Time range during which remote observers might respond with their ticker states.
	//
	// WARN!
	// This value must be at least 3 times less,
	// than block generation time range.
	// todo: ensure check of this constraint on ticker starting.
	//
	// Recommended value for production usage: 20 seconds.
	// Recommended value for development usage: 1 second.
	TickerSynchronisationTimeRange   = time.Second // * 20
	ComposerSynchronisationTimeRange = time.Second // * 20

	// This period of time is used as a buffer time window:
	// during this time window observer does not accepts any external events or messages,
	// and prepares to process next ticker tick.
	//
	// This time window MUST be shorter than block generation time range,
	// because it would be subtracted from the AverageBlockGenerationTimeRange.
	//
	// Recommended value for production usage: 20 seconds.
	// Recommended value for development usage: 1 second.
	BlockGenerationSilencePeriod = time.Second // * 20

	// todo: sync with the GEO engine
	GEOTransactionMaxParticipantsCount = 700
)

const (
	TransactionUUIDSize = 16

	Uint16ByteSize = 2
	Uint32ByteSize = 4
	Uint64ByteSize = 8
)
