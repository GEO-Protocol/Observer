package core

import (
	"geo-observers-blockchain/core/chain"
	"geo-observers-blockchain/core/chain/pool"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/crypto/keystore"
	"geo-observers-blockchain/core/geo"
	geoNet "geo-observers-blockchain/core/network/communicator/geo"
	observersNet "geo-observers-blockchain/core/network/communicator/observers"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/timer"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
)

type Core struct {
	settings              *settings.Settings
	keystore              *keystore.KeyStore
	timer                 *timer.Timer
	observersConfReporter *external.Reporter
	receiverGEONodes      *geoNet.Receiver
	receiverObservers     *observersNet.Receiver
	senderObservers       *observersNet.Sender
	poolClaims            *pool.Handler
	poolTSLs              *pool.Handler
	blocksProducer        *chain.BlocksProducer
}

func New(conf *settings.Settings) (core *Core, err error) {
	k, err := keystore.New()
	if err != nil {
		return
	}

	reporter := external.NewReporter(conf)
	producer, err := chain.NewBlocksProducer(conf, reporter, k)

	core = &Core{
		settings:              conf,
		keystore:              k,
		timer:                 timer.New(conf, reporter),
		observersConfReporter: reporter,
		senderObservers:       observersNet.NewSender(conf, reporter),
		receiverObservers:     observersNet.NewReceiver(),
		receiverGEONodes:      geoNet.NewReceiver(),
		poolClaims:            pool.NewHandler(reporter),
		poolTSLs:              pool.NewHandler(reporter),
		blocksProducer:        producer,
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
					c.log().Error(err)
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
	c.log().Info(
		"GEO Nodes communicator started"+
			"Connections are accepted on: ",
		c.settings.Nodes.Network.Host, ":", c.settings.Nodes.Network.Port)

	go c.receiverObservers.Run(
		c.settings.Observers.Network.Host,
		c.settings.Observers.Network.Port,
		errors)

	c.exitIfError(errors)
	c.log().Info(
		"Observers incoming connections receiver started. "+
			"Connections are accepted on: ",
		c.settings.Observers.Network.Host, ":", c.settings.Observers.Network.Port)

	go c.senderObservers.Run(
		c.settings.Observers.Network.Host,
		c.settings.Observers.Network.Port,
		errors)

	c.exitIfError(errors)
	log.Println("Observers outgoing info sender started")

	// ...
	// Other initialisation goes here
	// ...
}

func (c *Core) initProcessing(globalErrorsFlow chan error) {
	go c.poolClaims.Run(globalErrorsFlow)
	go c.poolTSLs.Run(globalErrorsFlow)

	go c.dispatchDataFlows(globalErrorsFlow)
	//go c.timer.Run(globalErrorsFlow)
	//
	//go c.blocksProducer.Run(globalErrorsFlow)
	//c.exitIfError(globalErrorsFlow)

}

func (c *Core) dispatchDataFlows(globalErrorsFlow chan error) {

	processFail := func(message string) {
		err := errors.ExpandWithStackTrace(utils.Error("core", message))
		globalErrorsFlow <- err
	}

	processError := func(err error) {
		err = errors.ExpandWithStackTrace(err)
		globalErrorsFlow <- err
	}

	for {
		select {

		// Events
		case eventConnectionClosed := <-c.receiverObservers.OutgoingEventsConnectionClosed:
			select {
			case c.senderObservers.IncomingEvents <- eventConnectionClosed:

			default:
				processFail("can't send event `connection closed` to the observers sender")
			}

		case tick := <-c.timer.OutgoingEventsTimeFrameEnd:
			select {
			case c.blocksProducer.IncomingEventTimeFrameEnded <- tick:

			default:
				processFail("can't send event `time frame ended` to the blocks producer")
			}

		// Block producer
		case outgoingBlockProposed := <-c.blocksProducer.OutgoingBlocksProposals:
			select {
			case c.senderObservers.OutgoingProposedBlocks <- outgoingBlockProposed:

			default:
				processFail("can't send proposed block to the observers sender")
			}

		case outgoingBlockSignature := <-c.blocksProducer.OutgoingBlocksSignatures:
			select {
			case c.senderObservers.OutgoingBlocksSignatures <- outgoingBlockSignature:

			default:
				processFail("can't send block signature to the observers sender")
			}

		// Observers receiver
		case incomingProposedBlock := <-c.receiverObservers.BlocksProposed:
			select {
			case c.blocksProducer.IncomingBlocksProposals <- incomingProposedBlock:

			default:
				processFail("can't send proposed block to the blocks producer")
			}

		case incomingBlockSignature := <-c.receiverObservers.BlockSignatures:
			select {
			case c.blocksProducer.IncomingBlocksSignatures <- incomingBlockSignature:

			default:
				processFail("can't send proposed block signature to the blocks producer")
			}

		// GEO Nodes Receiver
		case incomingTSL := <-c.receiverGEONodes.IncomingTSLs:
			select {
			case c.poolTSLs.IncomingInstances <- incomingTSL:

			default:
				processFail("can't send received tsl to the tsls pool")
			}

		case incomingClaim := <-c.receiverGEONodes.IncomingClaims:
			select {
			case c.poolClaims.IncomingInstances <- incomingClaim:

			default:
				processFail("can't send received claim to the claims pool")
			}

		// Timer
		case outgoingRequestTimeFrames := <-c.timer.OutgoingRequestsTimeFrames:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestTimeFrames:

			default:
				processFail("can't send request `time frames` to the observers sender")
			}

		case outgoingResponseTimeFrame := <-c.timer.OutgoingResponsesTimeFrame:
			select {
			case c.senderObservers.OutgoingResponses <- outgoingResponseTimeFrame:

			default:
				processFail("can't send response `time frames` to the observers sender")
			}

		// Claims pool
		case outgoingRequestClaimBroadcast := <-c.poolClaims.OutgoingRequestsInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestClaimBroadcast:

			default:
				processFail("can't send request `claim broadcast` to the observers sender")
			}

		case outgoingResponseClaimApprove := <-c.poolClaims.OutgoingResponsesInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingResponses <- &responses.ResponseClaimApprove{
				ResponsePoolInstanceBroadcastApprove: outgoingResponseClaimApprove}:

			default:
				processFail("can't send response `claim broadcast` to the observers sender")
			}

		case outgoingRequestTSLBroadcast := <-c.poolTSLs.OutgoingRequestsInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingRequests <- outgoingRequestTSLBroadcast:

			default:
				processFail("can't send request `TSL broadcast` to the observers sender")
			}

		case outgoingResponseTSLApprove := <-c.poolTSLs.OutgoingResponsesInstanceBroadcast:
			select {
			case c.senderObservers.OutgoingResponses <- &responses.ResponseTSLApprove{
				ResponsePoolInstanceBroadcastApprove: outgoingResponseTSLApprove}:

			default:
				processFail("can't send response `TSL broadcast` to the observers sender")
			}

		// Incoming requests and responses processing
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
		}
	}
}

