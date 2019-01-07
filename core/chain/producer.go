package chain

import (
	"bytes"
	"fmt"
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/pool"
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/crypto/keystore"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"time"
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
	OutgoingRequestsCandidateDigestBroadcast chan *requests.CandidateDigestBroadcast
	IncomingRequestsCandidateDigest          chan *requests.CandidateDigestBroadcast
	OutgoingResponsesCandidateDigestApprove  chan *responses.CandidateDigestApprove
	IncomingResponsesCandidateDigestApprove  chan *responses.CandidateDigestApprove
	OutgoingRequestsBlockSignaturesBroadcast chan *requests.BlockSignaturesBroadcast
	IncomingRequestsBlockSignatures          chan *requests.BlockSignaturesBroadcast

	IncomingEventTimeFrameEnded chan *ticker.EventTimeFrameEnd

	settings   *settings.Settings
	keystore   *keystore.KeyStore
	reporter   *external.Reporter
	poolTSLs   *pool.Handler
	poolClaims *pool.Handler
	chain      *Chain

	nextBlock *block.Signed
}

func NewProducer(
	conf *settings.Settings, reporter *external.Reporter,
	keystore *keystore.KeyStore) (producer *Producer, err error) {

	chain, err := NewChain()
	if err != nil {
		return
	}

	producer = &Producer{
		OutgoingRequestsCandidateDigestBroadcast: make(
			chan *requests.CandidateDigestBroadcast, 1),

		IncomingRequestsCandidateDigest: make(
			chan *requests.CandidateDigestBroadcast, common.ObserversMaxCount-1),

		OutgoingResponsesCandidateDigestApprove: make(
			chan *responses.CandidateDigestApprove, 1),

		IncomingResponsesCandidateDigestApprove: make(
			chan *responses.CandidateDigestApprove, common.ObserversMaxCount-1),

		OutgoingRequestsBlockSignaturesBroadcast: make(
			chan *requests.BlockSignaturesBroadcast, 1),

		IncomingRequestsBlockSignatures: make(
			chan *requests.BlockSignaturesBroadcast, 1),

		IncomingEventTimeFrameEnded: make(chan *ticker.EventTimeFrameEnd, 1),

		settings:   conf,
		reporter:   reporter,
		keystore:   keystore,
		poolTSLs:   pool.NewHandler(reporter),
		poolClaims: pool.NewHandler(reporter),
		chain:      chain,
	}
	return
}

func (p *Producer) Run(errors chan<- error) {
	go p.poolClaims.Run(errors)
	go p.poolTSLs.Run(errors)

	conf, err := p.reporter.GetCurrentConfiguration()
	if err != nil {
		// todo: report fatal error instead of panic
		panic(err)
	}

	for {
		select {
		// todo: add case when observers configuration has changed

		case tick := <-p.IncomingEventTimeFrameEnded:
			p.handleErrorIfAny(
				p.processTick(tick, conf))
		}
	}
}

func (p *Producer) processTick(tick *ticker.EventTimeFrameEnd, conf *external.Configuration) (err error) {
	defer func() {
		// Drop previously generated block candidate and all related data in any case.
		//
		// WARN:
		// Do not rewrite error!
		// It might be necessary.
		_ = p.processFinalStage()
	}()

	if p.settings.Debug {
		p.log().WithFields(log.Fields{
			"Index": tick.Index,
		}).Debug("Time frame changed")
	}

	currentObserverMustGenerateBlock := tick.Index == conf.CurrentObserverIndex
	if currentObserverMustGenerateBlock {
		return p.processBlockGenerationFlow(tick, conf)

	} else {
		return p.processValidationFlow(tick, conf)
	}
}

