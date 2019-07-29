package core

import (
	"geo-observers-blockchain/core/chain/chain"
	"geo-observers-blockchain/core/chain/pool"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/composer"
	"geo-observers-blockchain/core/crypto/keystore"
	"geo-observers-blockchain/core/geo"
	geoNet "geo-observers-blockchain/core/network/communicator/geo"
	geoRequestsCommon "geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	geoRequests "geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	observersNet "geo-observers-blockchain/core/network/communicator/observers"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"reflect"
	"time"
)

type Core struct {
	keystore              *keystore.KeyStore
	ticker                *ticker.Ticker
	observersConfReporter *external.Reporter
	communicatorGEONodes  *geoNet.Communicator
	receiverObservers     *observersNet.Receiver
	senderObservers       *observersNet.Sender
	poolClaims            *pool.Handler
	poolTSLs              *pool.Handler
	blockchain            *chain.Chain
	blocksProducer        *chain.Producer
	composer              *composer.Composer
}

func New() (core *Core, err error) {
	blockchain, err := chain.NewChain(chain.DataFilePath)
	if err != nil {
		return
	}

	keys, err := keystore.New()
	if err != nil {
		return
	}

	reporter := external.NewReporter(keys)
	poolTSLs := pool.NewHandler(reporter)
	poolClaims := pool.NewHandler(reporter)

	producer, err := chain.NewProducer(blockchain, keys, poolTSLs, poolClaims, reporter)
	chainComposer := composer.NewComposer()

	core = &Core{
		blockchain:     blockchain,
		composer:       chainComposer,
		blocksProducer: producer,
		keystore:       keys,
		ticker:         ticker.New(reporter),

		senderObservers:       observersNet.NewSender(reporter),
		receiverObservers:     observersNet.NewReceiver(),
		observersConfReporter: reporter,

		communicatorGEONodes: geoNet.New(),

		poolClaims: poolClaims,
		poolTSLs:   poolTSLs,
	}

	return
}

func (c *Core) Run() {
	globalErrorsFlow := make(chan error, 128)

	go func() {
		for {
			select {
			case err := <-globalErrorsFlow:
				{
					if err != nil {
						// todo: enhance error processing
						//c.log().Warn(err)
					}
				}
			}
		}
	}()

	c.initNetwork(globalErrorsFlow)
	c.initSubcomponents(globalErrorsFlow)

	// Blocking call.
	c.processMainFlow(globalErrorsFlow)
}

func (c *Core) initNetwork(errors chan error) {
	go c.communicatorGEONodes.Run(errors)

	c.receiverObservers.IgnoreRoundRelatedTraffic()
	go c.receiverObservers.Run(
		settings.Conf.Observers.Network.Host,
		settings.Conf.Observers.Network.Port,
		errors)

	go c.senderObservers.Run(
		settings.Conf.Observers.Network.Host,
		settings.Conf.Observers.Network.Port,
		errors)

	// ...
	// Other initialisation goes here
	// ...

	// Provide enough time for the network handler for bootstrap.
	time.Sleep(time.Millisecond * 100)
}

func (c *Core) initSubcomponents(globalErrorsFlow chan error) {
	go c.ticker.Run(globalErrorsFlow)
	go c.poolClaims.Run(globalErrorsFlow)
	go c.poolTSLs.Run(globalErrorsFlow)
	go c.dispatchDataFlows(globalErrorsFlow)
	go c.composer.Run(globalErrorsFlow)
}

