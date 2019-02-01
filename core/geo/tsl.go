package geo

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/utils"
)

var (
	TSLMinBinarySize = transactions.TxIDBinarySize + TSLMembersMinBinarySize
)

type TSL struct {
	TxUUID  *transactions.TxID
	Members *TSLMembers
}

func NewTSL() *TSL {
	return &TSL{
		TxUUID:  transactions.NewEmptyTxID(),
		Members: &TSLMembers{},
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
	err = t.TxUUID.UnmarshalBinary(data[:transactions.TxIDBinarySize])
	if err != nil {
		return
	}

	t.Members = &TSLMembers{}
	return t.Members.UnmarshalBinary(data[transactions.TxIDBinarySize:])
}

func (t *TSL) TxID() *transactions.TxID {
	return t.TxUUID
}
