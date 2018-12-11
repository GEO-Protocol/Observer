package chain

import (
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/utils"
)

type BlockSigned struct {
	Data *ProposedBlock

	Signatures *IndexedObserversSignatures

	// todo: add anchor to the external blockchain block number
}

func (b *BlockSigned) MarshalBinary() (data []byte, err error) {
	dataBinary, err := b.Data.MarshalBinary()
	if err != nil {
		return
	}

	signaturesBinary, err := b.Signatures.MarshalBinary()
	if err != nil {
		return
	}

	dataSizeBinary := utils.MarshalUint32(uint32(len(dataBinary)))
	signaturesSizeBinary := utils.MarshalUint16(uint16(len(signaturesBinary)))

	data = utils.ChainByteSlices(
		dataSizeBinary,
		dataBinary,
		signaturesSizeBinary,
		signaturesBinary)

	return
}

func (b *BlockSigned) UnmarshalBinary(data []byte) (err error) {

	dataSize, err := utils.UnmarshalUint32(data[:types.Uint32ByteSize])
	if err != nil {
		return
	}

	offset := types.Uint32ByteSize
	err = b.Data.UnmarshalBinary(data[offset : offset+int(dataSize)])
	if err != nil {
		return
	}

	offset += int(dataSize)
	signaturesSize, err := utils.UnmarshalUint16(data[offset : offset+types.Uint16ByteSize])
	offset += types.Uint16ByteSize

	totalSize := int(dataSize) +
		int(types.Uint32ByteSize) +
		int(signaturesSize) +
		int(types.Uint16ByteSize)

	err = b.Signatures.UnmarshalBinary(data[offset:totalSize])
	return
}