func (c *Core) processMainFlow(globalErrorsFlow chan error) {

	currentBlock, e := c.blockchain.LastBlock()
	if e != nil {
		globalErrorsFlow <- e.Error()
		return
	}

	c.log().WithFields(log.Fields{
		"CurrentChainHeight": currentBlock.Body.Index,
		"LastBlockHash":      currentBlock.Body.Hash.Hex(),
	}).Info("Processing started")

	// todo: consider moving this logic into separate function
	syncChain := func(tick *ticker.EventTimeFrameStarted,
		ticksChannel chan *ticker.EventTimeFrameStarted) (
		lastTimeFrameEventsChan chan *ticker.EventTimeFrameStarted, err error) {

		lastTimeFrameEventsChan = c.composer.SyncChain(c.blockchain, tick, ticksChannel)
		syncResult := <-c.composer.OutgoingEvents.SyncFinished
		if syncResult.Error != nil {
			return
		}

		c.blockchain, err = chain.NewChain(chain.DataFilePath)
		if err != nil {
			return
		}

		return
	}

	initialTimeFrame, e := c.ticker.InitialTimeFrame()
	if e != nil {
		c.log().Error(e.StackTrace())
		panic(e.Error())
	}

	lastTimeFrameEventsChan, err := syncChain(initialTimeFrame, c.ticker.OutgoingEventsTimeFrameEnd)
	if err != nil {
		globalErrorsFlow <- err // todo consolidate error
		panic(err)              // replace with fatal error throwing.
	}

	// Begin current round right with time frame, that was used during synchronisation,
	// to minimize probability of skipping last generated block.
	// Transferring current event back to the ticker's channel.
	c.ticker.OutgoingEventsTimeFrameEnd <- <-lastTimeFrameEventsChan

	c.receiverObservers.AcceptAllTraffic()
	for {
		err := c.blocksProducer.Run(c.blockchain)
		if err != nil {
			if err == errors.Collision || err == errors.SyncNeeded {
				c.log().Trace("Collision detected or sync. request occurred. " +
					"Ticker and chain synchronisation started")

				c.log().Trace("Collision detected. Chain sync started. Details: ", err.Error())

				c.receiverObservers.IgnoreRoundRelatedTraffic()

				// Additional ticker synchronisation might be needed.
				c.ticker.Restart()

				// Wait for ticker sync to be done.
				timeFrameEvent := <-c.ticker.OutgoingEventsTimeFrameEnd

				lastTimeFrameEventsChan, err = syncChain(timeFrameEvent, c.ticker.OutgoingEventsTimeFrameEnd)
				if err != nil {
					globalErrorsFlow <- err
					panic(err) // replace with fatal error throwing.
				}

				//// Begin current round right with time frame, that was used during synchronisation,
				//// to minimize probability of skipping last generated block.
				//c.ticker.OutgoingEventsTimeFrameEnd <- <- lastTimeFrameEventsChan
				c.receiverObservers.AcceptAllTraffic()

				c.blocksProducer.DropEnqueuedIncomingRequestsAndResponses()

			} else {
				panic(err)
			}
		}
	}
}

