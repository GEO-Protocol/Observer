package block

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
)

// Digest represents list of hashes of the records,
// that are proposed to be included into the next block.
type Digest struct {
	Attempt             uint16
	Index               uint64
	ExternalChainHeight uint64
	AuthorObserverIndex uint16
	ObserversConfHash   hash.SHA256Container

	// Hashes of the records that should be included into the block.
	// To preserve redundant traffic usage by observers communication,
	// only hashes of the records are transferred.
	ClaimsHashes hash.List
	TSLsHashes   hash.List

	// todo: replace by the BLAKE2b
	BlockHash hash.SHA256Container

	// Note: author's signature might bo not included.
	//       Request with block candidate would be signed,
	//       no need to sign the same data twice.
}

// todo: tests needed
func (c *Digest) MarshalBinary() (data []byte, err error) {
	if c.AuthorObserverIndex >= uint16(settings.ObserversMaxCount) {
		err = errors.ExpectationFailed
		return
	}

	claimsData, err := c.ClaimsHashes.MarshalBinary()
	if err != nil {
		return
	}

	tslsData, err := c.TSLsHashes.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(
		utils.MarshalUint16(c.Attempt),
		utils.MarshalUint64(c.Index),
		utils.MarshalUint64(c.ExternalChainHeight),
		utils.MarshalUint16(c.AuthorObserverIndex),
		c.ObserversConfHash.Bytes[:],
		c.BlockHash.Bytes[:],

		utils.MarshalUint16(uint16(len(claimsData))),
		claimsData,
		tslsData)

	return
}

// todo: tests needed
// todo: add check for min data length
func (c *Digest) UnmarshalBinary(data []byte) (err error) {
	const (
		offsetAttempt             = 0
		offsetHeight              = offsetAttempt + common.Uint16ByteSize
		offsetExternalChainHeight = offsetHeight + common.Uint64ByteSize
		offsetAuthorPosition      = offsetExternalChainHeight + common.Uint64ByteSize
		offsetObserversConfHash   = offsetAuthorPosition + common.Uint16ByteSize
		offsetBlockHash           = offsetObserversConfHash + hash.BytesSize
		offsetClaimsSize          = offsetBlockHash + hash.BytesSize
		offsetVariadicLengthData  = offsetClaimsSize + common.Uint16ByteSize
	)

	c.Attempt, err = utils.UnmarshalUint16(data[offsetAttempt:offsetHeight])
	if err != nil {
		return
	}

	c.Index, err = utils.UnmarshalUint64(data[offsetHeight:offsetExternalChainHeight])
	if err != nil {
		return
	}

	c.ExternalChainHeight, err = utils.UnmarshalUint64(data[offsetExternalChainHeight:offsetAuthorPosition])
	if err != nil {
		return
	}

	c.AuthorObserverIndex, err = utils.UnmarshalUint16(data[offsetAuthorPosition:offsetObserversConfHash])
	if err != nil {
		return
	}

	err = c.ObserversConfHash.UnmarshalBinary(data[offsetObserversConfHash:offsetBlockHash])
	if err != nil {
		return
	}

	err = c.BlockHash.UnmarshalBinary(data[offsetBlockHash:offsetClaimsSize])
	if err != nil {
		return
	}

	claimsDataSegmentSize, err := utils.UnmarshalUint16(data[offsetClaimsSize:offsetVariadicLengthData])
	if err != nil {
		return
	}

	claimsDataSegmentOffset := uint16(offsetVariadicLengthData)
	TSLsDataOffset := claimsDataSegmentOffset + claimsDataSegmentSize

	err = c.ClaimsHashes.UnmarshalBinary(data[claimsDataSegmentOffset:TSLsDataOffset])
	if err != nil {
		return
	}

	err = c.TSLsHashes.UnmarshalBinary(data[TSLsDataOffset:])
	if err != nil {
		return
	}

	return
}
