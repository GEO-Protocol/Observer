package transactions

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
)

type TransactionUUID struct {
	Bytes [common.TransactionUUIDSize]byte
}

func NewTransactionUUID() *TransactionUUID {
	return &TransactionUUID{}
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