func (c *Core) dispatchDataFlows(globalErrorsFlow chan error) {

	processTransferringFail := func(instance, processor interface{}) {
		err := utils.Error("core",
			reflect.TypeOf(instance).String()+" failed to process by "+reflect.TypeOf(processor).String())
		globalErrorsFlow <- err
	}

	processError := func(err error) {
		globalErrorsFlow <- err
	}

	// Prints debug trace about requests transferred for further processing.
	printDebugOutgoingInfoStruct := func(request interface{}) {
		if !settings.OutputCoreDispatcherDebug {
			return
		}

		c.log().WithFields(
			log.Fields{
				"type":      reflect.TypeOf(request).String(),
				"component": "core/dispatchDataFlows",
			}).Debug("Outgoing request sent")
	}

	for {
		select {

		// composer
		case outgoingRequestChainTop := <-c.composer.OutgoingRequests.ChainInfo:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestChainTop:
				printDebugOutgoingInfoStruct(outgoingRequestChainTop)

			default:
				processTransferringFail(outgoingRequestChainTop, c.senderObservers)
			}

		// outgoingEvents
		case eventConnectionClosed := <-c.receiverObservers.OutgoingEventsConnectionClosed:
			select {
			case c.senderObservers.IncomingEvents <- eventConnectionClosed:
				printDebugOutgoingInfoStruct(eventConnectionClosed)

			default:
				processTransferringFail(eventConnectionClosed, c.senderObservers)
			}

		case tick := <-c.ticker.OutgoingEventsTimeFrameEnd:
			select {
			case c.blocksProducer.IncomingEventTimeFrameEnded <- tick:
				printDebugOutgoingInfoStruct(tick)

			default:
				processTransferringFail(tick, c.blocksProducer)
			}

		// Block producer
		case outgoingRequestCandidateDigestBroadcast := <-c.blocksProducer.OutgoingRequestsCandidateDigestBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestCandidateDigestBroadcast:
				printDebugOutgoingInfoStruct(outgoingRequestCandidateDigestBroadcast)

			default:
				processTransferringFail(outgoingRequestCandidateDigestBroadcast, c.senderObservers)
			}

		case outgoingResponseCandidateDigestApprove := <-c.blocksProducer.OutgoingResponsesCandidateDigestApprove:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseCandidateDigestApprove:
				printDebugOutgoingInfoStruct(outgoingResponseCandidateDigestApprove)

			default:
				processTransferringFail(outgoingResponseCandidateDigestApprove, c.senderObservers)
			}

		case outgoingRequestBlockSignaturesBroadcast := <-c.blocksProducer.OutgoingRequestsBlockSignaturesBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestBlockSignaturesBroadcast:
				printDebugOutgoingInfoStruct(outgoingRequestBlockSignaturesBroadcast)

			default:
				processTransferringFail(outgoingRequestBlockSignaturesBroadcast, c.senderObservers)
			}

		case outgoingResponseChainTop := <-c.blocksProducer.OutgoingResponsesChainTop:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseChainTop:
				printDebugOutgoingInfoStruct(outgoingResponseChainTop)

			default:
				processTransferringFail(outgoingResponseChainTop, c.senderObservers)
			}

		case outgoingRequestTimeFrameCollision := <-c.blocksProducer.OutgoingRequestsTimeFrameCollisions:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestTimeFrameCollision:
				printDebugOutgoingInfoStruct(outgoingRequestTimeFrameCollision)

			default:
				processTransferringFail(outgoingRequestTimeFrameCollision, c.senderObservers)
			}

		case outgoingRequestBlockHashBroadcast := <-c.blocksProducer.OutgoingRequestsBlockHashBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestBlockHashBroadcast:
				printDebugOutgoingInfoStruct(outgoingRequestBlockHashBroadcast)

			default:
				processTransferringFail(outgoingRequestBlockHashBroadcast, c.senderObservers)
			}

		case outgoingResponseDigestReject := <-c.blocksProducer.OutgoingResponsesCandidateDigestReject:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseDigestReject:
				printDebugOutgoingInfoStruct(outgoingResponseDigestReject)

			default:
				processTransferringFail(outgoingResponseDigestReject, c.senderObservers)
			}

		// Ticker
		case outgoingRequestTimeFrames := <-c.ticker.OutgoingRequestsTimeFrames:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestTimeFrames:
				printDebugOutgoingInfoStruct(outgoingRequestTimeFrames)

			default:
				processTransferringFail(outgoingRequestTimeFrames, c.senderObservers)
			}

		case outgoingResponseTimeFrame := <-c.ticker.OutgoingResponsesTimeFrame:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseTimeFrame:
				printDebugOutgoingInfoStruct(outgoingResponseTimeFrame)

			default:
				processTransferringFail(outgoingResponseTimeFrame, c.senderObservers)
			}

		// Claims pool
		case outgoingRequestClaimBroadcast := <-c.poolClaims.OutgoingRequestsInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestClaimBroadcast:
				printDebugOutgoingInfoStruct(outgoingRequestClaimBroadcast)

			default:
				processTransferringFail(outgoingRequestClaimBroadcast, c.senderObservers)
			}

		case outgoingResponseClaimApprove := <-c.poolClaims.OutgoingResponsesInstanceBroadcast:
			resp := responses.ClaimApprove{PoolInstanceBroadcastApprove: outgoingResponseClaimApprove}

			select {
			case c.senderObservers.OutgoingResponses <- &resp:
				printDebugOutgoingInfoStruct(resp)

			default:
				processTransferringFail(outgoingResponseClaimApprove, c.senderObservers)
			}

		case outgoingRequestTSLBroadcast := <-c.poolTSLs.OutgoingRequestsInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestTSLBroadcast:
				printDebugOutgoingInfoStruct(outgoingRequestTSLBroadcast)

			default:
				processTransferringFail(outgoingRequestTSLBroadcast, c.senderObservers)
			}

		case outgoingResponseTSLApprove := <-c.poolTSLs.OutgoingResponsesInstanceBroadcast:
			resp := &responses.TSLApprove{PoolInstanceBroadcastApprove: outgoingResponseTSLApprove}

			select {
			case c.senderObservers.OutgoingResponses <- resp:
				printDebugOutgoingInfoStruct(resp)

			default:
				processTransferringFail(outgoingResponseTSLApprove, c.senderObservers)
			}

		// Observers incoming requests and responses processing
		case incomingRequest := <-c.receiverObservers.Requests:
			err := c.processIncomingRequest(incomingRequest)
			if err != nil {
				processError(err)
			}

		case incomingResponse := <-c.receiverObservers.Responses:
			err := c.processIncomingResponse(incomingResponse)
			if err != nil {
				processError(err)
			}

		// GEO Node requests processing
		case geoNodeRequest := <-c.communicatorGEONodes.Requests:
			err := c.processIncomingGEONodeRequest(geoNodeRequest)
			if err != nil {
				processError(err)
			}
		}
	}
}

