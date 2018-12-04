package core

import (
	"geo-observers-blockchain/core/chain"
	"geo-observers-blockchain/core/crypto/keystore"
	geoNet "geo-observers-blockchain/core/network/communicator/geo"
	observersNet "geo-observers-blockchain/core/network/communicator/observers"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
)

type Core struct {
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

	reporter := external.NewReporter()
	core = &Core{
		Settings: conf,
		Keystore: k,

		ObserversConfReporter: reporter,
		ObserversSender:       observersNet.NewSender(conf, reporter),
		ObserversReceiver:     observersNet.NewReceiver(),
		GEONodesCommunicator:  geoNet.NewNodesCommunicator(),

		BlocksProducer: chain.NewBlocksProducer(conf, reporter, k),
	}
	return
}

func (c *Core) Run() {
	errors := make(chan error)
	c.initialiseConnectionsHandlers(errors)
	c.initialiseProcessing(errors)

	// todo: begin monitor errors
	for {
		select {
		case err := <-errors:
			{
				if err != nil {
					c.log().Error(err)
				}
			}
		}
	}
}

func (c *Core) initialiseConnectionsHandlers(errors chan error) {
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

	go c.handleGEONodesRequests()
	go c.handleObserversCommunication()
}

func (c *Core) initialiseProcessing(errors chan error) {
	go c.BlocksProducer.Run(errors)
	//c.exitIfError(errors)

	go c.dispatchDataFlows(errors)
}

func (c *Core) handleGEONodesRequests() {
	// todo: implement
	select {}
}

func (c *Core) handleObserversCommunication() {
	// todo: implement
	select {}
}

func (c *Core) dispatchDataFlows(errors chan error) {
	for {
		select {

		// Outgoing data flow
		case outgoingBlockProposed := <-c.BlocksProducer.OutgoingBlocksProposals:
			{
				c.ObserversSender.OutgoingBlockProposal <- outgoingBlockProposed
			}

		// Incoming data flow
		case incomingProposedBlock := <-c.ObserversReceiver.BlocksProposed:
			{
				c.BlocksProducer.IncomingBlocksProposals <- incomingProposedBlock
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
