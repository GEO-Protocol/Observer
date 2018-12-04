package chain

import (
	"geo-observers-blockchain/core/chain/geo"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/crypto/keystore"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

const (
	AverageBlockTimeout = time.Second * 5
)

// todo: test for blocks collisions
// todo: test if 2 concurrent blocks collected relatively equal amount of approves
type BlocksProducer struct {

	// Outgoing flows
	OutgoingBlocksProposals chan *BlockProposal

	// Incoming flows

	// Collects proposed blocks from other observers.
	IncomingBlocksProposals chan *BlockProposal

	Settings   *settings.Settings
	ClaimsPool *ClaimsPool
	TSLsPool   *TSLsPool
	Chain      *Chain
	Keystore   *keystore.KeyStore

	ObserversConfigurationReporter *external.Reporter

	// Position of the current observer in received observers configuration.
	orderPosition uint16

	// todo: adjust after each block inserted
	nextBlockAllowedAfter time.Time

	proposedBlock *BlockProposal
}

func NewBlocksProducer(
	conf *settings.Settings,
	reporter *external.Reporter,
	keystore *keystore.KeyStore) *BlocksProducer {

	return &BlocksProducer{
		OutgoingBlocksProposals: make(chan *BlockProposal, 1),

		IncomingBlocksProposals: make(chan *BlockProposal, 1),

		Settings:   conf,
		ClaimsPool: NewClaimsPool(),
		TSLsPool:   NewTSLsPool(),
		Chain:      NewChain(),
		Keystore:   keystore,

		ObserversConfigurationReporter: reporter,
	}
}

func (p *BlocksProducer) Run(errors chan<- error) {

	handleRuntimeErrorIfPresent := func(err error) {
		if err != nil {
			log.Errorln(err)
			time.Sleep(time.Second * 5)
		}
	}

	var currentObserversConfigurationRev uint64 = math.MaxUint64

	for {
		obsConf := p.fetchObserversConfiguration()
		if obsConf.Revision != currentObserversConfigurationRev {
			// Observers configuration has changed.
			// As a result - observers order has changed as well.
			currentObserversConfigurationRev = obsConf.Revision
			p.resetObserversOrder(obsConf)
		}

		if p.shouldGenerateNextBlock(obsConf) {
			err := p.generateNextBlockProposal(obsConf)
			handleRuntimeErrorIfPresent(err)

			p.distributeBlockProposal()

		} else {
			// todo: process responses from other observers

			p.processBlocksFromOtherObservers()
			p.processBlocksSignaturesFromOtherObservers()

		}
	}

	//
	//
	//// todo: check if configuration changed - restart block producing
	//
	//for {
	//	// todo: sync with other observers (majority)
	//
	//	if ()
	//}

	// ExternalObserversBlockSignatures <-chan BlockSignatures, ExternalObserversBlocks <-chan SignedBlock, ExternalObserversBlocksProposed <-chan BlockProposal
}

// todo implement
func (p *BlocksProducer) shouldGenerateNextBlock(conf *external.Configuration) bool {

	// Returns "true" if next block should be generated.
	//
	// Calculates and updates approximate time when next block (after current one)
	// should be available to generate, to prevent redundant blocks generation and observers races.
	allowNextBlock := func() bool {
		p.nextBlockAllowedAfter = time.Now().Add(p.approximateRoundTimeout(conf))
		return true
	}

	// No next block should be generated,
	// until previous candidate block would not be dropped or approved.
	if p.hasProposedBlock() {
		return false
	}

	// If no appropriate timeout was spend from the last block generation -
	// prevent attempts to generate new block.
	if time.Now().Before(p.nextBlockAllowedAfter) {
		return false
	}

	// If next block number corresponds with current observer order number -
	// then the next block should be generated.
	var k = p.Chain.Height() / uint64(len(conf.Observers))
	if k == uint64(p.orderPosition) {
		return allowNextBlock()
	}

	// Prevent observers that already generated the block,
	// or has an ability to do it from generation of the block.
	if uint64(p.orderPosition) > k {

		timeWindow := AverageBlockTimeout * time.Duration(p.orderPosition)
		allowedAfter := p.Chain.LastBlockTimestamp.Add(timeWindow)

		// In case if no candidate block is present
		// and now is the time of the observer to generate the block - allow block generation.
		// (the time is related to the observer order in observers configuration).
		if time.Now().After(allowedAfter) {
			return allowNextBlock()
		}
	}

	return false
}

func (p *BlocksProducer) generateNextBlockProposal(conf *external.Configuration) (err error) {

	if p.hasProposedBlock() {
		return common.ErrAttemptToGenerateRedundantBlockProposal
	}

	// todo: include some data from mem pool
	claims := geo.Claims{}
	tsls := geo.TransactionSignaturesLists{}

	if p.Settings.Debug == false {
		if claims.Count == 0 && tsls.Count == 0 {
			// No new block should be generated in case if there is no info for it.
			err = nil
			return
		}
	}

	nextBlockHeight := p.Chain.Height() + 1

	blockProposal, err := NewBlockProposal(nextBlockHeight, conf.Hash(), claims, tsls)
	if err != nil {
		return
	}

	blockProposal.AuthorSignature, err = p.Keystore.SignHash(blockProposal.Hash)
	if err != nil {
		return
	}

	p.proposedBlock = blockProposal
	return
}

func (p *BlocksProducer) distributeBlockProposal() {
	// In case if current observer mem pool has no content for the next block -
	// generateNextBlockProposal() would not generate new block,
	// but the distributeBlockProposal() might be called.
	if p.proposedBlock != nil {
		p.OutgoingBlocksProposals <- p.proposedBlock
	}
}

func (p *BlocksProducer) fetchObserversConfiguration() *external.Configuration {
	for {
		conf, err := p.ObserversConfigurationReporter.GetCurrentConfiguration()
		if err != nil || conf == nil {
			p.log().Warn("Can't fetch observers configuration. Waiting 5 seconds.")
			time.Sleep(5 * time.Second) // todo: add exponential time to wait

		} else {
			return conf

		}
	}
}

func (p *BlocksProducer) processBlocksFromOtherObservers() {
	select {
	// todo: implement blocks fetching

	case blockProposed := <-p.IncomingBlocksProposals:
		{
			p.log().Info("validate block", blockProposed)
		}

	case _ = <-time.After(AverageBlockTimeout):
		{

			// todo: temp
			p.proposedBlock = nil

			return
		}
	}
}

func (p *BlocksProducer) resetObserversOrder(conf *external.Configuration) {
	for i, observer := range conf.Observers {
		if observer.Host == p.Settings.Observers.GNS.Host && observer.Port == p.Settings.Observers.GNS.Port {
			p.orderPosition = uint16(i)
		}
	}
}

// approximateRoundTimeout reports approximate duration of one round, based on current observers configuration.
func (p *BlocksProducer) approximateRoundTimeout(conf *external.Configuration) time.Duration {
	totalObserversCount := len(conf.Observers)
	return AverageBlockTimeout * time.Duration(totalObserversCount)
}

func (p *BlocksProducer) hasProposedBlock() bool {
	if p.proposedBlock != nil {
		return true
	}

	if len(p.OutgoingBlocksProposals) != 0 {
		return true
	}

	return false
}

func (p *BlocksProducer) log() *log.Entry {
	return log.WithFields(log.Fields{"subsystem": "BlockProducer"})
}

func (p *BlocksProducer) processBlocksSignaturesFromOtherObservers() {
	//panic("not implemented")
}