func (c *Core) processIncomingRequest(r requests.Request) (err error) {
	processTransferringFail := func(instance, processor interface{}) {
		err = utils.Error("core",
			reflect.TypeOf(instance).String()+
				" wasn't sent to "+
				reflect.TypeOf(processor).String()+
				" due to channel transferring delay/error")
	}

	switch r.(type) {
	case *requests.SynchronisationTimeFrames:
		select {
		case c.ticker.IncomingRequestsTimeFrames <- r.(*requests.SynchronisationTimeFrames):
		default:
			processTransferringFail(r, c.ticker)
		}

	case *requests.PoolInstanceBroadcast:
		switch r.(*requests.PoolInstanceBroadcast).Instance.(type) {
		case *geo.TSL:
			select {
			case c.poolTSLs.IncomingRequestsInstanceBroadcast <- r.(*requests.PoolInstanceBroadcast):
			default:
				processTransferringFail(r, c.poolTSLs)
			}

		case *geo.Claim:
			select {
			case c.poolClaims.IncomingRequestsInstanceBroadcast <- r.(*requests.PoolInstanceBroadcast):
			default:
				processTransferringFail(r, c.poolClaims)
			}

		default:
			err = utils.Error("core", "unexpected pool instance type occurred")
		}

	case *requests.CandidateDigestBroadcast:
		select {
		case c.blocksProducer.IncomingRequestsCandidateDigest <- r.(*requests.CandidateDigestBroadcast):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *requests.BlockSignaturesBroadcast:
		select {
		case c.blocksProducer.IncomingRequestsBlockSignatures <- r.(*requests.BlockSignaturesBroadcast):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *requests.ChainInfo:
		select {
		case c.blocksProducer.IncomingRequestsChainTop <- r.(*requests.ChainInfo):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	//case *requests.TimeFrameCollision:
	//	select {
	//	case c.ticker.IncomingRequestsTimeFrameCollision <- r.(*requests.TimeFrameCollision):
	//	default:
	//		processTransferringFail(r, c.ticker)
	//	}

	case *requests.BlockHashBroadcast:
		select {
		case c.blocksProducer.IncomingRequestsBlockHashBroadcast <- r.(*requests.BlockHashBroadcast):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	default:
		err = utils.Error("core", "unexpected request type occurred")
	}

	return
}

func (c *Core) processIncomingGEONodeRequest(r geoRequestsCommon.Request) (err error) {
	processTransferringFail := func(instance, processor interface{}) {
		err = utils.Error("core",
			reflect.TypeOf(instance).String()+
				" wasn't sent to "+
				reflect.TypeOf(processor).String()+
				" due to channel transferring delay/error")
	}

	switch r.(type) {
	case *geoRequests.LastBlockNumber:
		select {
		case c.blocksProducer.GEORequestsLastBlockHeight <- r.(*geoRequests.LastBlockNumber):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *geoRequests.ClaimAppend:
		select {
		case c.poolClaims.IncomingInstances <- r.(*geoRequests.ClaimAppend).Claim:
		default:
			processTransferringFail(r, c.poolClaims)
		}

	case *geoRequests.ClaimIsPresent:
		select {
		case c.blocksProducer.GEORequestsClaimIsPresent <- r.(*geoRequests.ClaimIsPresent):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *geoRequests.TSLAppend:
		select {
		case c.poolTSLs.IncomingInstances <- r.(*geoRequests.TSLAppend).TSL:
		default:
			processTransferringFail(r, c.poolTSLs)
		}

	case *geoRequests.TSLIsPresent:
		select {
		case c.blocksProducer.GEORequestsTSLIsPresent <- r.(*geoRequests.TSLIsPresent):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *geoRequests.TSLGet:
		select {
		case c.blocksProducer.GEORequestsTSLGet <- r.(*geoRequests.TSLGet):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *geoRequests.TxsStates:
		select {
		case c.blocksProducer.GEORequestsTxStates <- r.(*geoRequests.TxsStates):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	default:
		err = utils.Error("core", "unexpected request type occurred")
	}

	return
}

func (c *Core) processIncomingResponse(r responses.Response) (err error) {
	processTransferringFail := func(instance, processor interface{}) {
		err = utils.Error("core",
			reflect.TypeOf(instance).String()+
				" wasn't sent to "+
				reflect.TypeOf(processor).String()+
				" due to channel transferring delay/error")
	}

	switch r.(type) {
	case *responses.TimeFrame:
		select {
		case c.ticker.IncomingResponsesTimeFrame <- r.(*responses.TimeFrame):

		default:
			processTransferringFail(r, c.ticker)
		}

	case *responses.ClaimApprove:
		select {
		case c.poolClaims.IncomingResponsesInstanceBroadcast <- responses.NewPoolInstanceBroadcastApprove(
			r.Request(), r.ObserverIndex(), r.(*responses.ClaimApprove).Hash):

		default:
			processTransferringFail(r, c.poolClaims)
		}

	case *responses.TSLApprove:
		select {
		case c.poolTSLs.IncomingResponsesInstanceBroadcast <- responses.NewPoolInstanceBroadcastApprove(
			r.Request(), r.ObserverIndex(), r.(*responses.TSLApprove).Hash):

		default:
			processTransferringFail(r, c.poolTSLs)
		}

	case *responses.CandidateDigestApprove:
		select {
		case c.blocksProducer.IncomingResponsesCandidateDigestApprove <- r.(*responses.CandidateDigestApprove):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *responses.CandidateDigestReject:
		select {
		case c.blocksProducer.IncomingResponsesCandidateDigestReject <- r.(*responses.CandidateDigestReject):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	case *responses.ChainInfo:
		select {
		case c.composer.IncomingResponses.ChainInfo <- r.(*responses.ChainInfo):
		default:
			processTransferringFail(r, c.blocksProducer)
		}

	default:
		err = utils.Error("core", "unexpected response type occurred")
	}

	return
}

func (c *Core) exitIfError(errors <-chan error) {
	select {
	case err := <-errors:
		if err != nil {
			log.Fatal(err, "Exit")
		}
	}
}

func (c *Core) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Core"})
}
