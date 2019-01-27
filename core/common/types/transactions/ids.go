package transactions

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
)

const (
	TransactionIDsMinBinarySize = common.Uint16ByteSize + TxIDBinarySize
)

type TransactionIDs struct {
	At []*TxID
}

func NewTransactionIDs(ids []*TxID) *TransactionIDs {
	return &TransactionIDs{
		At: ids,
	}
}

func (ids *TransactionIDs) MarshalBinary() (data []byte, err error) {
	totalIDsCount := uint16(len(ids.At))
	totalIDsCountBinary := utils.MarshalUint16(totalIDsCount)
	data = make([]byte, 0, common.Uint16ByteSize+(totalIDsCount*TxIDBinarySize))
	data = utils.ChainByteSlices(data, totalIDsCountBinary)

	for _, id := range ids.At {
		idBinary, err := id.MarshalBinary()
		if err != nil {
			return nil, err
		}

		data = utils.ChainByteSlices(data, idBinary)
	}

	return
}

func (ids *TransactionIDs) UnmarshalBinary(data []byte) (err error) {
	if len(data) < TransactionIDsMinBinarySize {
		err = errors.InvalidDataFormat
		return
	}

	totalIDsCount, err := utils.UnmarshalUint16(data[:common.Uint16ByteSize])
	if err != nil {
		return
	}

	ids.At = make([]*TxID, totalIDsCount, totalIDsCount)
	if totalIDsCount == 0 {
		return
	}

	for i := 0; i < int(totalIDsCount); i++ {
		id := &TxID{}
		err = id.UnmarshalBinary(data[common.Uint16ByteSize+i*TxIDBinarySize:])
		if err != nil {
			return
		}

		ids.At[i] = id
	}

	return
}
