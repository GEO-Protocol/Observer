package chain

import (
	"bytes"
	"fmt"
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/ticker"
	"github.com/sirupsen/logrus"
	"time"
)

// validationFlowStates describes
type validationFlowStates struct {

	// BlockDigestApproved prevents processing of several block digests requests during one validation round.
	// This variable should be set to true in case if block digest has passed validation.
	BlockDigestApproved bool
}

// defaultValidationFlowStates creates new states instance with default values for each variable.
// Usable for restoring to the defaults and as a constructor.
func defaultValidationFlowStates() *validationFlowStates {
	return &validationFlowStates{
		BlockDigestApproved: false,
	}
}

func (p *Producer) processValidationFlow(tick *ticker.EventTimeFrameStarted) (e errors.E) {
	p.validationFlowStates = defaultValidationFlowStates()

	for {
		if settings.Conf.Debug {
			p.log().Trace("Validation flow round started")
		}

		select {
		// todo: process case when configuration changed,
		//       to be able to interrupt flow and react ASAP

		case requestDigestBroadcast := <-p.IncomingRequestsCandidateDigest:
			e = p.processIncomingDigest(requestDigestBroadcast, tick)

		case requestBlockSignatures := <-p.IncomingRequestsBlockSignatures:
			e = p.processReceivedBlockSignatures(requestBlockSignatures, tick)

		case reqBlockHashBroadcast := <-p.IncomingRequestsBlockHashBroadcast:
			e = p.processReceivedBlockHashBroadcast(reqBlockHashBroadcast)

		//
		// Common requests processing goes here.
		// Please, search for #1 (anchor to find all other similar places).
		// Unfortunately, I know no way to inject in select several cases that are defined somewhere else.
		// todo: process case when configuration changed, to be able to interrupt flow and react ASAP

		case reqChainTop := <-p.IncomingRequestsChainTop:
			e = p.processChainTopRequest(reqChainTop)

		case reqLastBlockHeight := <-p.GEORequestsLastBlockHeight:
			e = p.processGEOLastBlockHeightRequest(reqLastBlockHeight)

		case reqClaimIsPresent := <-p.GEORequestsClaimIsPresent:
			e = p.processGEOClaimIsPresentRequest(reqClaimIsPresent)

		case reqTSLIsPresent := <-p.GEORequestsTSLIsPresent:
			e = p.processGEOTSLIsPresentRequest(reqTSLIsPresent)

		case reqTSLGet := <-p.GEORequestsTSLGet:
			e = p.processGEOTSLGetRequest(reqTSLGet)

		case reqTxStates := <-p.GEORequestsTxStates:
			e = p.processGEOTxStatesRequest(reqTxStates)

		case <-time.After(p.blockValidationStageTimeLeft(tick)):
			p.log().Trace("time after")
			// It is very suspicious, no block digest was received.
			if p.validationFlowStates.BlockDigestApproved == false {
				return errors.AppendStackTrace(errors.ValidationFailed)
			}

			return
		}

		if e != nil {
			p.logErrorIfAny(e)
			return errors.AppendStackTrace(errors.BlockValidationStageFailed)
		}
	}
}

func (p *Producer) processIncomingDigest(
	request *requests.CandidateDigestBroadcast, tick *ticker.EventTimeFrameStarted) (e errors.E) {

	//// todo: TEMP!!
	//p.reportDigestReject(request)
	//return

	// Validation errors should only be logged,
	// but not reported as errors to not break processing flow.
	var validationError errors.E

	validationError = p.validateReceivedBlockCandidateDigest(request.Digest, tick)
	p.logErrorIfAny(validationError)
	if validationError != nil {
		p.reportDigestReject(request)

		// Processing should stop here, but no error should be reported.
		return nil
	}

	candidate, e := p.generateBlockCandidateFromDigest(request.Digest, tick.ObserversConfiguration)
	if e != nil {
		return
	}

	validationError = p.validateBlockCandidateDigestAndRelatedBlockCandidate(request.Digest, candidate)
	p.logErrorIfAny(validationError)
	if validationError != nil {
		p.reportDigestReject(request)

		// Processing should stop here, but no error should be reported.
		return nil
	}

	return p.approveBlockCandidateAndPropagateSignature(request, candidate, tick.ObserversConfiguration)
}

