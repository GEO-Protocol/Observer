package chain

import (
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/pool"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/crypto/keystore"
	geoRequests "geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
)

// ToDo: Tests
//       * Blocks collisions;
//       * 2 concurrent blocks collected relatively equal amount of approves;

// Improve:
//   * Consider including observers signatures under the block hash;
//   * Prevent claims/TSLs collisions (check if no equal claim/TSLs is already present in chain);
//   * Inform remote observers about collisions detected (force chains/tickers synchronisation);

// Producer generates new blocks and validates info received from remote observers.
type Producer struct {
	// Observers interface
	OutgoingRequestsCandidateDigestBroadcast chan *requests.CandidateDigestBroadcast
	IncomingRequestsCandidateDigest          chan *requests.CandidateDigestBroadcast
	OutgoingResponsesCandidateDigestApprove  chan *responses.CandidateDigestApprove
	IncomingResponsesCandidateDigestApprove  chan *responses.CandidateDigestApprove
	OutgoingResponsesCandidateDigestReject   chan *responses.CandidateDigestReject
	IncomingResponsesCandidateDigestReject   chan *responses.CandidateDigestReject
	OutgoingRequestsBlockSignaturesBroadcast chan *requests.BlockSignaturesBroadcast
	IncomingRequestsBlockSignatures          chan *requests.BlockSignaturesBroadcast
	IncomingRequestsChainTop                 chan *requests.ChainInfo
	OutgoingResponsesChainTop                chan *responses.ChainInfo
	OutgoingRequestsTimeFrameCollisions      chan *requests.TimeFrameCollision // todo: implement this mechanics
	OutgoingRequestsBlockHashBroadcast       chan *requests.BlockHashBroadcast
	IncomingRequestsBlockHashBroadcast       chan *requests.BlockHashBroadcast
	OutgoingRequestsCollisionNotifications   chan *requests.CollisionNotification
	IncomingRequestsCollisionNotifications   chan *requests.CollisionNotification

	// GEO Node interface
	GEORequestsLastBlockHeight chan *geoRequests.LastBlockNumber
	GEORequestsClaimIsPresent  chan *geoRequests.ClaimIsPresent
	GEORequestsTSLIsPresent    chan *geoRequests.TSLIsPresent
	GEORequestsTSLGet          chan *geoRequests.TSLGet
	GEORequestsTxStates        chan *geoRequests.TxsStates

	// Internal interface
	IncomingEventTimeFrameEnded chan *ticker.EventTimeFrameStarted

	keystore   *keystore.KeyStore
	reporter   *external.Reporter
	poolTSLs   *pool.Handler
	poolClaims *pool.Handler

	chain     *Chain
	nextBlock *block.Signed

	validationFlowStates *validationFlowStates
	generationFlowStates *generationFlowStates
}

func NewProducer(
	chain *Chain, keystore *keystore.KeyStore, poolTSLs, poolClaims *pool.Handler,
	reporter *external.Reporter) (producer *Producer, err error) {

	producer = &Producer{
		// Observers interface
		OutgoingRequestsCandidateDigestBroadcast: make(chan *requests.CandidateDigestBroadcast, 128),
		IncomingRequestsCandidateDigest:          make(chan *requests.CandidateDigestBroadcast, settings.ObserversMaxCount-1),
		OutgoingResponsesCandidateDigestApprove:  make(chan *responses.CandidateDigestApprove, 128),
		IncomingResponsesCandidateDigestApprove:  make(chan *responses.CandidateDigestApprove, settings.ObserversMaxCount-1),
		OutgoingResponsesCandidateDigestReject:   make(chan *responses.CandidateDigestReject, 128),
		IncomingResponsesCandidateDigestReject:   make(chan *responses.CandidateDigestReject, settings.ObserversMaxCount-1),
		OutgoingRequestsBlockSignaturesBroadcast: make(chan *requests.BlockSignaturesBroadcast, 128),
		IncomingRequestsBlockSignatures:          make(chan *requests.BlockSignaturesBroadcast, 128),
		IncomingRequestsChainTop:                 make(chan *requests.ChainInfo, 128),
		OutgoingResponsesChainTop:                make(chan *responses.ChainInfo, 128),
		OutgoingRequestsTimeFrameCollisions:      make(chan *requests.TimeFrameCollision, 128),
		OutgoingRequestsBlockHashBroadcast:       make(chan *requests.BlockHashBroadcast, 128),
		IncomingRequestsBlockHashBroadcast:       make(chan *requests.BlockHashBroadcast, 128),
		OutgoingRequestsCollisionNotifications:   make(chan *requests.CollisionNotification, settings.ObserversMaxCount-1),
		IncomingRequestsCollisionNotifications:   make(chan *requests.CollisionNotification, 128),

		// GEO Node interface
		GEORequestsLastBlockHeight: make(chan *geoRequests.LastBlockNumber, 128),
		GEORequestsClaimIsPresent:  make(chan *geoRequests.ClaimIsPresent, 128),
		GEORequestsTSLIsPresent:    make(chan *geoRequests.TSLIsPresent, 128),
		GEORequestsTSLGet:          make(chan *geoRequests.TSLGet, 128),
		GEORequestsTxStates:        make(chan *geoRequests.TxsStates, 128),

		// Internal interface
		IncomingEventTimeFrameEnded: make(chan *ticker.EventTimeFrameStarted, 128),

		chain:      chain,
		reporter:   reporter,
		keystore:   keystore,
		poolTSLs:   poolTSLs,
		poolClaims: poolClaims,
	}
	return
}