func (p *Producer) processBlockGenerationFlow(tick *ticker.EventTimeFrameEnd,
	conf *external.Configuration) (err error) {
	candidate, err := p.generateBlockCandidateFromPool(conf)
	if err != nil {
		return
	}

	p.nextBlock = &block.Signed{
		Body:       candidate,
		Signatures: signatures.NewIndexedObserversSignatures(common.ObserversMaxCount),
	}

	// Signing the block
	sig, err := p.keystore.SignHash(p.nextBlock.Body.Hash)
	p.nextBlock.Signatures.At[conf.CurrentObserverIndex] = sig

	err = p.distributeCandidateDigests()
	if err != nil {
		return
	}

	return p.collectResponsesAndProcessConsensus(tick, conf)
}

func (p *Producer) processValidationFlow(
	tick *ticker.EventTimeFrameEnd, observersConf *external.Configuration) (err error) {

	for {
		select {
		// todo: process case when configuration changed,
		//       to be able to interrupt flow and react ASAP

		case requestDigestBroadcast := <-p.IncomingRequestsCandidateDigest:
			err = p.processIncomingDigest(requestDigestBroadcast, tick, observersConf)
			if err != nil {
				// todo: report error and DO NOT STOP THE METHOD
			}
			continue

		case requestBlockSignatures := <-p.IncomingRequestsBlockSignatures:
			err = p.processReceivedBlockSignatures(requestBlockSignatures, tick, observersConf)
			if err != nil {
				// todo: report error and DO NOT STOP THE METHOD
			}
			continue

		case <-time.After(p.finalStageTimeLeft(tick)):
			return
		}
	}
}

func (p *Producer) processIncomingDigest(
	request *requests.CandidateDigestBroadcast,
	tick *ticker.EventTimeFrameEnd,
	conf *external.Configuration) (err error) {

	err = p.validateReceivedBlockCandidateDigest(request.Digest, tick, conf)
	if err != nil {
		return
	}

	candidate, err := p.generateBlockCandidateFromDigest(request.Digest, conf)
	if err != nil {
		return
	}

	err = p.validateBlockCandidateDigestAndRelatedBlockCandidate(request.Digest, candidate)
	if err != nil {
		return
	}

	return p.approveBlockCandidateAndPropagateSignature(request, candidate, conf)
}

func (p *Producer) processIncomingCandidateDigestApprove(
	response *responses.CandidateDigestApprove,
	conf *external.Configuration) (err error) {

	if response == nil || conf == nil {
		return errors.NilParameter
	}

	err = p.validateCandidateDigestSignatureResponse(response, conf)
	if err != nil {
		return
	}

	// Append received signature to the signatures list.
	p.nextBlock.Signatures.At[response.ObserverIndex()] = &response.Signature

	return
}

func (p *Producer) processReceivedBlockSignatures(
	request *requests.BlockSignaturesBroadcast,
	tick *ticker.EventTimeFrameEnd,
	conf *external.Configuration) (err error) {

	err = p.validateBlockSignaturesRequest(request, conf)
	if err != nil {
		return
	}

	p.nextBlock.Signatures = request.Signatures
	return p.commitBlock(tick)
}

func (p *Producer) processFinalStage() error {
	// Signed generation period or block validation period has been finished.
	// Even if proposed block is present and has collected some signatures - it MUST be dropped.
	p.nextBlock = nil
	return nil
}

