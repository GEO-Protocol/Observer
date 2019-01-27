package block

import (
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
)

type Signed struct {
	Body       *Body
	Signatures *signatures.IndexedObserversSignatures
}

func (b *Signed) MarshalBinary() (data []byte, err error) {
	dataBinary, err := b.Body.MarshalBinary()
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

func (b *Signed) UnmarshalBinary(data []byte) (err error) {
	dataSize, err := utils.UnmarshalUint32(data[:common.Uint32ByteSize])
	if err != nil {
		return
	}

	offset := common.Uint32ByteSize
	b.Body = &Body{}
	err = b.Body.UnmarshalBinary(data[offset : offset+int(dataSize)])
	if err != nil {
		return
	}

	offset += int(dataSize)
	signaturesSize, err := utils.UnmarshalUint16(data[offset : offset+common.Uint16ByteSize])
	offset += common.Uint16ByteSize

	totalSize := int(dataSize) +
		int(common.Uint32ByteSize) +
		int(signaturesSize) +
		int(common.Uint16ByteSize)

	b.Signatures = signatures.NewIndexedObserversSignatures(settings.ObserversMaxCount)
	err = b.Signatures.UnmarshalBinary(data[offset:totalSize])
	return
}
