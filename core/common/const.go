package common

import "time"

const (
	ObserversMaxCount               = 4                // todo: fix me to 1024
	ObserversConsensusCount         = 3                // todo: fix me to 768
	AverageBlockGenerationTimeRange = time.Second * 45 // todo: fix me to time.Hour

	// todo: sync with the GEO engine
	GeoTransactionMaxParticipantsCount = 1024 * 32
)
