package transactions

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
)

const (
	TxStatesMinBinarySize = common.Uint16ByteSize + 1
	TxStatesMaxCount      = 1024 * 10
)

type TxStates struct {
	At []*TxState
}

func NewTxStates(states []*TxState) *TxStates {
	return &TxStates{At: states}
}

func (states *TxStates) Append(state *TxState) (err error) {
	if len(states.At) >= TxStatesMaxCount {
		err = errors.MaxCountReached
		return
	}

	states.At = append(states.At, state)
	return
}

func (states *TxStates) MarshalBinary() (data []byte, err error) {
	totalStatesCount := uint16(len(states.At))
	totalStatesCountBinary := utils.MarshalUint16(totalStatesCount)

	data = make([]byte, 0, common.Uint16ByteSize+(totalStatesCount*TxIDBinarySize))
	data = utils.ChainByteSlices(data, totalStatesCountBinary)

	for _, state := range states.At {
		idBinary, err := state.MarshalBinary()
		if err != nil {
			return nil, err
		}

		data = utils.ChainByteSlices(data, idBinary)
	}

	return
}

func (states *TxStates) UnmarshalBinary(data []byte) (err error) {
	if len(data) < TxStatesMinBinarySize {
		err = errors.InvalidDataFormat
		return
	}

	totalStatesCount, err := utils.UnmarshalUint16(data[:common.Uint16ByteSize])
	if err != nil {
		return
	}

	states.At = make([]*TxState, totalStatesCount, totalStatesCount)
	for i := 0; i < int(totalStatesCount); i++ {
		state := &TxState{}
		err = state.UnmarshalBinary(data[common.Uint16ByteSize+TxStateBinarySize*i:])
		if err != nil {
			return
		}

		states.At[i] = state
	}
	return
}
