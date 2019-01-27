package chain

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	geoRequests "geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	geoResponses "geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	"time"
)

func (p *Producer) reportGEORequestError(r *common.RequestWithResponse, err error) error {
	select {
	case r.ErrorsChannel() <- err:
	default:
	}

	return err
}

func (p *Producer) processGEOLastBlockHeightRequest(req *geoRequests.LastBlockNumber) (err error) {
	req.ResponseChannel() <- &geoResponses.LastBlockHeight{
		Height: p.chain.Height(),
	}
	return
}

func (p *Producer) processGEOClaimIsPresentRequest(req *geoRequests.ClaimIsPresent) (err error) {
	resultsChannel, errorsChannel := p.poolClaims.ContainsInstance(req.TxID)

	presentInPool := false
	select {
	case result := <-resultsChannel:
		presentInPool = result

	case err := <-errorsChannel:
		return p.reportGEORequestError(req.RequestWithResponse, err)

	case <-time.After(time.Second * 2):
		return p.reportGEORequestError(req.RequestWithResponse, errors.TimeoutFired)
	}

	blockNumber, err := p.chain.BlockWithClaim(req.TxID)
	if err != nil {
		return p.reportGEORequestError(req.RequestWithResponse, err)
	}

	response := &geoResponses.ClaimIsPresent{
		PresentInBlock: blockNumber,
		PresentInPool:  presentInPool,
	}
	req.ResponseChannel() <- response
	return
}

func (p *Producer) processGEOTSLIsPresentRequest(req *geoRequests.TSLIsPresent) (err error) {
	resultsChannel, errorsChannel := p.poolTSLs.ContainsInstance(req.TxID)

	presentInPool := false
	select {
	case result := <-resultsChannel:
		presentInPool = result

	case err := <-errorsChannel:
		return p.reportGEORequestError(req.RequestWithResponse, err)

	case <-time.After(time.Second * 2):
		return p.reportGEORequestError(req.RequestWithResponse, errors.TimeoutFired)
	}

	blockNumber, err := p.chain.BlockWithTSL(req.TxID)
	if err != nil {
		return p.reportGEORequestError(req.RequestWithResponse, err)
	}

	response := &geoResponses.TSLIsPresent{
		PresentInBlock: blockNumber,
		PresentInPool:  presentInPool,
	}
	req.ResponseChannel() <- response
	return
}

func (p *Producer) processGEOTSLGetRequest(req *geoRequests.TSLGet) (err error) {
	sendResponse := func(tsl *geo.TSL) {
		req.ResponseChannel() <- &geoResponses.TSLGet{
			TSL:       nil,
			IsPresent: false,
		}
	}

	sendNotFound := func() {
		req.ResponseChannel() <- &geoResponses.TSLGet{IsPresent: false}
	}

	tsl, err := p.chain.GetTSL(req.TxID)
	if err != nil {
		if err == errors.NotFound {
			sendNotFound()

		} else {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}
	}

	sendResponse(tsl)
	return
}

func (p *Producer) processGEOTxStatesRequest(req *geoRequests.TxsStates) (err error) {
	if req == nil || len(req.TxIDs.At) == 0 {
		return errors.InvalidParameter
	}
	response := geoResponses.NewTxStates()

	appendClaimInPoolState := func(TxID *transactions.TxID) (err error) {
		resultsChannel, errorsChannel := p.poolTSLs.ContainsInstance(TxID)
		select {
		case isPresent := <-resultsChannel:
			if isPresent {
				state, err := transactions.NewTxState(transactions.TxStateClaimInPool)
				if err != nil {
					return p.reportGEORequestError(req.RequestWithResponse, err)
				}

				err = response.States.Append(state)
				if err != nil {
					return p.reportGEORequestError(req.RequestWithResponse, err)
				}

			} else {
				state, err := transactions.NewTxState(transactions.TxStateNoInfo)
				if err != nil {
					return p.reportGEORequestError(req.RequestWithResponse, err)
				}

				err = response.States.Append(state)
				if err != nil {
					return p.reportGEORequestError(req.RequestWithResponse, err)
				}
			}

			return

		case err := <-errorsChannel:
			return p.reportGEORequestError(req.RequestWithResponse, err)

		case <-time.After(time.Second * 2):
			err = errors.TimeoutFired
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}
	}

	appendClaimState := func(TxID *transactions.TxID) (err error) {
		_, err = p.chain.GetClaim(TxID)
		if err == errors.NotFound {
			return appendClaimInPoolState(TxID)
		}
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}

		state, err := transactions.NewTxState(transactions.TxStateClaimInChain)
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}

		err = response.States.Append(state)
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}

		return
	}

	appendTxState := func(TxID *transactions.TxID) (err error) {
		_, err = p.chain.GetTSL(TxID)
		if err == errors.NotFound {
			return appendClaimState(TxID)
		}
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}

		state, err := transactions.NewTxState(transactions.TxStateTSLInChain)
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}

		err = response.States.Append(state)
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}

		return
	}

	for _, TxID := range req.TxIDs.At {
		err = appendTxState(TxID)
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err)
		}
	}

	select {
	case req.ResponseChannel() <- response:
	default:
		err = errors.ChannelTransferringFailed
		return p.reportGEORequestError(req.RequestWithResponse, err)
	}

	return
}
