package transactions

import "geo-observers-blockchain/core/common/errors"

const (
	TxStateNoInfo       = 0
	TxStateClaimInPool  = 1
	TxStateClaimInChain = 2
	TxStateTSLInChain   = 3
)

const (
	TxStateBinarySize = 1
)

type TxState struct {
	State uint8
}

func NewTxState(state uint8) (s *TxState, err error) {
	if state > TxStateTSLInChain {
		err = errors.InvalidParameter
		return
	}

	s = &TxState{State: state}
	return
}

func (state *TxState) MarshalBinary() (data []byte, err error) {
	data = make([]byte, 1, 1)
	data[0] = byte(state.State)
	return
}

func (state *TxState) UnmarshalBinary(data []byte) (err error) {
	if len(data) < TxStateBinarySize {
		err = errors.InvalidDataFormat
		return
	}

	state.State = data[0]
	return
}
