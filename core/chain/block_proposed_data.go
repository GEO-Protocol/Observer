package chain

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/utils"
)

type ProposedBlockData struct {
	// Sequence number of the proposed block.
	Height uint64

	// Sequence number of the observer, that has proposed the block.
	AuthorPosition uint16

	// Hashes of the records that should be included in the block.
	// To preserve traffic usage for observers communication,
	// only hashes of the records are transferred.
	ClaimsHashes hash.List
	TSLsHashes   hash.List

	// Hash of observers configuration.
	// Hash is needed as an anchor to the external conf. blockchain.
	ObserversConfHash hash.SHA256Container
}

func NewProposedBlockData(
	height uint64, authorPosition uint16, conf hash.SHA256Container,
	claims *geo.Claims, TSLs *geo.TransactionSignaturesLists) (blockData *ProposedBlockData, err error) {

	if height == 0 {
		err = errors.InvalidBlockHeight
		return
	}

	blockData = &ProposedBlockData{}
	blockData.Height = height
	blockData.AuthorPosition = authorPosition
	blockData.ObserversConfHash = conf

	for _, claim := range claims.At {
		data, err := claim.MarshalBinary()
		if err != nil {
			return nil, err
		}

		err = blockData.ClaimsHashes.Add(hash.NewSHA256Container(data))
		if err != nil {
			return nil, err
		}
	}

	for _, tsl := range TSLs.At {
		data, err := tsl.MarshalBinary()
		if err != nil {
			return nil, err
		}

		err = blockData.TSLsHashes.Add(hash.NewSHA256Container(data))
		if err != nil {
			return nil, err
		}
	}

	return
}

func (b *ProposedBlockData) MarshalBinary() (data []byte, err error) {

	observersConfHashData, err := b.ObserversConfHash.MarshalBinary()
	if err != nil {
		return
	}

	claimsData, err := b.ClaimsHashes.MarshalBinary()
	if err != nil {
		return
	}

	tslsData, err := b.TSLsHashes.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(
		// Static length structures.
		utils.MarshalUint64(b.Height),
		utils.MarshalUint16(b.AuthorPosition),
		observersConfHashData,

		// Variadic length structures.
		// Header.
		utils.MarshalUint16(uint16(len(claimsData))),
		// (Last fields size should not be included.
		// On data marshalling - last field size is bounded by the bytes array right border.)

		claimsData,
		tslsData)

	return
}

func (b *ProposedBlockData) UnmarshalBinary(data []byte) (err error) {
	const (
		offsetHeight                = 0
		offsetAuthorPosition        = offsetHeight + common.Uint64ByteSize
		offsetObserversConfHashData = offsetAuthorPosition + common.Uint16ByteSize
		fieldOffsetClaimsSize       = offsetObserversConfHashData + hash.BytesSize
		offsetVariadicLengthData    = fieldOffsetClaimsSize + common.Uint16ByteSize
	)

	b.Height, err = utils.UnmarshalUint64(data[offsetHeight:offsetAuthorPosition])
	if err != nil {
		return
	}

	b.AuthorPosition, err = utils.UnmarshalUint16(data[offsetAuthorPosition:offsetObserversConfHashData])
	if err != nil {
		return
	}

	err = b.ObserversConfHash.UnmarshalBinary(data[offsetObserversConfHashData:fieldOffsetClaimsSize])
	if err != nil {
		return
	}

	claimsDataSegmentSize, err := utils.UnmarshalUint16(data[fieldOffsetClaimsSize:offsetVariadicLengthData])
	if err != nil {
		return
	}

	claimsDataSegmentOffset := uint16(offsetVariadicLengthData)
	TSLsDataOffset := claimsDataSegmentOffset + claimsDataSegmentSize

	err = b.ClaimsHashes.UnmarshalBinary(data[claimsDataSegmentOffset:TSLsDataOffset])
	if err != nil {
		return
	}

	err = b.TSLsHashes.UnmarshalBinary(data[TSLsDataOffset:])
	if err != nil {
		return
	}

	return
}

func (b *ProposedBlockData) Hash() (h hash.SHA256Container, err error) {
	data, err := b.MarshalBinary()
	if err != nil {
		return
	}

	h = hash.NewSHA256Container(data)
	return
}
