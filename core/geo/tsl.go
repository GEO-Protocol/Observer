package geo

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/utils"
)

const (
	TSLMinBinarySize = common.TransactionUUIDSize + transactions.MembersMinBinarySize
)

type TSL struct {
	TxUUID  *transactions.TxID
	Members *transactions.Members
}

func NewTSL() *TSL {
	return &TSL{
		TxUUID:  transactions.NewTxID(),
		Members: &transactions.Members{},
	}
}

func (t *TSL) MarshalBinary() (data []byte, err error) {
	if t.TxUUID == nil || t.Members == nil {
		return nil, errors.NilInternalDataStructure
	}

	uuidBinary, err := t.TxUUID.MarshalBinary()
	if err != nil {
		return
	}

	membersBinary, err := t.Members.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(uuidBinary, membersBinary)
	return
}

func (t *TSL) UnmarshalBinary(data []byte) (err error) {
	if len(data) < TSLMinBinarySize {
		return errors.InvalidDataFormat
	}

	t.TxUUID = &transactions.TxID{}
	err = t.TxUUID.UnmarshalBinary(data[:common.TransactionUUIDSize])
	if err != nil {
		return
	}

	t.Members = &transactions.Members{}
	return t.Members.UnmarshalBinary(data[common.TransactionUUIDSize:])
}

func (t *TSL) TxID() *transactions.TxID {
	return t.TxUUID
}