func (p *Producer) generateBlockCandidateFromPool(
	conf *external.Configuration) (candidate *block.Body, err error) {

	// todo: move to separate method
	getBlockReadyTSLs := func() (tsls *geo.TransactionSignaturesLists, err error) {
		channel, errorsChannel := p.poolTSLs.BlockReadyInstances()

		tsls = &geo.TransactionSignaturesLists{}
		tsls.At = make([]*geo.TransactionSignaturesList, 0, 0) // todo: create constructor for it

		select {
		case i := <-channel:
			for _, instance := range i.At {
				tsls.At = append(tsls.At, instance.(*geo.TransactionSignaturesList))
			}

		case _ = <-errorsChannel:
			err = errors.TSLsPoolReadFailed

		case <-time.After(time.Second):
			err = errors.TSLsPoolReadFailed
		}

		return
	}

	// todo: move to separate method
	getBlockReadyClaims := func() (claims *geo.Claims, err error) {
		channel, errorsChannel := p.poolTSLs.BlockReadyInstances()

		claims = &geo.Claims{}
		claims.At = make([]*geo.Claim, 0, 0)

		select {
		case i := <-channel:
			for _, instance := range i.At {
				claims.At = append(claims.At, instance.(*geo.Claim))
			}

		case _ = <-errorsChannel:
			err = errors.TSLsPoolReadFailed

		case <-time.After(time.Second):
			err = errors.ClaimsPoolReadFailed
		}

		return
	}

	tsls, err := getBlockReadyTSLs()
	if err != nil {
		return
	}

	claims, err := getBlockReadyClaims()
	if err != nil {
		return
	}

	return p.generateBlockCandidate(tsls, claims, conf, conf.CurrentObserverIndex)
}

// todo: this method is very similar to the the original block generation
func (p *Producer) generateBlockCandidateFromDigest(
	digest *block.Digest, conf *external.Configuration) (candidate *block.Body, err error) {

	// todo: move to separate method
	getBlockReadyTSLs := func() (tsls *geo.TransactionSignaturesLists, err error) {
		channel, errorsChannel := p.poolTSLs.BlockReadyInstancesByHashes(digest.TSLsHashes.At)

		tsls = &geo.TransactionSignaturesLists{}
		select {
		case i := <-channel:
			for _, instance := range i.At {
				tsls.At = append(tsls.At, instance.(*geo.TransactionSignaturesList))
			}

		case _ = <-errorsChannel:
			err = errors.TSLsPoolReadFailed

		case <-time.After(time.Second):
			err = errors.TSLsPoolReadFailed
		}

		return
	}

	// todo: move to separate method
	getBlockReadyClaims := func() (claims *geo.Claims, err error) {
		channel, errorsChannel := p.poolTSLs.BlockReadyInstancesByHashes(digest.ClaimsHashes.At)

		claims = &geo.Claims{}
		select {
		case i := <-channel:
			for _, instance := range i.At {
				claims.At = append(claims.At, instance.(*geo.Claim))
			}

		case _ = <-errorsChannel:
			err = errors.TSLsPoolReadFailed

		case <-time.After(time.Second):
			err = errors.ClaimsPoolReadFailed
		}

		return
	}

	tsls, err := getBlockReadyTSLs()
	if err != nil {
		return
	}

	claims, err := getBlockReadyClaims()
	if err != nil {
		return
	}

	return p.generateBlockCandidate(tsls, claims, conf, digest.AuthorObserverIndex)
}

func (p *Producer) generateBlockCandidate(
	tsls *geo.TransactionSignaturesLists, claims *geo.Claims,
	conf *external.Configuration,
	authorObserverPosition uint16) (candidate *block.Body, err error) {

	if p.hasProposedBlock() {
		return nil, errors.AttemptToGenerateRedundantBlock
	}

	nextIndex := p.chain.Height()
	previousBlock, err := p.chain.At(nextIndex - 1)
	if err != nil {
		return
	}

	candidate = &block.Body{
		Index:               nextIndex,
		ExternalChainHeight: conf.CurrentExternalChainHeight(),
		AuthorObserverIndex: authorObserverPosition,
		ObserversConfHash:   conf.Hash(),
		Claims:              claims,
		TSLs:                tsls,
	}

	err = candidate.SortInternalSequences()
	if err != nil {
		return
	}

	err = candidate.UpdateHash(previousBlock.Body.Hash)
	if err != nil {
		return
	}

	if p.settings.Debug {
		p.log().WithFields(log.Fields{
			"Index": candidate.Index,
			"Hash":  candidate.Hash.Hex(),
		}).Debug("Candidate block generated")
	}

	return candidate, nil
}