func (p *Producer) processIncomingCandidateDigestApprove(
	response *responses.CandidateDigestApprove, conf *external.Configuration) (e errors.E) {

	if settings.Conf.Debug {
		p.log().Debug("processIncomingCandidateDigestApprove: received approve response")
	}

	if response == nil || conf == nil {
		return errors.AppendStackTrace(errors.NilParameter)
	}

	validationError := p.validateCandidateDigestApprove(response, conf)
	if validationError != nil {
		if settings.Conf.Debug {
			p.logErrorIfAny(validationError)
		}

		// In case if approve doesn't pass validation -
		// it should not be interpreted as reject!
		// Otherwise any malicious packet, that would be composed as block approve,
		// would be able to reach internal producer logic.
		//
		// In this case the response should be silently ignored.
		return nil
	}

	// Append received signature to the signatures list.
	p.nextBlock.Signatures.At[response.ObserverIndex()] = &response.Signature
	p.generationFlowStates.observersIndexesApprovedDigest.Add(response.ObserverIndex())
	return
}

func (p *Producer) processIncomingCandidateDigestReject(
	response *responses.CandidateDigestReject, conf *external.Configuration) (e errors.E) {

	p.log().Debug("processIncomingCandidateDigestReject: received reject response")

	validationError := p.validateCandidateDigestReject(response, conf)
	if validationError != nil {
		if settings.Conf.Debug {
			p.logErrorIfAny(validationError)
		}

		// In case if response doesn't pass validation -
		// it should not be interpreted as reject!
		// Otherwise any malicious packet, that would be composed as block reject,
		// would be able to reach internal producer logic.
		//
		// In this case the response should be silently ignored.
		p.log().Debug("processIncomingCandidateDigestReject: received response rejected as invalid")
		return nil

	} else {
		p.log().Debug("processIncomingCandidateDigestReject: received response taken into account")
		p.generationFlowStates.observersIndexesRejectedDigest.Add(response.SenderIndex())

		// In case if bigger part of network reports invalid digest -
		// current observer must be resynchronized, because it is obviously outdated.
		rejectsCount := p.generationFlowStates.observersIndexesRejectedDigest.Cardinality()
		minimalReportersCount := settings.ObserversMaxCount/2 + 1

		if rejectsCount >= minimalReportersCount {
			p.log().Debug("Observer seems to be outdated. Sync requested.")
			return errors.AppendStackTrace(errors.SyncNeeded)
		}
	}

	return
}

func (p *Producer) processReceivedBlockSignatures(
	request *requests.BlockSignaturesBroadcast, tick *ticker.EventTimeFrameStarted) (e errors.E) {

	e = p.validateBlockSignaturesRequest(request, tick.ObserversConfiguration)
	if e != nil {
		return
	}

	p.nextBlock.Signatures = request.Signatures
	err := p.commitBlock(tick)
	if err != nil {
		return errors.AppendStackTrace(err)
	}

	return
}

