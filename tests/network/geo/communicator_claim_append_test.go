package geo

import (
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0/requests"
	"testing"
)

const (
	AppendClaimRequestID = 128
)

func TestClaimAppendRequestID(t *testing.T) {
	if //noinspection GoBoolExpressions
	AppendClaimRequestID != common.ReqClaimAppend {
		t.Fatal()
	}
}

func TestClaimAppendPoolOnly(t *testing.T) {
	{
		// TSL with one signature.
		// (check in pool only).
		claim := createEmptyClaim(1)
		requestClaimAppend(t, claim)

		response := requestClaimIsPresent(t, claim.TxUUID)
		if !response.PresentInPool {
			t.Error()
		}
	}

	{
		// TSL with one signature.
		// (check in block only, observers cluster is needed).
		// todo: add implementation.
	}

	{
		// TSL with several signatures
	}
}

func TestAppendToChain(t *testing.T) {
	// todo: implement

	{
		// TSL with one signature.
	}

	{
		// TSL with several signatures
	}
}

func TestInvalidClaim(t *testing.T) {
	// todo: implement

	{
		// Invalid data: no signatures
	}

	{
		// Invalid data: invalid signatures
	}
}

func requestClaimAppend(t *testing.T, claim *geo.Claim) {
	conn := ConnectToObserver(t)
	defer conn.Close()

	request := requests.ClaimAppend{Claim: claim}
	requestBinary, err := request.MarshallBinary()
	if err != nil {
		t.Error()
	}

	SendData(t, conn, requestBinary)
}

func createEmptyClaim(membersCount int) (claim *geo.Claim) {
	txID, _ := transactions.NewRandomTransactionUUID()
	members := &transactions.Members{}

	for i := 0; i < membersCount; i++ {
		_ = members.Add(transactions.NewMember(uint16(i)))
	}

	claim = &geo.Claim{
		TxUUID:  txID,
		Members: members,
	}
	return
}
