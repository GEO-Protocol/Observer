// This file contains private requests of composer,
// that are used only for internal logic processing.
package composer

import (
	"geo-observers-blockchain/core/chain/chain"
	"geo-observers-blockchain/core/ticker"
)

// syncRequest represents synchronisation request.
// Used to inform composer that chain synchronisation is required.
type syncRequest struct {
	Chain                  *chain.Chain
	Tick                   *ticker.EventTimeFrameStarted
	TicksChannel           chan *ticker.EventTimeFrameStarted
	LastProcessedEventChan chan *ticker.EventTimeFrameStarted
}
