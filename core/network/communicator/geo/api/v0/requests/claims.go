package requests

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/utils"
)

type ClaimAppend struct {
	*common.RequestWithoutResponse
	Claim *geo.Claim
}

func (request *ClaimAppend) MarshallBinary() (data []byte, err error) {
	typeID := []byte{common.ReqClaimAppend}
	claimBinary, err := request.Claim.MarshalBinary()
	return utils.ChainByteSlices(typeID, claimBinary), nil
}

func (request *ClaimAppend) UnmarshalBinary(data []byte) (err error) {
	request.Claim = geo.NewClaim()
	return request.Claim.UnmarshalBinary(data)
}

// --------------------------------------------------------------------------------------------------------------------

type ClaimIsPresent struct {
	*common.RequestWithResponse
	TxID *transactions.TxID
}

func NewClaimIsPresent(TxID *transactions.TxID) *ClaimIsPresent {
	return &ClaimIsPresent{
		RequestWithResponse: common.NewRequestWithResponse(),
		TxID:                TxID,
	}
}

func (request *ClaimIsPresent) MarshalBinary() (data []byte, err error) {
	typeID := []byte{common.ReqClaimIsPresent}
	txIDBinary, err := request.TxID.MarshalBinary()
	return utils.ChainByteSlices(typeID, txIDBinary), nil
}

func (request *ClaimIsPresent) UnmarshalBinary(data []byte) (err error) {
	request.RequestWithResponse = common.NewRequestWithResponse()
	request.TxID = transactions.NewTxID()
	return request.TxID.UnmarshalBinary(data)
}
