package core

import (
	"geo-observers-blockchain/core/chain"
	"geo-observers-blockchain/core/crypto/keystore"
	geoNet "geo-observers-blockchain/core/network/communicator/geo"
	observersNet "geo-observers-blockchain/core/network/communicator/observers"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/timer"
	log "github.com/sirupsen/logrus"
)

type Core struct {
	timer *timer.Timer

	// todo: make members private
	Settings *settings.Settings
	Keystore *keystore.KeyStore

	ObserversConfReporter *external.Reporter
	GEONodesCommunicator  *geoNet.NodesCommunicator
	ObserversReceiver     *observersNet.Receiver
	ObserversSender       *observersNet.Sender

	BlocksProducer *chain.BlocksProducer
}

func New(conf *settings.Settings) (core *Core, err error) {
	k, err := keystore.New()
	if err != nil {
		return
	}

	reporter := external.NewReporter(conf)
	producer, err := chain.NewBlocksProducer(conf, reporter, k)

	core = &Core{
		timer: timer.New(reporter),

		Settings: conf,
		Keystore: k,

		ObserversConfReporter: reporter,
		ObserversSender:       observersNet.NewSender(conf, reporter),
		ObserversReceiver:     observersNet.NewReceiver(),
		GEONodesCommunicator:  geoNet.NewNodesCommunicator(),
		BlocksProducer:        producer,
	}
	return
}

func (c *Core) Run() {
	errors := make(chan error, 128)

	c.initNetwork(errors)
	c.initProcessing(errors)

	for {
		select {
		case err := <-errors:
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
	go c.GEONodesCommunicator.Run(
		c.Settings.Nodes.Network.Host,
		c.Settings.Nodes.Network.Port,
		errors)

	c.exitIfError(errors)
	c.log().Info("GEO Nodes communicator started")

	go c.ObserversReceiver.Run(
		c.Settings.Observers.Network.Host,
		c.Settings.Observers.Network.Port,
		errors)

	c.exitIfError(errors)
	c.log().Info(
		"Observers incoming connections receiver started. "+
			"Connections are accepted on: ",
		c.Settings.Observers.Network.Host, ":", c.Settings.Observers.Network.Port)

	go c.ObserversSender.Run(
		c.Settings.Observers.Network.Host,
		c.Settings.Observers.Network.Port,
		errors)

	c.exitIfError(errors)
	log.Println("Observers outgoing info sender started")

	// ...
	// Other initialisation goes here
	// ...
}

func (c *Core) initProcessing(errors chan error) {
	go c.dispatchDataFlows(errors)
	go c.timer.Run(errors)

	//go c.BlocksProducer.Run(errors)
	//c.exitIfError(errors)

}

func (c *Core) dispatchDataFlows(errors chan error) {
	for {
		select {

		//
		// Internal
		//
		case frame := <-c.timer.OutgoingEventsTimeFrameEnd:
			{
				// todo: include block producer

				c.log().WithFields(log.Fields{"Frame": frame.Index}).Info("Block")
			}

		//
		// Events
		//

		case eventConnectionClosed := <-c.ObserversReceiver.OutgoingEventsConnectionClosed:
			{
				select {
				case c.ObserversSender.IncomingEvents <- eventConnectionClosed:
					{

					}
				default:
					c.log().Error("can't send event `connection closed` to the observers sender")
				}
			}

		//
		// Network
		//

		// Outgoing data flow
		case outgoingBlockProposed := <-c.BlocksProducer.OutgoingBlocksProposals:
			{
				select {
				case c.ObserversSender.OutgoingProposedBlocks <- outgoingBlockProposed:
					{

					}
				default:
					c.log().Error("can't send proposed block to the observers sender")
				}
			}

		case outgoingBlockSignature := <-c.BlocksProducer.OutgoingBlocksSignatures:
			{
				select {
				case c.ObserversSender.OutgoingBlocksSignatures <- outgoingBlockSignature:
					{

					}
				default:
					c.log().Error("can't send block signature to the observers sender")
				}
			}

		case outgoingRequestTimeFrames := <-c.timer.OutgoingRequestsTimeFrames:
			{
				select {
				case c.ObserversSender.OutgoingRequests <- outgoingRequestTimeFrames:
					{

					}
				default:
					c.log().Error("can't send request `time frames` to the observers sender")
				}
			}

		case outgoingResponseTimeFrame := <-c.timer.OutgoingResponsesTimeFrame:
			{
				select {
				case c.ObserversSender.OutgoingResponses <- outgoingResponseTimeFrame:
					{

					}
				default:
					c.log().Error("can't send response `time frames` to the observers sender")
				}
			}

		// Incoming data flow
		case incomingProposedBlock := <-c.ObserversReceiver.BlocksProposed:
			{
				select {
				case c.BlocksProducer.IncomingBlocksProposals <- incomingProposedBlock:
					{

					}
				default:
					c.log().Error("can't send proposed block to the blocks producer")
				}
			}

		case incomingBlockSignature := <-c.ObserversReceiver.BlockSignatures:
			{
				select {
				case c.BlocksProducer.IncomingBlocksSignatures <- incomingBlockSignature:
					{

					}
				default:
					c.log().Error("can't send proposed block signature to the blocks producer")
				}
			}

		case incomingRequest := <-c.ObserversReceiver.Requests:
			{
				c.processIncomingRequest(incomingRequest, errors)
			}

		case incomingResponse := <-c.ObserversReceiver.Responses:
			{
				c.processIncomingResponse(incomingResponse, errors)
			}
		}
	}
}

func (c *Core) processIncomingRequest(r requests.Request, errors chan error) {
	switch r.(type) {
	case *requests.RequestTimeFrames:
		{
			select {
			case c.timer.IncomingRequestsTimeFrames <- r.(*requests.RequestTimeFrames):
				{

				}
			default:
				c.log().Error("can't send request `time frames` to the timer")
			}
		}
	}
}

func (c *Core) processIncomingResponse(r responses.Response, errors chan error) {
	switch r.(type) {
	case *responses.ResponseTimeFrame:
		{
			select {
			case c.timer.IncomingResponsesTimeFrame <- r.(*responses.ResponseTimeFrame):
				{

				}
			default:
				c.log().Error("can't send response `time frames` to the timer")
			}
		}
	}
}

func (c *Core) exitIfError(errors <-chan error) {
	select {
	case err := <-errors:
		{
			if err != nil {
				log.Fatal(err, "Exit")
			}
		}
	}
}

func (c *Core) log() *log.Entry {
	return log.WithFields(log.Fields{"subsystem": "Core"})
}