func (p *Producer) validateReceivedBlockCandidateDigest(
	digest *block.Digest, tick *ticker.EventTimeFrameStarted) (e errors.E) {

	// todo: fetch observers configuration that is related to the digest.ExternalChainHeight,
	//       current configuration might be irrelevant in case if digest was generated relatively long ago.
	//       It is OK for the beta release, but it must be fixed in the RC.

	if digest == nil || tick == nil || tick.ObserversConfiguration == nil {
		return errors.AppendStackTrace(errors.NilParameter)
	}

	if p.validationFlowStates.BlockDigestApproved {
		// Some other block digest is already approved.
		// No any other block digests should be approved any more.
		e = errors.AppendStackTrace(errors.InvalidBlockCandidateDigest)
		return
	}

	// Digest must be generated only by the observer,
	// that is granted to generate block at the moment.
	// todo: inform ticker about possible ticker offset error,
	//       try to sync clocks with majority of observers once more,
	//       but not more than once per some period of time
	//       (prevent malicious attack and ticker draining)
	if digest.AuthorObserverIndex != tick.Index {
		if settings.OutputBlocksProducerDebug {
			p.log().Debug(fmt.Sprint(
				"validateReceivedBlockCandidateDigest: "+
					"digest.AuthorObserverIndex != tick.Index, ", digest.AuthorObserverIndex, ", ", tick.Index))
		}

		return errors.AppendStackTrace(errors.InvalidTimeFrame)
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

	confHash := tick.ObserversConfiguration.Hash()
	if bytes.Compare(digest.ObserversConfHash.Bytes[:], confHash.Bytes[:]) != 0 {
		if settings.OutputBlocksProducerDebug {
			p.log().Debug(fmt.Sprint(
				"validateReceivedBlockCandidateDigest: " +
					"digest.ObserversConfHash != confHash, "))
		}

		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigest)
	}

	// Body block height must be greater that current chain height.
	if digest.Index <= p.chain.Height()-1 {
		if settings.OutputBlocksProducerDebug {
			p.log().Debug(fmt.Sprint(
				"validateReceivedBlockCandidateDigest: "+
					"digest.Index <= p.chain.Index()-1, ", digest.Index, ", ", p.chain.Height()-1))
		}

		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigest)
	}

	if digest.ExternalChainHeight > tick.ObserversConfiguration.CurrentExternalChainHeight() {
		if settings.OutputBlocksProducerDebug {
			p.log().Debug(fmt.Sprint(
				"validateReceivedBlockCandidateDigest: "+
					"digest.ExternalChainHeight > conf.CurrentExternalChainHeight(), ",
				digest.ExternalChainHeight, ", ", tick.ObserversConfiguration.CurrentExternalChainHeight()))
		}

		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigest)
	}

	return
}

func (p *Producer) validateBlockCandidateDigestAndRelatedBlockCandidate(
	digest *block.Digest, candidate *block.Body) (e errors.E) {

	if bytes.Compare(digest.BlockHash.Bytes[:], candidate.Hash.Bytes[:]) != 0 {
		//if settings.OutputBlocksProducerDebug {
		//	p.log().Debug(
		//		"validateBlockCandidateDigestAndRelatedBlockCandidate: " +
		//			"internal generated digest doesn't corresponds with received candidate")
		//}

		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigest)
	}

	return
}

func (p *Producer) validateCandidateDigestApprove(
	response *responses.CandidateDigestApprove, conf *external.Configuration) (e errors.E) {

	if p.nextBlock == nil {
		if settings.OutputBlocksProducerDebug {
			p.log().Debug(
				"validateCandidateDigestApprove: " +
					"There is no any pending proposed block")
		}

		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestApprove)
	}

	if len(conf.Observers) < int(response.ObserverIndex()+1) {
		if settings.OutputBlocksProducerDebug {
			p.log().Debug(
				"validateCandidateDigestApprove: " +
					"Current observers configuration has no observer with such index")
		}

		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestApprove)
	}

	remoteObserver := conf.Observers[response.ObserverIndex()]
	if !settings.Conf.Debug {
		// Prevent validation of it's own signatures,
		// except in debug mode.
		if settings.Conf.Observers.Network.Host == remoteObserver.Host &&
			settings.Conf.Observers.Network.Port == remoteObserver.Port {
			return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestApprove)
		}
	}

	isValid := p.keystore.CheckExternalSignature(
		p.nextBlock.Body.Hash, response.Signature, remoteObserver.PubKey)

	if !isValid {
		if settings.OutputBlocksProducerDebug {
			p.log().WithFields(logrus.Fields{
				"RemoteObserverIndex": response.ObserverIndex(),
				"PubKey":              remoteObserver.PubKey.X.String() + "; " + remoteObserver.PubKey.Y.String(),
				"PubKey (S)":          response.Signature.S,
				"PubKey (R)":          response.Signature.R,
				"BlockHash":           p.nextBlock.Body.Hash.Hex(),
			}).Debug(
				"validateCandidateDigestApprove: " +
					"received signature is not related to the generated block, or observer")
		}

		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestApprove)
	}

	p.log().WithFields(logrus.Fields{
		"RemoteObserverIndex": response.ObserverIndex(),
		"PubKey":              remoteObserver.PubKey.X.String() + "; " + remoteObserver.PubKey.Y.String(),
		"PubKey (S)":          response.Signature.S,
		"PubKey (R)":          response.Signature.R,
		"BlockHash":           p.nextBlock.Body.Hash.Hex(),
	}).Debug(
		"validateCandidateDigestApprove: signature is valid")
	return
}

