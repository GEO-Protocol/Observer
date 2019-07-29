package chain

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/settings"
)

func (p *Producer) processChainTopRequest(request *requests.ChainInfo) (e errors.E) {
	requestedBlock, err := p.chain.BlockAt(request.LastBlockIndex)
	if err != nil {
		// In case if block can't be fetched - then received block index is invalid.
		// This error is not related to the logic of current observer, and must be not reported as an error,
		// and the request must be simply ignored.

		if settings.Conf.Debug {
			p.log().Debug(
				"Received chain top request. " +
					"Current chain contains no block with such index. " +
					"Request ignored.")
		}
		return
	}

	lastBlock, e := p.chain.LastBlock()
	if e != nil {
		return
	}

	conf, err := p.reporter.GetCurrentConfiguration()
	if err != nil {
		// todo: conver error
		return errors.AppendStackTrace(err)
	}

	select {
	case p.OutgoingResponsesChainTop <- responses.NewChainTop(
		request, conf.CurrentObserverIndex, &lastBlock.Body.Hash, &requestedBlock.Body.Hash):

	default:
		e = errors.AppendStackTrace(errors.ChannelTransferringFailed)
	}

	return
}
