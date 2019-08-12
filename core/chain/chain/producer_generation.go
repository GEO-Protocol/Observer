package chain

import (
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	"geo-observers-blockchain/core/utils"
	"github.com/deckarep/golang-set"
	"github.com/sirupsen/logrus"
	"time"
)

type generationFlowStates struct {
	observersIndexesApprovedDigest mapset.Set
	observersIndexesRejectedDigest mapset.Set
}

func defaultGenerationFlowStates() *generationFlowStates {
	return &generationFlowStates{
		observersIndexesApprovedDigest: mapset.NewSet(),
		observersIndexesRejectedDigest: mapset.NewSet(),
	}
}

func (p *Producer) processBlockGenerationFlow(tick *ticker.EventTimeFrameStarted) (e errors.E) {
	p.log().Trace("Block generation flow started")

	p.generationFlowStates = defaultGenerationFlowStates()

	candidate, e := p.generateBlockCandidateFromPool(tick.ObserversConfiguration)
	if e != nil {
		return
	}
	p.log().Trace("Candidate generated")

	p.nextBlock = &block.Signed{
		Body:       candidate,
		Signatures: signatures.NewIndexedObserversSignatures(settings.ObserversMaxCount),
	}

	// Signing the block
	sig, e := p.keystore.SignHash(p.nextBlock.Body.Hash)
	if e != nil {
		return e
	}
	p.log().Trace("block signed")

	p.nextBlock.Signatures.At[tick.ObserversConfiguration.CurrentObserverIndex] = sig

	e = p.distributeCandidateDigests()
	if e != nil {
		return
	}
	p.log().Trace("candidate distributed")

	return p.collectResponsesAndProcessConsensus(tick, tick.ObserversConfiguration)
}

func (p *Producer) generateBlockCandidateFromPool(
	conf *external.Configuration) (candidate *block.Body, e errors.E) {
	p.log().Trace("generateBlockCandidateFromPool")
	// todo: move to separate method
	getBlockReadyTSLs := func() (TSLs *geo.TSLs, e errors.E) {
		channel, errorsChannel := p.poolTSLs.BlockReadyInstances()

		TSLs = &geo.TSLs{}
		TSLs.At = make([]*geo.TSL, 0, 0) // todo: create constructor for it

		select {
		case i := <-channel:
			for _, instance := range i.At {
				TSLs.At = append(TSLs.At, instance.(*geo.TSL))
			}

		case _ = <-errorsChannel:
			e = errors.AppendStackTrace(errors.TSLsPoolReadFailed)

		case <-time.After(time.Second):
			e = errors.AppendStackTrace(errors.TSLsPoolReadFailed)
		}

		return
	}

	// todo: move to separate method
	getBlockReadyClaims := func() (claims *geo.Claims, e errors.E) {
		channel, errorsChannel := p.poolClaims.BlockReadyInstances()

		claims = &geo.Claims{}
		claims.At = make([]*geo.Claim, 0, 0)

		select {
		case i := <-channel:
			for _, instance := range i.At {
				claims.At = append(claims.At, instance.(*geo.Claim))
			}

		case _ = <-errorsChannel:
			e = errors.AppendStackTrace(errors.ClaimsPoolReadFailed)

		case <-time.After(time.Second):
			e = errors.AppendStackTrace(errors.ClaimsPoolReadFailed)
		}

		return
	}

	TSLs, e := getBlockReadyTSLs()
	if e != nil {
		return
	}

	claims, e := getBlockReadyClaims()
	if e != nil {
		return
	}

	return p.generateBlockCandidate(TSLs, claims, conf, conf.CurrentObserverIndex)
}

