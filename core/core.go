package core

import (
	"geo-observers-blockchain/core/chain/chain"
	"geo-observers-blockchain/core/chain/pool"
	"geo-observers-blockchain/core/crypto/keystore"
	"geo-observers-blockchain/core/geo"
	geoNet "geo-observers-blockchain/core/network/communicator/geo"
	geoRequestsCommon "geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	geoRequests "geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	observersNet "geo-observers-blockchain/core/network/communicator/observers"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"reflect"
	"time"
)

type Core struct {
	settings              *settings.Settings
	keystore              *keystore.KeyStore
	timer                 *ticker.Ticker
	observersConfReporter *external.Reporter
	receiverGEONodes      *geoNet.Receiver
	receiverObservers     *observersNet.Receiver
	senderObservers       *observersNet.Sender
	poolClaims            *pool.Handler
	poolTSLs              *pool.Handler
	blocksProducer        *chain.Producer
	composer              *chain.Composer
}

func New(conf *settings.Settings) (core *Core, err error) {
	k, err := keystore.New()
	if err != nil {
		return
	}

	reporter := external.NewReporter(conf, k)
	poolTSLs := pool.NewHandler(reporter)
	poolClaims := pool.NewHandler(reporter)
	composer := chain.NewComposer(reporter, conf)
	producer, err := chain.NewProducer(conf, reporter, k, poolTSLs, poolClaims, composer)

	core = &Core{
		settings:              conf,
		keystore:              k,
		timer:                 ticker.New(conf, reporter),
		observersConfReporter: reporter,
		senderObservers:       observersNet.NewSender(conf, reporter),
		receiverObservers:     observersNet.NewReceiver(),
		receiverGEONodes:      geoNet.NewReceiver(),
		poolClaims:            poolClaims,
		poolTSLs:              poolTSLs,
		blocksProducer:        producer,
		composer:              composer,
	}

	return
}

func (c *Core) Run() {
	globalErrorsFlow := make(chan error, 128)

	c.initNetwork(globalErrorsFlow)
	c.initProcessing(globalErrorsFlow)

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
}

func (c *Core) initNetwork(errors chan error) {
	go c.receiverGEONodes.Run(
		c.settings.Nodes.Network.Host,
		c.settings.Nodes.Network.Port,
		errors)
	c.exitIfError(errors)

	go c.receiverObservers.Run(
		c.settings.Observers.Network.Host,
		c.settings.Observers.Network.Port,
		errors)
	c.exitIfError(errors)

	go c.senderObservers.Run(
		c.settings.Observers.Network.Host,
		c.settings.Observers.Network.Port,
		errors)

	c.exitIfError(errors)

	// ...
	// Other initialisation goes here
	// ...

	// Provide enough time for the network handler for bootstrap.
	time.Sleep(time.Millisecond * 100)
}

func (c *Core) initProcessing(globalErrorsFlow chan error) {
	go c.timer.Run(globalErrorsFlow)
	go c.blocksProducer.Run(globalErrorsFlow)

	go c.dispatchDataFlows(globalErrorsFlow)
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

	for {
		select {

		// Events
		case eventConnectionClosed := <-c.receiverObservers.OutgoingEventsConnectionClosed:
			select {
			case c.senderObservers.IncomingEvents <- eventConnectionClosed:
			default:
				processTransferringFail(eventConnectionClosed, c.senderObservers)
			}

		case tick := <-c.timer.OutgoingEventsTimeFrameEnd:
			select {
			case c.blocksProducer.IncomingEventTimeFrameEnded <- tick:
			default:
				processTransferringFail(tick, c.blocksProducer)
			}

		// Block producer
		case outgoingRequestCandidateDigestBroadcast := <-c.blocksProducer.OutgoingRequestsCandidateDigestBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestCandidateDigestBroadcast:
			default:
				processTransferringFail(outgoingRequestCandidateDigestBroadcast, c.senderObservers)
			}

		case outgoingResponseCandidateDigestApprove := <-c.blocksProducer.OutgoingResponsesCandidateDigestApprove:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseCandidateDigestApprove:
			default:
				processTransferringFail(outgoingResponseCandidateDigestApprove, c.senderObservers)
			}

		case outgoingRequestBlockSignaturesBroadcast := <-c.blocksProducer.OutgoingRequestsBlockSignaturesBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestBlockSignaturesBroadcast:
			default:
				processTransferringFail(outgoingRequestBlockSignaturesBroadcast, c.senderObservers)
			}

		case outgoingRequestChainTop := <-c.composer.OutgoingRequestsChainTop:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestChainTop:
			default:
				processTransferringFail(outgoingRequestChainTop, c.senderObservers)
			}

		case outgoingResponseChainTop := <-c.blocksProducer.OutgoingResponsesChainTop:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseChainTop:
			default:
				processTransferringFail(outgoingResponseChainTop, c.senderObservers)
			}

		// Ticker
		case outgoingRequestTimeFrames := <-c.timer.OutgoingRequestsTimeFrames:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestTimeFrames:
			default:
				processTransferringFail(outgoingRequestTimeFrames, c.senderObservers)
			}

		case outgoingResponseTimeFrame := <-c.timer.OutgoingResponsesTimeFrame:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseTimeFrame:
			default:
				processTransferringFail(outgoingResponseTimeFrame, c.senderObservers)
			}

		// Claims pool
		case outgoingRequestClaimBroadcast := <-c.poolClaims.OutgoingRequestsInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestClaimBroadcast:
			default:
				processTransferringFail(outgoingRequestClaimBroadcast, c.senderObservers)
			}

		case outgoingResponseClaimApprove := <-c.poolClaims.OutgoingResponsesInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingResponses <- &responses.ClaimApprove{
				PoolInstanceBroadcastApprove: outgoingResponseClaimApprove}:
			default:
				processTransferringFail(outgoingResponseClaimApprove, c.senderObservers)
			}

		case outgoingRequestTSLBroadcast := <-c.poolTSLs.OutgoingRequestsInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestTSLBroadcast:
			default:
				processTransferringFail(outgoingRequestTSLBroadcast, c.senderObservers)
			}

		case outgoingResponseTSLApprove := <-c.poolTSLs.OutgoingResponsesInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingResponses <- &responses.TSLApprove{
				PoolInstanceBroadcastApprove: outgoingResponseTSLApprove}:
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
		case geoNodeRequest := <-c.receiverGEONodes.Requests:
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
		case c.timer.IncomingRequestsTimeFrames <- r.(*requests.SynchronisationTimeFrames):
		default:
			processTransferringFail(r, c.timer)
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

	case *requests.ChainTop:
		select {
		case c.blocksProducer.IncomingRequestsChainTop <- r.(*requests.ChainTop):
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
		case c.timer.IncomingResponsesTimeFrame <- r.(*responses.TimeFrame):

		default:
			processTransferringFail(r, c.timer)
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

	case *responses.ChainTop:
		select {
		case c.composer.IncomingResponsesChainTop <- r.(*responses.ChainTop):
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
