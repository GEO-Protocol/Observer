package composer

import (
	"geo-observers-blockchain/core/chain/chain"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/events"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	//"geo-observers-blockchain/core/requests"
	//"geo-observers-blockchain/core/responses"
)

type outgoingRequests struct {
	ChainInfo chan *requests.ChainInfo
}

type incomingResponses struct {
	ChainInfo chan *responses.ChainInfo
}

type outgoingEvents struct {
	SyncFinished chan *events.SyncFinished
}

type Composer struct {
	OutgoingEvents    outgoingEvents
	OutgoingRequests  outgoingRequests
	IncomingResponses incomingResponses

	internalEvents chan interface{}
}

func NewComposer() (composer *Composer) {
	composer = &Composer{
		OutgoingEvents: outgoingEvents{
			SyncFinished: make(chan *events.SyncFinished, 1),
		},

		OutgoingRequests: outgoingRequests{
			ChainInfo: make(chan *requests.ChainInfo, 1),
		},

		IncomingResponses: incomingResponses{
			ChainInfo: make(chan *responses.ChainInfo, settings.ObserversMaxCount),
		},

		internalEvents: make(chan interface{}, 256),
	}
	return
}

func (c *Composer) Run(globalErrorsFlow chan<- error) {
	reportErrorIfAny := func(err errors.E) {
		if err != nil {
			c.logError(err, "")

			globalErrorsFlow <- err.Error()
		}
	}

	for {
		select {
		case event := <-c.internalEvents:

			switch event.(type) {
			case *syncRequest:
				reportErrorIfAny(
					c.processSyncRequest(event.(*syncRequest)))

			default:
				reportErrorIfAny(
					errors.AppendStackTrace(errors.UnexpectedEvent))
			}
		}
	}
}

// SyncChain initialises chain synchronisation with the rest of network.
//
// In case if majority of the observers would agree on some version of the chain (consensus would be reached) -
// current node would synchronise with the majority, even in case if current chain version differs!
//
// In case if consensus can't be reached - incremental synchronisation would be performed.
// Node would iterate through available observers clusters (from bigger one to smallest one)
// and try to sync it's chain with it. In case if chain, proposed by this cluster,
// differs from chain on current node - no sync would be performed, and node would move to the next one.
// No chain sync would be performed in case if no cluster corresponds current chain.
//
// WARN: It is expected that only one sync attempt would be performed at a time.
// 		 Current implementation provides no checks for this case in runtime!
//
// WARN: This method must be called when block producer is stopped.
// 		 Otherwise, chain corruption is possible.
//
func (c *Composer) SyncChain(chain *chain.Chain, tick *ticker.EventTimeFrameStarted,
	ticksChannel chan *ticker.EventTimeFrameStarted) (lastProcessedEventChan chan *ticker.EventTimeFrameStarted) {

	request := &syncRequest{
		Chain:                  chain,
		Tick:                   tick,
		TicksChannel:           ticksChannel,
		LastProcessedEventChan: make(chan *ticker.EventTimeFrameStarted, 10)}

	c.internalEvents <- request
	return request.LastProcessedEventChan
}