func (c *Core) processIncomingRequest(r requests.Request) (err error) {
	switch r.(type) {
	case *requests.RequestSynchronisationTimeFrames:
		select {
		case c.timer.IncomingRequestsTimeFrames <- r.(*requests.RequestSynchronisationTimeFrames):

		default:
			err = utils.Error("core", "can't send request `time frames` to the timer")
		}

	case *requests.RequestPoolInstanceBroadcast:
		switch r.(*requests.RequestPoolInstanceBroadcast).Instance.(type) {
		case *geo.TransactionSignaturesList:
			select {
			case c.poolTSLs.IncomingRequestsInstanceBroadcast <- r.(*requests.RequestPoolInstanceBroadcast):

			default:
				err = utils.Error("core", "can't send request `tsl broadcast` to the tsl pool")
			}

		case *geo.Claim:
			select {
			case c.poolClaims.IncomingRequestsInstanceBroadcast <- r.(*requests.RequestPoolInstanceBroadcast):

			default:
				err = utils.Error("core", "can't send request `claim broadcast` to the claims pool")
			}

		default:
			err = utils.Error("core", "unexpected pool instance type occurred")
		}

	default:
		err = utils.Error("core", "unexpected request type occurred")
	}

	return
}

func (c *Core) processIncomingResponse(r responses.Response) (err error) {
	switch r.(type) {
	case *responses.ResponseTimeFrame:
		select {
		case c.timer.IncomingResponsesTimeFrame <- r.(*responses.ResponseTimeFrame):

		default:
			err = utils.Error("core", "can't send response `time frames` to the timer")
		}

	case *responses.ResponseClaimApprove:
		select {
		case c.poolClaims.IncomingResponsesInstanceBroadcast <- responses.NewResponsePoolInstanceBroadcastApprove(
			r.Request(), r.ObserverNumber(), r.(*responses.ResponseClaimApprove).Hash):

		default:
			err = utils.Error("core", "can't send request `claim broadcast` to the claims pool")
		}

	case *responses.ResponseTSLApprove:
		select {
		case c.poolTSLs.IncomingResponsesInstanceBroadcast <- responses.NewResponsePoolInstanceBroadcastApprove(
			r.Request(), r.ObserverNumber(), r.(*responses.ResponseTSLApprove).Hash):

		default:
			err = utils.Error("core", "can't send response `tsl broadcast` to the tsl pool")
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
	return log.WithFields(log.Fields{"subsystem": "Core"})
}
