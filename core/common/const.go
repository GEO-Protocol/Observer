package common

import "time"

const (
	ObserversMaxCount       = 4 // todo: fix me to 1024
	ObserversConsensusCount = 2 // todo: fix me to 768

	// This period of time is delegated to the observer for block generation and distribution.
	// It is expected, that block generation would be almost timeless,
	// and the rest of time would be spent for the block distribution.
	//
	// Recommended value for production usage: 10 minutes.
	// Recommended value for development usage: 2 seconds.
	AverageBlockGenerationTimeRange = time.Second * 5 // todo: fix me to time.Hour

	// This period of time is used as a buffer time window:
	// during this time window observer does not accepts any external events or messages,
	// and prepares to process next timer tick.
	//
	// This time window MUST be shorter than block generation time range,
	// because it would be subtracted from the AverageBlockGenerationTimeRange.
	//
	// Recommended value for production usage: 20 seconds.
	// Recommended value for development usage: 1 second.
	BlockGenerationSilencePeriod = time.Second

	// todo: sync with the GEO engine
	GEOTransactionMaxParticipantsCount = 700
)

const (
	TransactionUUIDSize = 16

	Uint16ByteSize = 2
	Uint32ByteSize = 4
	Uint64ByteSize = 8
)