func (p *Producer) approveBlockCandidateAndPropagateSignature(
	request *requests.CandidateDigestBroadcast,
	candidate *block.Body,
	conf *external.Configuration) (err error) {

	p.nextBlock = &block.Signed{
		Body:       candidate,
		Signatures: signatures.NewIndexedObserversSignatures(common.ObserversMaxCount),
	}

	signature, err := p.keystore.SignHash(p.nextBlock.Body.Hash)
	if err != nil {
		return err
	}

	if p.settings.Debug {
		p.log().WithFields(log.Fields{
			"BlockHash":     p.nextBlock.Body.Hash.Hex(),
			"Signature (S)": signature.S,
			"Signature (R)": signature.R,
		}).Debug("Block digest approved")
	}

	// todo: remember block hash, number and signature on the disk
	// todo: prevent double signing of the same candidate with the same attempt

	select {
	case p.OutgoingResponsesCandidateDigestApprove <- responses.NewCandidateDigestApprove(
		request, conf.CurrentObserverIndex, *signature):
	default:
		err = errors.ChannelTransferringFailed
	}

	if p.settings.Debug {
		context := log.Fields{
			"Index":   request.Digest.Index,
			"Attempt": request.Digest.Attempt}
		p.log().WithFields(context).Debug("Signed candidate digest signed")
	}

	return
}