func (p *Producer) DropEnqueuedIncomingRequestsAndResponses() {
	for len(p.IncomingRequestsCandidateDigest) > 0 {
		<-p.IncomingRequestsCandidateDigest
	}

	for len(p.IncomingResponsesCandidateDigestApprove) > 0 {
		<-p.IncomingResponsesCandidateDigestApprove
	}

	for len(p.IncomingResponsesCandidateDigestReject) > 0 {
		<-p.IncomingResponsesCandidateDigestReject
	}

	for len(p.IncomingRequestsBlockSignatures) > 0 {
		<-p.IncomingRequestsBlockSignatures
	}

	for len(p.IncomingRequestsBlockHashBroadcast) > 0 {
		<-p.IncomingRequestsBlockHashBroadcast
	}

	for len(p.IncomingRequestsCollisionNotifications) > 0 {
		<-p.IncomingRequestsCollisionNotifications
	}

	// todo: comment
	// todo: add another inc types
}

func (p *Producer) Run(chain *Chain) (err error) {
	p.chain = chain
	p.nextBlock = nil

	// Contains indexes of time frames, during which potential(!) time frame collision has been occurred.
	// This type of collisions might occur when observer, that generates next block candidate
	// (observer that is granted for block generation), reports block candidate with index,
	// that is not equal to current time frame index.
	//
	// In this case 2 options are possible:
	// 1. Ticker of remote observer is missconfigured, and generated block candidate is invalid.
	//	  Other observer is granted for block generation, and received block candidate was missgenerated,
	//	  and should be ignored.
	// 2. Ticker of current observer is missconfigured and additional sync is required.
	//
	// Both cases are equally possible.
	// It is worth to mention, that in case if this collision happens with one observer,
	// it would automatically occur on all other observers, except block generator.
	// If all observers would simply try to resync - it would lead to global network reset and,
	// potentially, significant delay in network processing.
	//
	// to prevent global network reset, one additional rule is applied:
	// resync attempt is taken only in case if time frame collision is detected 2 and more rounds in a row.
	//
	// Here map is used a set.
	timeFramesThatCouldBeInvalid := make(map[uint16]bool) // todo: drop this logic

	for {
		var e errors.E

		select {
		// todo: add case when observers configuration has changed

		case tick := <-p.IncomingEventTimeFrameEnded:
			roundError := p.processTick(tick)
			if roundError != nil {
				// todo: move this into the separate function

				if roundError.Error() == errors.BlockValidationStageFailed {
					const maxAmountOfTimeFrameCollisionsInARow = 2
					if len(timeFramesThatCouldBeInvalid) < maxAmountOfTimeFrameCollisionsInARow {
						// Potential time frame collision is detected and should be recorded.
						timeFramesThatCouldBeInvalid[tick.Index] = true
						continue

					} else {
						// Amount of potential time frame collisions is greater than allowed amount.
						// It is very probable, that current observer's ticker is not synchronised
						// with the rest of network.
						//
						// Reporting the error to the core level for additional sync to be performed.
						timeFramesThatCouldBeInvalid = make(map[uint16]bool) // Clearing the counter.
						return errors.Collision
					}

				} else {
					// Default error handler.
					// In case if unexpected error occurred - it must be reported to the core level.
					// todo: temporary message
					if len(timeFramesThatCouldBeInvalid) > 0 {
						p.log().Trace("CLEARED!")
					}
					timeFramesThatCouldBeInvalid = make(map[uint16]bool)

					return roundError.Error()
				}

			} else {
				// todo: temporary message
				if len(timeFramesThatCouldBeInvalid) > 0 {
					p.log().Trace("CLEARED!")
				}
				timeFramesThatCouldBeInvalid = make(map[uint16]bool)

				// ...
				// Another logic that is specific for round completion should go here.
			}

		//
		// Common requests processing goes here.
		// Please, search for #1 (anchor to find all other similar places).
		// Unfortunately, I know no way to inject in select several cases that are defined somewhere else.
		// todo: process case when configuration changed, to be able to interrupt flow and react ASAP

		case reqChainTop := <-p.IncomingRequestsChainTop:
			e = p.processChainTopRequest(reqChainTop)

		case reqLastBlockHeight := <-p.GEORequestsLastBlockHeight:
			e = p.processGEOLastBlockHeightRequest(reqLastBlockHeight)

		case reqClaimIsPresent := <-p.GEORequestsClaimIsPresent:
			e = p.processGEOClaimIsPresentRequest(reqClaimIsPresent)

		case reqTSLIsPresent := <-p.GEORequestsTSLIsPresent:
			e = p.processGEOTSLIsPresentRequest(reqTSLIsPresent)

		case reqTSLGet := <-p.GEORequestsTSLGet:
			e = p.processGEOTSLGetRequest(reqTSLGet)

		case reqTxStates := <-p.GEORequestsTxStates:
			e = p.processGEOTxStatesRequest(reqTxStates)

		}

		if e != nil {
			return e.Error()
		}
	}
}

func (p *Producer) processTick(tick *ticker.EventTimeFrameStarted) (e errors.E) {
	defer func() {
		// Drop previously generated block candidate and all related data in any case.
		//
		// WARN:
		// Do not rewrite error!
		// It might be necessary.
		_ = p.processFinalStage()
	}()

	p.logCurrentTimeFrameIndex(tick)

	currentObserverMustGenerateBlock := tick.Index == tick.ObserversConfiguration.CurrentObserverIndex
	if currentObserverMustGenerateBlock {
		return p.processBlockGenerationFlow(tick)

	} else {
		return p.processValidationFlow(tick)
	}
}

func (p *Producer) processFinalStage() error {
	// Signed generation period or block validation period has been finished.
	// Even if proposed block is present and has collected some signatures - it MUST be dropped.
	p.nextBlock = nil
	return nil
}
