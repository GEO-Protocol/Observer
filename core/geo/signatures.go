package geo

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/transactions"
	"geo-observers-blockchain/core/crypto/lamport"
	"geo-observers-blockchain/core/utils"
)

type TransactionSignaturesList struct {
	TxUUID     *transactions.TransactionUUID
	Signatures *lamport.Signatures
}

func NewTransactionSignaturesList() *TransactionSignaturesList {
	return &TransactionSignaturesList{
		TxUUID:     transactions.NewTransactionUUID(),
		Signatures: &lamport.Signatures{},
	}
}

func (t *TransactionSignaturesList) MarshalBinary() (data []byte, err error) {
	if t.TxUUID == nil || t.Signatures == nil {
		return nil, errors.NilInternalDataStructure
	}

	uuidData, err := t.TxUUID.MarshalBinary()
	if err != nil {
		return
	}

	signaturesData, err := t.Signatures.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(uuidData, signaturesData)
	return
}

func (t *TransactionSignaturesList) UnmarshalBinary(data []byte) (err error) {
	const (
		minDataLength = common.TransactionUUIDSize + common.Uint16ByteSize
	)

	if len(data) < minDataLength {
		return errors.InvalidDataFormat
	}

	t.TxUUID = &transactions.TransactionUUID{}
	err = t.TxUUID.UnmarshalBinary(data[:common.TransactionUUIDSize])
	if err != nil {
		return
	}

	t.Signatures = &lamport.Signatures{}
	err = t.Signatures.UnmarshalBinary(data[common.TransactionUUIDSize:])
	if err != nil {
		return
	}

	return
}

//---------------------------------------------------------------------------------------------------------------------

const TransactionSignaturesListsMaxCount = common.GEOTransactionMaxParticipantsCount

type TransactionSignaturesLists struct {
	At []*TransactionSignaturesList
}

func (t *TransactionSignaturesLists) Add(tsl *TransactionSignaturesList) error {
	if tsl == nil {
		return errors.NilParameter
	}

	if t.Count() < TransactionSignaturesListsMaxCount {
		t.At = append(t.At, tsl)
		return nil
	}

	return errors.MaxCountReached
}

func (t *TransactionSignaturesLists) Count() uint16 {
	return uint16(len(t.At))
}

// Format:
// 2B - Total TSLsHashes count.
// [2B, 2B, ... 2B] - TSLsHashes sizes.
// [NB, NB, ... NB] - TSLsHashes bodies.
func (t *TransactionSignaturesLists) MarshalBinary() (data []byte, err error) {

	// todo: add internal data size prediction and allocate memory at once.
	var (
		initialDataSize = common.Uint16ByteSize + // Total TSLsHashes count.
			common.Uint32ByteSize*t.Count() // TSLsHashes sizes fields.
	)

	data = make([]byte, 0, initialDataSize)
	data = append(data, utils.MarshalUint16(t.Count())...)
	TSLs := make([][]byte, 0, t.Count())

	for _, TSL := range t.At {
		TSLBinary, err := TSL.MarshalBinary()
		if err != nil {
			return nil, err
		}

		// Skip empty TSL, if any.
		if len(TSLBinary) == 0 {
			continue
		}

		// Append TSL size directly to the data stream.
		data = append(data, utils.MarshalUint32(uint32(len(TSLBinary)))...)

		// ClaimsHashes would be attached to the data after all TSLsHashes size fields would be written.
		TSLs = append(TSLs, TSLBinary)
	}

	data = append(data, utils.ChainByteSlices(TSLs...)...)
	return
}

func (t *TransactionSignaturesLists) UnmarshalBinary(data []byte) (err error) {
	count, err := utils.UnmarshalUint16(data[:common.Uint16ByteSize])
	if err != nil {
		return
	}

	t.At = make([]*TransactionSignaturesList, count, count)
	if count == 0 {
		return
	}

	TSLsSizes := make([]uint32, 0, t.Count())

	var i uint16
	var offset uint32 = common.Uint16ByteSize
	for i = 0; i < count; i++ {
		TSLSize, err := utils.UnmarshalUint32(data[offset : offset+common.Uint32ByteSize])
		if err != nil {
			return err
		}
		if TSLSize == 0 {
			err = errors.InvalidDataFormat
		}

		TSLsSizes = append(TSLsSizes, TSLSize)
		offset += common.Uint32ByteSize
	}

	for i = 0; i < count; i++ {
		TSLSize := TSLsSizes[i]

		TSL := NewTransactionSignaturesList()
		err := TSL.UnmarshalBinary(data[offset : offset+TSLSize])
		if err != nil {
			return err
		}

		t.At[i] = TSL
		offset += TSLSize
	}

	return
}
