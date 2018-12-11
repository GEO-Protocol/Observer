package geo

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/crypto/lamport"
	"geo-observers-blockchain/core/utils"
)

type TransactionSignaturesList struct {
	TxUUID     *types.TransactionUUID
	Signatures *lamport.Signatures
}

func NewTransactionSignaturesList() *TransactionSignaturesList {
	return &TransactionSignaturesList{
		TxUUID:     types.NewTransactionUUID(),
		Signatures: &lamport.Signatures{},
	}
}

func (t *TransactionSignaturesList) MarshalBinary() (data []byte, err error) {
	if t.TxUUID == nil || t.Signatures == nil {
		return nil, common.ErrNilInternalDataStructure
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
		minDataLength = types.TransactionUUIDSize + types.Uint16ByteSize
	)

	if len(data) < minDataLength {
		return common.ErrInvalidDataFormat
	}

	err = t.TxUUID.UnmarshalBinary(data[:types.TransactionUUIDSize])
	if err != nil {
		return
	}

	err = t.Signatures.UnmarshalBinary(data[types.TransactionUUIDSize:])
	if err != nil {
		return
	}

	return
}

//---------------------------------------------------------------------------------------------------------------------

const TransactionSignaturesListsMaxCount = common.GeoTransactionMaxParticipantsCount

type TransactionSignaturesLists struct {
	At []*TransactionSignaturesList
}

func (t *TransactionSignaturesLists) Add(tsl *TransactionSignaturesList) error {
	if tsl == nil {
		return common.ErrNilParameter
	}

	if t.Count() < TransactionSignaturesListsMaxCount {
		t.At = append(t.At, tsl)
		return nil
	}

	return common.ErrMaxCountReached
}

func (t *TransactionSignaturesLists) Count() uint16 {
	return uint16(len(t.At))
}

// Format:
// 2B - Total TSLs count.
// [2B, 2B, ... 2B] - TSLs sizes.
// [NB, NB, ... NB] - TSLs bodies.
func (t *TransactionSignaturesLists) MarshalBinary() (data []byte, err error) {

	// todo: add internal data size prediction and allocate memory at once.
	var (
		initialDataSize = types.Uint16ByteSize + // Total TSLs count.
			types.Uint32ByteSize*t.Count() // TSLs sizes fields.
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

		// Claims would be attached to the data after all TSLs size fields would be written.
		TSLs = append(TSLs, TSLBinary)
	}

	data = append(data, utils.ChainByteSlices(TSLs...)...)
	return
}

func (t *TransactionSignaturesLists) UnmarshalBinary(data []byte) (err error) {
	count, err := utils.UnmarshalUint16(data[:types.Uint16ByteSize])
	if err != nil {
		return
	}

	t.At = make([]*TransactionSignaturesList, count, count)
	if count == 0 {
		return
	}

	TSLsSizes := make([]uint32, 0, t.Count())

	var i uint16
	var offset uint32 = types.Uint16ByteSize
	for i = 0; i < count; i++ {
		TSLSize, err := utils.UnmarshalUint32(data[offset : offset+types.Uint32ByteSize])
		if err != nil {
			return err
		}
		if TSLSize == 0 {
			err = common.ErrInvalidDataFormat
		}

		TSLsSizes = append(TSLsSizes, TSLSize)
		offset += types.Uint32ByteSize
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
