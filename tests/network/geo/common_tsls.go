package geo

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/responses"
	"testing"
)

func RequestTSLAppend(t *testing.T, tsl *geo.TSL, observerIndex int) {
	conn := ConnectToObserver(t, observerIndex)
	defer conn.Close()

	request := requests.TSLAppend{TSL: tsl}
	requestBinary, err := request.MarshalBinary()
	if err != nil {
		t.Error()
	}

	SendData(t, conn, requestBinary)
}

func CreateEmptyTSL(membersCount int) (tsl *geo.TSL) {
	txID, _ := transactions.NewRandomTxID(1)
	members := &geo.TSLMembers{}

	for i := 0; i < membersCount; i++ {
		_ = members.Add(geo.NewTSLMember(uint16(i)))
	}

	tsl = &geo.TSL{
		TxUUID:  txID,
		Members: members,
	}
	return
}

func RequestTSLIsPresent(t *testing.T, TxID *transactions.TxID, observerIndex int) *responses.TSLIsPresent {
	conn := ConnectToObserver(t, observerIndex)
	defer conn.Close()

	request := requests.NewTSLIsPresent(TxID)
	SendRequest(t, request, conn)

	response := &responses.TSLIsPresent{}
	GetResponse(t, response, conn)
	return response
}