// todo: this method is very similar to the the original block generation
func (p *Producer) generateBlockCandidateFromDigest(
	digest *block.Digest, conf *external.Configuration) (candidate *block.Body, e errors.E) {

	// todo: move to separate method
	getBlockReadyTSLs := func() (tsls *geo.TSLs, e errors.E) {
		channel, errorsChannel := p.poolTSLs.BlockReadyInstancesByHashes(digest.TSLsHashes.At)

		tsls = &geo.TSLs{}
		select {
		case i := <-channel:
			for _, instance := range i.At {
				tsls.At = append(tsls.At, instance.(*geo.TSL))
			}

		case _ = <-errorsChannel:
			e = errors.AppendStackTrace(errors.TSLsPoolReadFailed)

		case <-time.After(time.Second):
			e = errors.AppendStackTrace(errors.TSLsPoolReadFailed)
		}

		return
	}

	// todo: move to separate method
	getBlockReadyClaims := func() (claims *geo.Claims, e errors.E) {
		channel, errorsChannel := p.poolClaims.BlockReadyInstancesByHashes(digest.ClaimsHashes.At)

		claims = &geo.Claims{}
		select {
		case i := <-channel:
			for _, instance := range i.At {
				claims.At = append(claims.At, instance.(*geo.Claim))
			}

		case _ = <-errorsChannel:
			e = errors.AppendStackTrace(errors.ClaimsPoolReadFailed)

		case <-time.After(time.Second):
			e = errors.AppendStackTrace(errors.ClaimsPoolReadFailed)
		}

		return
	}

	tsls, e := getBlockReadyTSLs()
	if e != nil {
		return
	}

	claims, e := getBlockReadyClaims()
	if e != nil {
		return
	}

	return p.generateBlockCandidate(tsls, claims, conf, digest.AuthorObserverIndex)
}

func (p *Producer) generateBlockCandidate(
	TLSs *geo.TSLs, claims *geo.Claims, conf *external.Configuration,
	authorObserverPosition uint16) (candidate *block.Body, e errors.E) {

	p.log().Trace("generateBlockCandidate")
	if p.hasProposedBlock() {
		e = errors.AppendStackTrace(errors.AttemptToGenerateRedundantBlock)
		return
	}

	nextIndex := p.chain.Height()
	previousBlock, err := p.chain.BlockAt(nextIndex - 1)
	if err != nil {
		return
	}

	candidate = &block.Body{
		Index:               nextIndex,
		ExternalChainHeight: conf.CurrentExternalChainHeight(),
		AuthorObserverIndex: authorObserverPosition,
		ObserversConfHash:   conf.Hash(),
		Claims:              claims,
		TSLs:                TLSs,
	}

	e = candidate.SortInternalSequences()
	if e != nil {
		return
	}

	e = candidate.UpdateHash(previousBlock.Body.Hash)
	if e != nil {
		return
	}

	if settings.OutputBlocksProducerDebug {
		p.log().WithFields(logrus.Fields{
			"Index":     candidate.Index,
			"BlockHash": candidate.Hash.Hex(),
		}).Debug("Candidate block generated")
	}

	return
}

func (p *Producer) approveBlockCandidateAndPropagateSignature(
	request *requests.CandidateDigestBroadcast, candidate *block.Body, conf *external.Configuration) (e errors.E) {

	// Prevent other block digests processing.
	// By default, there should no be any other digests,
	// but keep protocol strict this rule was added.
	p.validationFlowStates.BlockDigestApproved = true

	p.nextBlock = &block.Signed{
		Body:       candidate,
		Signatures: signatures.NewIndexedObserversSignatures(settings.ObserversMaxCount),
	}

	signature, e := p.keystore.SignHash(p.nextBlock.Body.Hash)
	if e != nil {
		return
	}

	if settings.OutputBlocksProducerDebug {
		p.log().WithFields(logrus.Fields{
			"BlockHash":  p.nextBlock.Body.Hash.Hex(),
			"PubKey (S)": signature.S,
			"PubKey (R)": signature.R,
		}).Debug("Block digest approved")
	}

	// todo: remember block hash, number and signature on the disk

	select {
	case p.OutgoingResponsesCandidateDigestApprove <- responses.NewCandidateDigestApprove(
		request, conf.CurrentObserverIndex, *signature):
	default:
		e = errors.AppendStackTrace(errors.ChannelTransferringFailed)
	}

	if settings.OutputBlocksProducerDebug {
		context := logrus.Fields{
			"Index":   request.Digest.Index,
			"Attempt": request.Digest.Attempt}
		p.log().WithFields(context).Debug("Signed candidate digest signed")
	}

	p.validationFlowStates.BlockDigestApproved = true
	return
}

