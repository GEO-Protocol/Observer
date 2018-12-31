package chain

import (
	"errors"
	"geo-observers-blockchain/core/chain/pool"
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common"
	e2 "geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/crypto/keystore"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/network/messages"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/timer"
	log "github.com/sirupsen/logrus"
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
	OutgoingBlocksProposals  chan *ProposedBlockData
	OutgoingBlocksSignatures chan *messages.SignatureMessage

	// Incoming flows
	IncomingBlocksProposals  chan *ProposedBlockData
	IncomingBlocksSignatures chan *messages.SignatureMessage

	// Incoming events
	IncomingEventTimeFrameEnded chan *timer.EventTimeFrameEnd

	// Pools
	tslsPool   *pool.Pool
	claimsPool *pool.Pool

	// Internal
	Settings *settings.Settings
	Chain    *Chain
	Keystore *keystore.KeyStore

	ObserversConfigurationReporter *external.Reporter

	//// Position of the current observer in received observers configuration.
	//orderPosition uint16
	//
	//// todo: adjust after each block inserted
	//nextBlockAllowedAfter time.Time

	proposedBlock *ProposedBlockData

	// todo: make it possible to collect signatures for several blocks, by their hashes.
	//       (defence from signatures holding attack)
	//
	// Collects signatures from other observers for the proposed block.
	// The format: <observer index -> signature for current proposed block>
	//
	// On each one observers configuration update -
	// this list would automatically resize to fit current observers configuration.
	proposedBlockSignatures *signatures.IndexedObserversSignatures
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
		OutgoingBlocksProposals:  make(chan *ProposedBlockData, ChannelBufferSize),
		OutgoingBlocksSignatures: make(chan *messages.SignatureMessage, ChannelBufferSize),

		IncomingBlocksProposals:  make(chan *ProposedBlockData, ChannelBufferSize),
		IncomingBlocksSignatures: make(chan *messages.SignatureMessage, ChannelBufferSize),

		IncomingEventTimeFrameEnded: make(chan *timer.EventTimeFrameEnd),

		tslsPool:   pool.NewPool(),
		claimsPool: pool.NewPool(),
		Settings:   conf,
		Chain:      chain,
		Keystore:   keystore,

		ObserversConfigurationReporter: reporter,
	}
	return
}

func (p *BlocksProducer) Run(errors chan<- error) {
	// todo: ensure chain is in sync with the majority of the observers.

	currentObserversConf, err := p.ObserversConfigurationReporter.GetCurrentConfiguration()
	p.resetObserversConfiguration(currentObserversConf)
	if err != nil {
		// todo: report fatal error instead of panic
		panic(err)
	}

	for {
		select {
		// todo: add case when observers configuration has changed

		case tick := <-p.IncomingEventTimeFrameEnded:
			{
				err := p.processTick(tick, currentObserversConf)
				p.handleRuntimeErrorIfPresent(err)
			}
		}
	}
}

func (p *BlocksProducer) processTick(tick *timer.EventTimeFrameEnd, conf *external.Configuration) (err error) {
	p.log().WithFields(
		log.Fields{
			"Index": tick.Index}).Debug(
		"Time frame changed")

	if tick.Index == conf.CurrentObserverIndex {
		return p.processBlockGenerationFlow(tick, conf)
	}

	return p.processValidationFlow(tick, conf)
}

func (p *BlocksProducer) processBlockGenerationFlow(
	tick *timer.EventTimeFrameEnd, conf *external.Configuration) (err error) {

	err = p.generateNextProposedBlock(conf)
	if err != nil {
		return
	}

	p.distributeProposedBlock()
	return p.collectSignaturesForProposedBlockAndProcessConsensus(tick, conf)
}

func (p *BlocksProducer) processValidationFlow(
	tick *timer.EventTimeFrameEnd, observersConf *external.Configuration) (err error) {

	select {
	case block := <-p.IncomingBlocksProposals:
		{
			return p.processIncomingProposedBlock(block)
		}

	// todo: signatures lists, not signature alone
	case _ = <-p.IncomingBlocksSignatures:
		{
			return
		}

	case _ = <-time.After(p.silencePeriodTimeLeft(tick)):
		{
			return p.processSilencePeriod()
		}

		// todo: process case when configuration changed,
		//       to be able to interrupt flow and react ASAP
	}
}

func (p *BlocksProducer) generateNextProposedBlock(conf *external.Configuration) (err error) {

	if p.hasProposedBlock() {
		return e2.ErrAttemptToGenerateRedundantProposedBlock
	}

	if p.proposedBlockSignatures != nil {
		return e2.ErrAttemptToGenerateRedundantProposedBlock
	}

	// todo: include some data from mem pool
	claims := &geo.Claims{}
	tsls := &geo.TransactionSignaturesLists{}

	if p.Settings.Debug == false {
		if claims.Count() == 0 && tsls.Count() == 0 {
			// No new block should be generated in case if there is no info for it.
			// (except debug mode)
			return
		}
	}

	nextBlockHeight := p.Chain.Height() + 1
	blockProposal, err := NewProposedBlockData(nextBlockHeight, conf.CurrentObserverIndex, conf.Hash(), claims, tsls)
	if err != nil {
		return
	}

	p.proposedBlock = blockProposal
	p.log().WithFields(
		log.Fields{
			"height":       p.proposedBlock.Height,
			"claims_count": p.proposedBlock.ClaimsHashes.Count(),
			"tsls_count":   p.proposedBlock.TSLsHashes.Count()}).Debug(
		"Proposed block generated.")
	return
}

