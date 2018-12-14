package timer

import "geo-observers-blockchain/core/network/external"

type EventTimeFrameEnd struct {
	// Index of the current time frame.
	// Minimal value = 0
	// Maximal value = max. observers count - 1.
	Index uint16

	Configuration *external.Configuration
}

type EventTickerStarted struct{}