func (p *Producer) collectResponsesAndProcessConsensus(
	tick *ticker.EventTimeFrameStarted, conf *external.Configuration) (e errors.E) {

	p.log().Trace("collectResponsesAndProcessConsensus")
	if p.nextBlock == nil {
		return errors.AppendStackTrace(errors.NotFound)
	}

	for {
		select {

		case responseApprove := <-p.IncomingResponsesCandidateDigestApprove:
			p.log().Trace("IncomingResponsesCandidateDigestApprove")
			e = p.processIncomingCandidateDigestApprove(responseApprove, conf)

		case responseReject := <-p.IncomingResponsesCandidateDigestReject:
			p.log().Trace("IncomingResponsesCandidateDigestReject")
			e = p.processIncomingCandidateDigestReject(responseReject, conf)

		//// Block digest mighte arrive aafter sync // todo: comment better
		//case reqBlockHashBroadcast := <-p.IncomingRequestsBlockHashBroadcast:
		//	err = p.processReceivedBlockHashBroadcast(reqBlockHashBroadcast)
		//	return

		//
		// Common requests processing goes here.
		// Please, search for #1 (anchor to find all other similar places).
		// Unfortunately, I know no way to inject in select several cases that are defined somewhere else.
		// todo: process case when configuration changed, to be able to interrupt flow and react ASAP

		case reqChainTop := <-p.IncomingRequestsChainTop:
			p.log().Trace("IncomingRequestsChainTop")
			e = p.processChainTopRequest(reqChainTop)

		case reqLastBlockHeight := <-p.GEORequestsLastBlockHeight:
			p.log().Trace("GEORequestsLastBlockHeight")
			e = p.processGEOLastBlockHeightRequest(reqLastBlockHeight)

		case reqClaimIsPresent := <-p.GEORequestsClaimIsPresent:
			p.log().Trace("GEORequestsClaimIsPresent")
			e = p.processGEOClaimIsPresentRequest(reqClaimIsPresent)

		case reqTSLIsPresent := <-p.GEORequestsTSLIsPresent:
			p.log().Trace("GEORequestsTSLIsPresent")
			e = p.processGEOTSLIsPresentRequest(reqTSLIsPresent)

		case reqTSLGet := <-p.GEORequestsTSLGet:
			p.log().Trace("GEORequestsTSLGet")
			e = p.processGEOTSLGetRequest(reqTSLGet)

		case reqTxStates := <-p.GEORequestsTxStates:
			p.log().Trace("GEORequestsTxStates")
			e = p.processGEOTxStatesRequest(reqTxStates)

		case <-time.After(p.blockGenerationStageTimeLeft(tick)):
			p.log().Trace("After")
			if p.nextBlock.Signatures.IsMajorityApprovesCollected() {
				p.log().Trace("processConsensusAchieved")
				return p.processConsensusAchieved(tick)

			} else {
				p.log().Trace("processNoConsensus")
				return p.processNoConsensus()
			}
		}

		if e != nil {
			return
		}
	}
}

func (p *Producer) blockGenerationStageTimeLeft(tick *ticker.EventTimeFrameStarted) (left time.Duration) {
	return tick.BlockGenerationStageEndTimestamp.Sub(time.Now())
}

