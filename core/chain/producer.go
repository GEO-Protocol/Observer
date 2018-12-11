package chain

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/crypto/keystore"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/network/messages"
	"geo-observers-blockchain/core/settings"
	log "github.com/sirupsen/logrus"
	"math"
	"time"
)

const (
	ChannelBufferSize = 1
)

// todo: test for blocks collisions
// todo: test if 2 concurrent blocks collected relatively equal amount of approves
// todo: consider including signatures under the block hash,
//  this would really increase consensus complexity,
//  but would guarantee signatures constraint

// todo: prevent claims collisions in blockchain
type BlocksProducer struct {

	// Outgoing flows
	OutgoingBlocksProposals  chan *ProposedBlock
	OutgoingBlocksSignatures chan *messages.SignatureMessage

	// Incoming flows
	IncomingBlocksProposals  chan *ProposedBlock
	IncomingBlocksSignatures chan *messages.SignatureMessage

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

	proposedBlock *ProposedBlock

	// todo: make it possible to collect signatures for several blocks, by their hashes.
	//       (defence from signatures holding attack)
	//
	// Collects signatures from other observers for the proposed block.
	// The format: <observer index -> signature for current proposed block>
	//
	// On each one observers configuration update -
	// this list would automatically resize to fit current observers configuration.
	proposedBlockSignatures *IndexedObserversSignatures

	// External errors flow
	errorsFlow chan<- error
}

func NewBlocksProducer(
	conf *settings.Settings,
	reporter *external.Reporter,
	keystore *keystore.KeyStore) (producer *BlocksProducer, err error) {

	chain, err := NewChain()
	if err != nil {
		return
	}

	producer = &BlocksProducer{
		// Outgoing data flow
		OutgoingBlocksProposals:  make(chan *ProposedBlock, ChannelBufferSize),
		OutgoingBlocksSignatures: make(chan *messages.SignatureMessage, ChannelBufferSize),

		// Incoming data flow
		IncomingBlocksProposals:  make(chan *ProposedBlock, ChannelBufferSize),
		IncomingBlocksSignatures: make(chan *messages.SignatureMessage, ChannelBufferSize),

		Settings:   conf,
		ClaimsPool: NewClaimsPool(),
		TSLsPool:   NewTSLsPool(),
		Chain:      chain,
		Keystore:   keystore,

		ObserversConfigurationReporter: reporter,
	}
	return
}