func (p *Producer) validateCandidateDigestReject(
	response *responses.CandidateDigestReject, conf *external.Configuration) (e errors.E) {

	if p.nextBlock == nil {
		// No block candidate is present.
		// No reject is expected.
		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestReject)
	}

	if response.Hash.Compare(&p.nextBlock.Body.Hash) == false {
		// Received block reject is related to other block candidate, than current one.
		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestReject)
	}

	if len(conf.Observers) < int(response.ObserverIndex()+1) {
		// Current configuration have no observer with such index.
		return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestReject)
	}

	// ToDo: Current implementation of response.ObserverIndex() is broken.
	//		 Return value of this method is always equal to current observer index.
	// 		 When it would be fixed - next commented rule should be enable back.
	//
	//if conf.CurrentObserverIndex == response.ObserverIndex() {
	//	// Producer must not generate approve/reject responses.
	//	return errors.AppendStackTrace(errors.InvalidBlockCandidateDigestReject)
	//}

	return nil
}

func (p *Producer) validateBlockSignaturesRequest(
	request *requests.BlockSignaturesBroadcast, conf *external.Configuration) (e errors.E) {

	if p.nextBlock == nil {
		// There is no any pending proposed block.
		return errors.AppendStackTrace(errors.InvalidBlockSignatures)
	}

	if request.Signatures.IsMajorityApprovesCollected() == false {
		return errors.AppendStackTrace(errors.InvalidBlockSignatures)
	}

	for i, sig := range request.Signatures.At {
		if sig == nil {
			continue
		}

		observer := conf.Observers[i]
		isValid := p.keystore.CheckExternalSignature(p.nextBlock.Body.Hash, *sig, observer.PubKey)
		if isValid == false {
			if settings.OutputBlocksProducerDebug {
				p.log().WithFields(logrus.Fields{
					"BlockHash":  p.nextBlock.Body.Hash.Hex(),
					"PubKey (S)": sig.S,
					"PubKey (R)": sig.R,
				}).Error("validateBlockSignaturesRequest: signature check failed")
			}

			return errors.AppendStackTrace(errors.InvalidBlockSignatures)
		}
	}

	return
}

func (p *Producer) blockValidationStageTimeLeft(tick *ticker.EventTimeFrameStarted) (left time.Duration) {
	return tick.ValidationStageEndTimestamp.Sub(time.Now())
}

func (p *Producer) processReceivedBlockHashBroadcast(request *requests.BlockHashBroadcast) (e errors.E) {
	lastBlock, e := p.chain.LastBlock()
	if e != nil {
		return
	}

	if !lastBlock.Body.Hash.Compare(request.Hash) {
		return errors.AppendStackTrace(errors.Collision)
	}

	return
}

// reportDigestReject schedules sending of the response about block digest rejecting.
// The response would be sent to the observer, which has sent the request.
// Returns error in case if response can't be sent through the corresponding outgoing responses channel.
func (p *Producer) reportDigestReject(broadcastRequest *requests.CandidateDigestBroadcast) (e errors.E) {
	response := responses.NewCandidateDigestReject(broadcastRequest)

	select {
	case p.OutgoingResponsesCandidateDigestReject <- response:
		if settings.Conf.Debug {
			p.log().WithFields(logrus.Fields{
				"BlockHash":     broadcastRequest.Digest.BlockHash.Hex(),
				"ObserverIndex": broadcastRequest.ObserverIndex(),
			}).Debug("Block digest has been rejected")
		}

	default:
		e = errors.AppendStackTrace(errors.ChannelTransferringFailed)
	}

	return
}
