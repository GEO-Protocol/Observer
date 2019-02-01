package transactions

import (
	"bytes"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
	"github.com/google/uuid"
)

const (
	// TxID is composed from Final Block Number (8B) and TxUUID (16B).
	TxIDBinarySize = 24
)

type TxID struct {
	Bytes [TxIDBinarySize]byte
}

func NewEmptyTxID() *TxID {
	return &TxID{}
}

func NewRandomTxID(blockNumber uint64) (txID *TxID, err error) {
	uuidBinary, err := uuid.New().MarshalBinary()
	if err != nil {
		return nil, err
	}

	blockNumberBinary := utils.MarshalUint64(blockNumber)

	txID = &TxID{}
	copy(txID.Bytes[0:8], blockNumberBinary)
	copy(txID.Bytes[8:TxIDBinarySize], uuidBinary)
	return
}

func (u *TxID) MarshalBinary() (data []byte, err error) {
	return u.Bytes[:TxIDBinarySize], nil
}

func (u *TxID) UnmarshalBinary(data []byte) error {
	if copy(u.Bytes[:], data[:TxIDBinarySize]) == TxIDBinarySize {
		return nil
	}

	return errors.InvalidDataFormat
}

func (u *TxID) Compare(other *TxID) bool {
	return bytes.Compare(u.Bytes[:], other.Bytes[:]) == 0
}

func (u *TxID) FinalBlockNumber() uint64 {
	blockNumber, _ := utils.UnmarshalUint64(u.Bytes[0:8])
	return blockNumber
}
