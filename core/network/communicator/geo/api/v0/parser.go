package v0

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
)

const (
	protocolHeaderBytesSize = 2
)

func ParseRequest(data []byte) (request requests.Request, e errors.E) {
	e = validateProtocolHeader(data)
	if e != nil {
		return
	}

	return dispatchRequest(data[1:])
}

func validateProtocolHeader(data []byte) (e errors.E) {
	// Request must contains at least protocol version and it's ID.
	if len(data) < protocolHeaderBytesSize {
		return errors.AppendStackTrace(errors.InvalidDataFormat)
	}

	if data[0] != ProtocolVersion {
		return errors.AppendStackTrace(errors.InvalidDataFormat)
	}

	return
}

func dispatchRequest(data []byte) (request requests.Request, e errors.E) {
	requestID := data[0]
	requestData := data[1:]

	switch requestID {
	case ReqChainLastBlockNumber:
		return parseRequest(&requests.LastBlockHeight{}, requestData)

	case ReqTSLAppend:
		return parseRequest(&requests.TSLAppend{}, requestData)

	case ReqTSLGet:
		return parseRequest(&requests.TSLGet{}, requestData)

	case ReqTSLIsPresent:
		return parseRequest(&requests.TSLIsPresent{}, requestData)

	case ReqClaimAppend:
		return parseRequest(&requests.ClaimAppend{}, requestData)

	case ReqClaimGet:
		return parseRequest(&requests.ClaimGet{}, requestData)

	case ReqClaimIsPresent:
		return parseRequest(&requests.ClaimIsPresent{}, requestData)

	default:
		return nil, errors.AppendStackTrace(errors.InvalidDataFormat)
	}
}

func parseRequest(request requests.Request, data []byte) (r requests.Request, e errors.E) {
	err := request.UnmarshalBinary(data)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}

	return request, nil
}