func (p *BlocksProducer) distributeProposedBlock() {
	// In case if current observer mem pool has no content for the next block -
	// generateNextProposedBlock() would not generate new block,
	// but the distributeProposedBlock() might be called.
	if p.proposedBlock != nil {
		p.OutgoingBlocksProposals <- p.proposedBlock
	}
}

func (p *BlocksProducer) collectSignaturesForProposedBlockAndProcessConsensus(
	tick *timer.EventTimeFrameEnd, conf *external.Configuration) (err error) {

	var (
		ErrLoopBreak = errors.New("")
	)

	processSignatureCollection := func() (err error) {
		select {
		case sig := <-p.IncomingBlocksSignatures:
			{
				err = p.processIncomingProposedBlockSignature(sig, conf)
				if err != nil {
					return
				}

				if p.isConsensusHasBeenReachedOnProposedBlock() {
					err = p.propagateCollectedSignatures()
					if err != nil {
						return
					}

					err = p.commitProposedBlock()
					if err != nil {
						// todo: process collision
						// todo: use internal event loop for notification about synchronisation needed
						return
					}

					return ErrLoopBreak

				} else {
					// Wait for another signature from another observer.
					return nil
				}
			}

		case _ = <-time.After(p.silencePeriodTimeLeft(tick)):
			{
				return p.processSilencePeriod()
			}

			// todo: process case when configuration changed,
			//       to be able to interrupt flow and react ASAP
		}
	}

	for {
		if p.proposedBlock != nil {
			err = processSignatureCollection()
			if err != nil {
				if err == ErrLoopBreak {
					return nil

				} else {
					return err
				}
			}
		}
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
	case _ = <-time.After(common.AverageBlockGenerationTimeRange - common.BlockGenerationSilencePeriod):
		{

			// todo: temp
			p.proposedBlock = nil
			return
		}
	}
}

func (p *BlocksProducer) resetObserversConfiguration(conf *external.Configuration) {

	//// Observers configuration has been changed.
	//// Registry position of the current observer might change.
	//// Now it must be reset in accordance to new configuration.
	//for i, observer := range conf.Observers {
	//	if observer.Host == p.settings.Observers.GNS.Host && observer.Port == p.settings.Observers.GNS.Port {
	//		p.orderPosition = uint16(i)
	//	}
	//}

	// Due to configuration change -
	// previous signatures lists for the blocks are invalid now.
	// Signatures registry must reset.
	p.proposedBlockSignatures = signatures.NewIndexedObserversSignatures(len(conf.Observers))
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

func (p *BlocksProducer) validateProposedBlock(block *ProposedBlockData) (valid bool, err error) {
	if block == nil {
		return false, e2.NilParameter
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
		return false, e2.NilParameter
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

	h, err := p.proposedBlock.Hash()
	if err != nil {
		return false, err
	}

	isValid = p.Keystore.CheckExternalSignature(h, *sig.Signature, senderObserver.PubKey)
	return
}

func (p *BlocksProducer) processIncomingProposedBlock(block *ProposedBlockData) (err error) {
	if block == nil {
		return e2.NilParameter
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

func (p *BlocksProducer) signBlockAndPropagateSignature(block *ProposedBlockData) (err error) {
	h, err := p.proposedBlock.Hash()
	if err != nil {
		return err
	}

	signature, err := p.Keystore.SignHash(h)
	if err != nil {
		return err
	}

	// todo: remember block hash, number and signature on the disk
	// todo: prevent double signing of the same block number

	p.OutgoingBlocksSignatures <- &messages.SignatureMessage{
		Signature: signature, AddresseeObserverIndex: block.AuthorPosition}

	context := log.Fields{"Height": block.Height}
	p.log().WithFields(context).Info("Proposed block signed")
	return
}

func (p *BlocksProducer) processIncomingProposedBlockSignature(
	sig *messages.SignatureMessage, conf *external.Configuration) (err error) {

	if sig == nil {
		return e2.NilParameter
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
	}

	return
}

func (p *BlocksProducer) processSilencePeriod() error {
	// Block generation period or block validation period has been finished.
	// Event if proposed block is present and has collected some signatures -
	// it must be dropped.
	p.proposedBlock = nil
	p.proposedBlockSignatures = nil

	return nil
}

func (p *BlocksProducer) silencePeriodTimeLeft(tick *timer.EventTimeFrameEnd) time.Duration {
	return time.Now().Sub(tick.SilencePeriodTimestamp)
}

func (p *BlocksProducer) commitProposedBlock() (err error) {
	bs := &BlockSigned{
		Data:       p.proposedBlock,
		Signatures: p.proposedBlockSignatures,
	}

	err = p.Chain.Append(bs)
	if err != nil {
		return
	}

	p.proposedBlock = nil
	p.proposedBlockSignatures = nil
	return
}

func (p *BlocksProducer) propagateCollectedSignatures() (err error) {
	p.log().Warn("propagateCollectedSignatures is NOT IMPLEMENTED")

	return nil
}

func (p *BlocksProducer) isConsensusHasBeenReachedOnProposedBlock() bool {
	totalApprovesCount := 0

	for _, sig := range p.proposedBlockSignatures.Signatures {
		if sig != nil {
			totalApprovesCount++
		}
	}

	return totalApprovesCount >= common.ObserversConsensusCount
}

func (p *BlocksProducer) handleRuntimeErrorIfPresent(err error) {
	if err != nil {
		//p.errorsFlow <- err
		time.Sleep(time.Second * 5)
	}
}

func (p *BlocksProducer) log() *log.Entry {
	return log.WithFields(log.Fields{"subsystem": "BlockProducer"})
}