func (p *Producer) distributeCandidateDigests() (e errors.E) {
	if p.nextBlock == nil {
		return errors.AppendStackTrace(errors.NotFound)
	}

	digest, e := p.nextBlock.Body.GenerateDigest()
	if e != nil {
		return
	}

	select {
	case p.OutgoingRequestsCandidateDigestBroadcast <- requests.NewCandidateDigestBroadcast(digest):
	default:
		return errors.AppendStackTrace(errors.ChannelTransferringFailed)
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

func (p *Producer) commitBlock(tick *ticker.EventTimeFrameStarted) (err error) {
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
		for _, claim := range p.nextBlock.Body.Claims.At {
			data, err := claim.MarshalBinary()
			if err != nil {
				return
			}

			hashes = append(hashes, hash.NewSHA256Container(data))
		}
		p.poolClaims.DropInstances(hashes)
	}

	err = p.chain.Append(p.nextBlock)
	committed := p.nextBlock
	if err != nil {
		return
	}

	dropTSLsFromPool()
	dropClaimsFromPool()

	p.log().WithFields(logrus.Fields{
		"Index":                      committed.Body.Index,
		"BlockHash":                  committed.Body.Hash.Hex(),
		"ClaimsCount":                committed.Body.Claims.Count(),
		"TSLsCount":                  committed.Body.TSLs.Count(),
		"SignaturesCount":            committed.Signatures.Count(),
		"SignaturesObserversIndexes": committed.Signatures.VotesIndexes(),
	}).Info("Block committed")

	// Generate event about next block generated.
	select {
	case p.OutgoingRequestsBlockHashBroadcast <- requests.NewBlockHashBroadcast(&p.nextBlock.Body.Hash):
	default:
		err = errors.ChannelTransferringFailed
		return
	}

	p.nextBlock = nil
	return
}

func (p *Producer) hasProposedBlock() bool {
	return p.nextBlock != nil
}

func (p *Producer) processConsensusAchieved(tick *ticker.EventTimeFrameStarted) (e errors.E) {
	err := p.distributeCollectedSignatures()
	if err != nil {
		// todo: convert error
		return errors.AppendStackTrace(err)
	}

	err = p.commitBlock(tick)
	if err != nil {
		// todo: convert error
		return errors.AppendStackTrace(err)
	}

	err = p.processFinalStage()
	if err != nil {
		// todo: convert error
		return errors.AppendStackTrace(err)
	}

	return
}

func (p *Producer) processNoConsensus() (e errors.E) {
	p.revealCollisionIfAny()

	// In case if next block was not generated - error should be reported to the core.
	// This case might be result of synchronisation failure of current observer.
	return errors.AppendStackTrace(errors.NoConsensus)
}

func (p *Producer) revealCollisionIfAny() (e errors.E) {
	digestsRejectsCollected := p.generationFlowStates.observersIndexesRejectedDigest.Cardinality()
	if digestsRejectsCollected < 2 {
		// In case if observer would try to sync right after first reject received -
		// there is a probability of very slow network consolidation.
		return
	}

	digestsApprovesCollected := p.generationFlowStates.observersIndexesApprovedDigest.Cardinality()
	if digestsRejectsCollected > digestsApprovesCollected {
		if settings.Conf.Debug {
			p.log().Debug("Rejects > Approves. Observer seems to be outdated. Sync requested.")
		}

		return errors.AppendStackTrace(errors.SyncNeeded)
	}

	return
}

func (p *Producer) notifyApproversAboutProbableCollisionDetected() {
	p.generationFlowStates.observersIndexesApprovedDigest.Each(func(i interface{}) bool {
		p.notifyObserverAboutProbableCollision(i.(uint16))
		return false
	})
}

func (p *Producer) notifyRejectorsAboutProbableCollisionDetected() {
	p.generationFlowStates.observersIndexesRejectedDigest.Each(func(i interface{}) bool {
		p.notifyObserverAboutProbableCollision(i.(uint16))
		return false
	})
}

func (p *Producer) notifyObserverAboutProbableCollision(observerIndex uint16) {
	notification := requests.NewCollisionNotification(observerIndex)

	select {
	case p.OutgoingRequestsCollisionNotifications <- notification:
	default:
		e := errors.AppendStackTrace(errors.ChannelTransferringFailed)

		p.log().WithFields(logrus.Fields{
			"ObserverIndex": observerIndex,
			"StackTrace":    e.StackTrace(),
		}).Error("Can't transfer collision notification request for further processing.")
	}
}
