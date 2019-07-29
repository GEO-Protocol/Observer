package chain

import (
	"fmt"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	geoRequests "geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	geoResponses "geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	"time"
)

func (p *Producer) reportGEORequestError(r common.RequestWithResponse, err error) errors.E {
	select {
	case r.ErrorsChannel() <- err:
	default:
	}

	return errors.AppendStackTrace(err)
}

func (p *Producer) processGEOLastBlockHeightRequest(req *geoRequests.LastBlockNumber) (e errors.E) {
	req.ResponseChannel() <- &geoResponses.LastBlockHeight{
		Height: p.chain.Height(),
	}
	return
}

func (p *Producer) processGEOClaimIsPresentRequest(req *geoRequests.ClaimIsPresent) (e errors.E) {
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

func (p *Producer) processGEOTSLIsPresentRequest(req *geoRequests.TSLIsPresent) (e errors.E) {
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

func (p *Producer) processGEOTSLGetRequest(req *geoRequests.TSLGet) (e errors.E) {
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

func (p *Producer) processGEOTxStatesRequest(req *geoRequests.TxsStates) (e errors.E) {
	if req == nil || len(req.TxIDs.At) == 0 {
		// todo: replace error
		return errors.AppendStackTrace(errors.InvalidParameter)
	}
	response := geoResponses.NewTxStates(p.chain.Height())

	appendClaimInPoolState := func(TxID *transactions.TxID) (e errors.E) {
		resultsChannel, errorsChannel := p.poolClaims.ContainsInstance(TxID)
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
			return p.reportGEORequestError(req.RequestWithResponse, errors.TimeoutFired)
		}
	}

	appendClaimState := func(TxID *transactions.TxID) (e errors.E) {
		_, err := p.chain.GetClaim(TxID)
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

	appendTxState := func(TxID *transactions.TxID) (e errors.E) {
		_, err := p.chain.GetTSL(TxID)
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
		fmt.Println("------", TxID.Bytes)

		err := appendTxState(TxID)
		if err != nil {
			return p.reportGEORequestError(req.RequestWithResponse, err.Error())
		}
	}

	select {
	case req.ResponseChannel() <- response:
	default:
		return p.reportGEORequestError(req.RequestWithResponse, errors.ChannelTransferringFailed)
	}

	return
}
