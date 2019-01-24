package transactions

import (
	"bytes"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"github.com/google/uuid"
)

type TransactionUUID struct {
	Bytes [common.TransactionUUIDSize]byte
}

func NewTransactionUUID() *TransactionUUID {
	return &TransactionUUID{}
}

func NewRandomTransactionUUID() (TxID *TransactionUUID, err error) {
	uuidBinary, err := uuid.New().MarshalBinary()
	if err != nil {
		return nil, err
	}

	TxID = &TransactionUUID{}
	copy(TxID.Bytes[:], uuidBinary)
	return
}

func (u *TransactionUUID) MarshalBinary() (data []byte, err error) {
	return u.Bytes[:common.TransactionUUIDSize], nil
}

func (u *TransactionUUID) UnmarshalBinary(data []byte) error {
	if copy(u.Bytes[:], data[:common.TransactionUUIDSize]) == common.TransactionUUIDSize {
		return nil
	}

	return errors.InvalidDataFormat
}

func (u *TransactionUUID) Compare(other *TransactionUUID) bool {
	return bytes.Compare(u.Bytes[:], other.Bytes[:]) == 0
}
