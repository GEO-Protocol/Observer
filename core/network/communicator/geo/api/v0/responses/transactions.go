package responses

import "geo-observers-blockchain/core/common/types/transactions"

type TxStates struct {
	States *transactions.TxStates
}

func NewTxStates() *TxStates {
	return &TxStates{
		States: transactions.NewTxStates([]*transactions.TxState{})}
}

func (states *TxStates) MarshalBinary() ([]byte, error) {
	return states.States.MarshalBinary()
}

func (states *TxStates) UnmarshalBinary(data []byte) (err error) {
	states.States = &transactions.TxStates{}
	return states.States.UnmarshalBinary(data)
}