func (p *Producer) collectResponsesAndProcessConsensus(
	tick *ticker.EventTimeFrameEnd, conf *external.Configuration) (err error) {

	if p.nextBlock == nil {
		return errors.NotFound
	}

	var ErrLoopBreak = utils.Error("", "")
	processNextArrivedApproveOrTimeout := func() (err error) {
		select {
		case responseApprove := <-p.IncomingResponsesCandidateDigestApprove:
			err = p.processIncomingCandidateDigestApprove(responseApprove, conf)
			if err != nil {
				return
			}

			if p.nextBlock.Signatures.IsMajorityApprovesCollected() {
				err = p.distributeCollectedSignatures()
				if err != nil {
					return
				}

				err = p.commitBlock(tick)
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

		case <-time.After(p.finalStageTimeLeft(tick)):
			_ = p.processFinalStage()
			return ErrLoopBreak

			// todo: process case when configuration changed,
			//       to be able to interrupt flow and react ASAP
		}
	}

	for {
		err = processNextArrivedApproveOrTimeout()
		if err != nil {
			if err == ErrLoopBreak {
				return nil

			} else {
				return err
			}
		}
	}
}

func (p *Producer) validateReceivedBlockCandidateDigest(
	digest *block.Digest,
	tick *ticker.EventTimeFrameEnd,
	conf *external.Configuration) (err error) {

	// todo: fetch observers configuration that is related to the digest.ExternalChainHeight,
	//       current configuration might be irrelevant in case if digest was generated relatively long ago.
	//       It is OK for the beta release, but it must be fixed in the RC.

	if digest == nil || tick == nil || conf == nil {
		return errors.NilParameter
	}

	// Digest must be generated only by the observer,
	// that is granted to generate block at the moment.
	// todo: inform ticker about possible ticker offset error,
	//       try to sync clocks with majority of observers once more,
	//       but not more than once per some period of time
	//       (prevent malicious attack and ticker draining)
	if digest.AuthorObserverIndex != tick.Index {
		p.log().Debug(fmt.Sprint(
			"validateReceivedBlockCandidateDigest: "+
				"digest.AuthorObserverIndex != tick.Index, ", digest.AuthorObserverIndex, ", ", tick.Index))

		return errors.InvalidBlockCandidateDigest
	}

	// todo: implement logic
	//// The same observer might try to regenerate block in case if some other observer proposes changes,
	//// but only by increasing the attempt number.
	//if p.nextBlock != nil {
	//	if digest.Attempt <= p.nextBlock.Attempt {
	//		p.log().Debug(fmt.Sprint(
	//			"validateReceivedBlockCandidateDigest: "+
	//				"digest.Attempt > digest.Attempt, ", digest.Attempt, ", ", tick.Index))
	//
	//		return errors.InvalidBlockCandidateDigest
	//	}
	//}

	confHash := conf.Hash()
	if bytes.Compare(digest.ObserversConfHash.Bytes[:], confHash.Bytes[:]) != 0 {
		p.log().Debug(fmt.Sprint(
			"validateReceivedBlockCandidateDigest: " +
				"digest.ObserversConfHash != confHash, "))

		return errors.InvalidBlockCandidateDigest
	}

	// Body block height must be greater that current chain height.
	if digest.Index <= p.chain.Height()-1 {
		p.log().Debug(fmt.Sprint(
			"validateReceivedBlockCandidateDigest: "+
				"digest.Index <= p.chain.Index()-1, ", digest.Index, ", ", p.chain.Height()-1))

		return errors.InvalidBlockCandidateDigest
	}

	if digest.ExternalChainHeight > conf.CurrentExternalChainHeight() {
		p.log().Debug(fmt.Sprint(
			"validateReceivedBlockCandidateDigest: "+
				"digest.ExternalChainHeight > conf.CurrentExternalChainHeight(), ",
			digest.ExternalChainHeight, ", ", conf.CurrentExternalChainHeight()))

		return errors.InvalidBlockCandidateDigest
	}

	return
}

func (p *Producer) validateBlockCandidateDigestAndRelatedBlockCandidate(
	digest *block.Digest,
	candidate *block.Body) (err error) {

	if bytes.Compare(digest.BlockHash.Bytes[:], candidate.Hash.Bytes[:]) != 0 {
		return errors.InvalidBlockCandidateDigest
	}

	return
}

func (p *Producer) validateCandidateDigestSignatureResponse(
	response *responses.CandidateDigestApprove, conf *external.Configuration) (err error) {

	if p.nextBlock == nil {
		p.log().Debug(
			"validateCandidateDigestSignatureResponse: " +
				"There is no any pending proposed block")

		return errors.InvalidBlockCandidateDigestApprove
	}

	if len(conf.Observers) < int(response.ObserverIndex()+1) {
		p.log().Debug(
			"validateCandidateDigestSignatureResponse: " +
				"Current observers configuration has no observer with such index")

		return errors.InvalidBlockCandidateDigestApprove
	}

	remoteObserver := conf.Observers[response.ObserverIndex()]
	if !p.settings.Debug {
		// Prevent validation of it's own signatures,
		// except in debug mode.
		if p.settings.Observers.Network.Host == remoteObserver.Host &&
			p.settings.Observers.Network.Port == remoteObserver.Port {
			return errors.InvalidBlockCandidateDigestApprove
		}
	}

	isValid := p.keystore.CheckExternalSignature(
		p.nextBlock.Body.Hash, response.Signature, remoteObserver.PubKey)

	if !isValid {
		p.log().WithFields(log.Fields{
			"RemoteObserverIndex": response.ObserverIndex(),
			"PubKey":              remoteObserver.PubKey.X.String() + "; " + remoteObserver.PubKey.Y.String(),
			"Signature (S)":       response.Signature.S,
			"Signature (R)":       response.Signature.R,
			"BlockHash":           p.nextBlock.Body.Hash.Hex(),
		}).Debug(
			"validateCandidateDigestSignatureResponse: " +
				"received signature is not related to the generated block, or observer")

		return errors.InvalidBlockCandidateDigestApprove
	}

	return
}

func (p *Producer) validateBlockSignaturesRequest(
	request *requests.BlockSignaturesBroadcast, conf *external.Configuration) (err error) {

	if p.nextBlock == nil {
		// There is no any pending proposed block.
		return errors.InvalidBlockSignatures
	}

	if request.Signatures.IsMajorityApprovesCollected() == false {
		return errors.InvalidBlockSignatures
	}

	for i, sig := range request.Signatures.At {
		if sig == nil {
			continue
		}

		observer := conf.Observers[i]
		isValid := p.keystore.CheckExternalSignature(p.nextBlock.Body.Hash, *sig, observer.PubKey)
		if isValid == false {
			if p.settings.Debug {
				p.log().WithFields(log.Fields{
					"BlockHash":     p.nextBlock.Body.Hash.Hex(),
					"Signature (S)": sig.S,
					"Signature (R)": sig.R,
				}).Debug("validateBlockSignaturesRequest: signature check failed")
			}

			return errors.InvalidBlockSignatures
		}
	}

	return
}

func (p *Producer) finalStageTimeLeft(tick *ticker.EventTimeFrameEnd) time.Duration {
	return time.Now().Sub(tick.FinalStageTimestamp)
}

func (p *Producer) distributeCandidateDigests() (err error) {
	if p.nextBlock == nil {
		return errors.NotFound
	}

	digest, err := p.nextBlock.Body.GenerateDigest()
	if err != nil {
		return
	}

	select {
	case p.OutgoingRequestsCandidateDigestBroadcast <- requests.NewCandidateDigestBroadcast(digest):
	default:
		err = utils.Error("producer", "can't transfer CandidateDigestBroadcast to the core")
	}

	return
}

func (p *Producer) distributeCollectedSignatures() (err error) {
	if p.nextBlock == nil {
		return errors.NotFound
	}

	select {
	case p.OutgoingRequestsBlockSignaturesBroadcast <- requests.NewBlockSignaturesBroadcast(
		p.nextBlock.Signatures):
	default:
		err = utils.Error("producer", "can't transfer BlockSignaturesBroadcast to the core")
	}

	return
}

func (p *Producer) commitBlock(tick *ticker.EventTimeFrameEnd) (err error) {
	if p.nextBlock == nil {
		err = errors.AttemptToGenerateRedundantBlock
		p.log().Error(err.Error())
		return
	}

	dropTSLsFromPool := func() {
		hashes := make([]hash.SHA256Container, 0, len(p.nextBlock.Body.TSLs.At))
		for _, tsl := range p.nextBlock.Body.TSLs.At {
			data, err := tsl.MarshalBinary()
			if err != nil {
				return
			}

			hashes = append(hashes, hash.NewSHA256Container(data))
		}
		p.poolTSLs.DropInstances(hashes)
	}

	dropClaimsFromPool := func() {
		hashes := make([]hash.SHA256Container, 0, len(p.nextBlock.Body.Claims.At))
		for _, tsl := range p.nextBlock.Body.Claims.At {
			data, err := tsl.MarshalBinary()
			if err != nil {
				return
			}

			hashes = append(hashes, hash.NewSHA256Container(data))
		}
		p.poolTSLs.DropInstances(hashes)
	}

	err = p.chain.Append(p.nextBlock)
	committed := p.nextBlock
	if err != nil {
		return
	}

	dropTSLsFromPool()
	dropClaimsFromPool()

	p.log().WithFields(log.Fields{
		"Index":                      committed.Body.Index,
		"Hash":                       committed.Body.Hash.Hex(),
		"ClaimsCount":                committed.Body.Claims.Count(),
		"TSLsCount":                  committed.Body.TSLs.Count(),
		"SignaturesCount":            committed.Signatures.Count(),
		"SignaturesObserversIndexes": committed.Signatures.VotesIndexes(),
	}).Debug("Block committed")

	p.nextBlock = nil
	return
}

func (p *Producer) hasProposedBlock() bool {
	return p.nextBlock != nil
}

func (p *Producer) handleErrorIfAny(err error) {
	if err != nil {
		if p.settings.Debug {
			p.log().Debug(err)
		}
	}
}

func (p *Producer) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Producer"})
}
