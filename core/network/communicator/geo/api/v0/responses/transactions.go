package responses

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/utils"
)

type TxStates struct {
	States             *transactions.TxStates
	CurrentBlockNumber uint64
}

func NewTxStates(blockNumber uint64) *TxStates {
	return &TxStates{
		States:             transactions.NewTxStates([]*transactions.TxState{}),
		CurrentBlockNumber: blockNumber,
	}
}

func (states *TxStates) MarshalBinary() (data []byte, err error) {
	blockNumberBinary := utils.MarshalUint64(states.CurrentBlockNumber)
	statesBinary, err := states.States.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(blockNumberBinary, statesBinary)
	return
}

func (states *TxStates) UnmarshalBinary(data []byte) (err error) {
	states.CurrentBlockNumber, err = utils.UnmarshalUint64(data[:common.Uint64ByteSize])

	states.States = &transactions.TxStates{}
	err = states.States.UnmarshalBinary(data[:common.Uint64ByteSize])
	if err != nil {
		return
	}

	return
}
