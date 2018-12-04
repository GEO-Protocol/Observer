package types

const (
	TransactionUUIDSize = 16

	Uint16ByteSize = 2
	Uint32ByteSize = 4
	Uint64ByteSize = 8
)

type TransactionUUID struct {
	Bytes [TransactionUUIDSize]byte
}

func NewTransactionUUID() *TransactionUUID {
	return &TransactionUUID{}
}

func (u *TransactionUUID) MarshalBinary() (data []byte, err error) {
	return u.Bytes[:TransactionUUIDSize], nil
}

func (u *TransactionUUID) UnmarshalBinary(data []byte) error {
	if copy(u.Bytes[:], data[:TransactionUUIDSize]) == TransactionUUIDSize {
		return nil

	} else {
		return ErrorInvalidCopyOperation

	}
}
