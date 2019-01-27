package transactions

import (
	"bytes"
	"geo-observers-blockchain/core/common/errors"
	"github.com/google/uuid"
)

const (
	TxIDBinarySize = 16
)

type TxID struct {
	Bytes [TxIDBinarySize]byte
}

func NewTxID() *TxID {
	return &TxID{}
}

func NewRandomTransactionUUID() (txID *TxID, err error) {
	uuidBinary, err := uuid.New().MarshalBinary()
	if err != nil {
		return nil, err
	}

	txID = &TxID{}
	copy(txID.Bytes[:], uuidBinary)
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