func (p *BlocksProducer) Run(errors chan<- error) {

	// todo: sync with majority of other observers.

	// todo: remove this
	// Attach received errors flow to the internal channel,
	// to prevent redundant errors channel propagation.
	p.errorsFlow = errors

	// Revision number of current observers configuration.
	// Observers configuration might change on the fly,
	// to catch the moment of change - revision number is used.
	var currentObsConfRev uint64 = math.MaxUint64

	for {
		obsConf := p.fetchObserversConfiguration()
		if obsConf.Revision != currentObsConfRev {
			// Observers configuration has changed.
			// As a result - observers order has changed as well
			// so observers need to be reconfigured.
			currentObsConfRev = obsConf.Revision
			p.resetObserversConfiguration(obsConf)
		}

		if p.shouldGenerateNextBlock(obsConf) {
			err := p.generateNextBlockProposal(obsConf)
			p.handleRuntimeErrorIfPresent(err)

			p.distributeBlockProposal()

		} else {
			// Process responses and other data from other observers.
			err := p.processBlocksAndResponsesFromOtherObservers(obsConf)
			p.handleRuntimeErrorIfPresent(err)
		}
	}
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

	// todo: обзервер не може генерувати блоки, якщо він перед точкою генерації
	//  якщо останній обзеревер не генеруаватиме блок - генерація блоків зупиниться.
	//  потрібна формула для коректного прорахунку часу на генерацію блоку.

	// Prevent observers that already generated the block,
	// or has an ability to do it from generation of the block.
	if uint64(p.orderPosition) > k {

		timeWindow := common.AverageBlockGenerationTimeRange * time.Duration(p.orderPosition)
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
		return common.ErrAttemptToGenerateRedundantProposedBlock
	}

	// todo: include some data from mem pool
	claims := geo.Claims{}
	tsls := geo.TransactionSignaturesLists{}

	if p.Settings.Debug == false {
		if claims.Count() == 0 && tsls.Count() == 0 {
			// No new block should be generated in case if there is no info for it.
			// (except debug mode)
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

func (p *BlocksProducer) processBlocksAndResponsesFromOtherObservers(conf *external.Configuration) (err error) {
	select {
	// todo: implement blocks fetching

	case block := <-p.IncomingBlocksProposals:
		{
			err = p.processIncomingProposedBlock(block)
			return
		}

	case sig := <-p.IncomingBlocksSignatures:
		{
			err = p.processIncomingProposedBlockSignature(sig, conf)
			return
		}

	// todo: delta for next block time point must be here
	case _ = <-time.After(common.AverageBlockGenerationTimeRange):
		{

			// todo: temp
			p.proposedBlock = nil
			return
		}
	}
}

func (p *BlocksProducer) resetObserversConfiguration(conf *external.Configuration) {

	// Observers configuration has been changed.
	// Registry position of the current observer might change.
	// Now it must be reset in accordance to new configuration.
	for i, observer := range conf.Observers {
		if observer.Host == p.Settings.Observers.GNS.Host && observer.Port == p.Settings.Observers.GNS.Port {
			p.orderPosition = uint16(i)
		}
	}

	// Due to configuration change -
	// previous signatures lists for the blocks are invalid now.
	// Signatures registry must reset.
	p.proposedBlockSignatures = NewIndexedObserversSignatures(len(conf.Observers))
}

// approximateRoundTimeout reports approximate duration of one round, based on current observers configuration.
func (p *BlocksProducer) approximateRoundTimeout(conf *external.Configuration) time.Duration {
	totalObserversCount := len(conf.Observers)
	return common.AverageBlockGenerationTimeRange * time.Duration(totalObserversCount)
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

func (p *BlocksProducer) validateProposedBlock(block *ProposedBlock) (valid bool, err error) {
	if block == nil {
		return false, common.ErrNilParameter
	}

	// todo: check whose order now to generate the block.
	// todo: check that block was generated by right observer, whose order is now.

	// todo: if block with such number is already included into the chain - reject
	// todo: if block with such number has been rejected earlier - reject
	// todo: check internal block structure

	return true, nil
}

func (p *BlocksProducer) validateProposedBlockSignature(
	sig *messages.SignatureMessage, conf *external.Configuration) (isValid bool, err error) {

	if sig == nil {
		return false, common.ErrNilParameter
	}

	if p.proposedBlock == nil {
		// There is no any pending proposed block.
		return false, nil
	}

	if len(conf.Observers) < int(sig.AddresseeObserverIndex+1) {
		// Current observers configuration has no observer with such index.
		return false, nil
	}

	senderObserver := conf.Observers[sig.AddresseeObserverIndex]
	if !p.Settings.Debug {
		// Prevent validation of it's own signatures,
		// except in debug mode.
		if p.Settings.Observers.Network.Host == senderObserver.Host &&
			p.Settings.Observers.Network.Port == senderObserver.Port {
			return false, nil
		}
	}

	isValid = p.Keystore.CheckExternalSignature(p.proposedBlock.Hash, *sig.Signature, senderObserver.PubKey)
	return
}

func (p *BlocksProducer) processIncomingProposedBlock(block *ProposedBlock) (err error) {
	if block == nil {
		return common.ErrNilParameter
	}

	isBlockValid, err := p.validateProposedBlock(block)
	if err != nil {
		return
	}

	if !isBlockValid {
		return
	}

	return p.signBlockAndPropagateSignature(block)
}

func (p *BlocksProducer) signBlockAndPropagateSignature(block *ProposedBlock) (err error) {
	signature, err := p.Keystore.SignHash(block.Hash)
	if err != nil {
		return err
	}

	// todo: remember block hash, number and signature on the disk
	// todo: prevent double signing of the same block number

	p.OutgoingBlocksSignatures <- &messages.SignatureMessage{
		Signature: signature, AddresseeObserverIndex: block.AuthorPosition}

	context := log.Fields{"Height": block.Height, "Hash": block.Hash.Hex()}
	p.log().WithFields(context).Info("Proposed block signed")
	return
}

func (p *BlocksProducer) processIncomingProposedBlockSignature(
	sig *messages.SignatureMessage, conf *external.Configuration) (err error) {

	if sig == nil {
		return common.ErrNilParameter
	}

	isSignatureValid, err := p.validateProposedBlockSignature(sig, conf)
	if err != nil {
		return
	}

	if isSignatureValid {
		// Append received signature to the signatures list.
		if int(sig.AddresseeObserverIndex+1) < len(p.proposedBlockSignatures.Signatures) {
			p.proposedBlockSignatures.Signatures[sig.AddresseeObserverIndex] = sig.Signature
		}

		// Check consensus.
		totalApproves := 0
		for _, sig := range p.proposedBlockSignatures.Signatures {
			if sig != nil {
				totalApproves++
			}
		}

		// todo: uncoment
		//consensusReached := totalApproves >= common.ObserversConsensusCount

		consensusReached := true
		if consensusReached {
			bs := &BlockSigned{Data: p.proposedBlock, Signatures: p.proposedBlockSignatures}
			err = p.Chain.Append(bs)
			if err != nil {
				return
			}

			// todo: send signatures collected to all other
			err = p.propagateCollectedSignatures()
			if err != nil {
				return
			}

			p.proposedBlock = nil
			p.proposedBlockSignatures = NewIndexedObserversSignatures(len(conf.Observers))
		}
	}

	return
}

func (p *BlocksProducer) propagateCollectedSignatures() (err error) {
	p.log().Warn("propagateCollectedSignatures is NOT IMPLEMENTED")

	return nil
}

func (p *BlocksProducer) handleRuntimeErrorIfPresent(err error) {
	if err != nil {
		p.errorsFlow <- err
		time.Sleep(time.Second * 5)
	}
}

func (p *BlocksProducer) log() *log.Entry {
	return log.WithFields(log.Fields{"subsystem": "BlockProducer"})
}
