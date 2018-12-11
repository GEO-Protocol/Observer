package chain

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/crypto/ecdsa"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/utils"
)

type ProposedBlock struct {
	// Sequence number of the proposed block.
	Height uint64

	// Hash of observers configuration.
	// Hash is needed as an anchor to the external conf. blockchain.
	ObserversConfHash types.SHA256Container

	// Sequence number of the observers, that has proposed the block.
	AuthorPosition uint16

	Claims geo.Claims
	TSLs   geo.TransactionSignaturesLists

	// Hash of the entire block.
	Hash types.SHA256Container

	// Signature of the block hash.
	// By default - empty.
	AuthorSignature *ecdsa.Signature
}

func NewBlockProposal(
	height uint64, observersConfHash types.SHA256Container,
	claims geo.Claims, TSLs geo.TransactionSignaturesLists) (block *ProposedBlock, err error) {

	if height == 0 {
		err = common.ErrInvalidBlockHeight
		return
	}

	block = &ProposedBlock{}
	block.Height = height
	block.ObserversConfHash = observersConfHash
	block.Claims = claims
	block.TSLs = TSLs

	err = block.updateHash()
	return
}

func (b *ProposedBlock) IsSigned() bool {
	return b.AuthorSignature != nil
}

func (b *ProposedBlock) AttachSignature(signature *ecdsa.Signature) {
	b.AuthorSignature = signature
}

func (b *ProposedBlock) MarshalBinary() (data []byte, err error) {
	hashData, err := b.Hash.MarshalBinary()
	if err != nil {
		return
	}

	observersConfHashData, err := b.ObserversConfHash.MarshalBinary()
	if err != nil {
		return
	}

	claimsData, err := b.Claims.MarshalBinary()
	if err != nil {
		return
	}

	tslsData, err := b.TSLs.MarshalBinary()
	if err != nil {
		return
	}

	authorSignatureData, err := b.AuthorSignature.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(
		// Static length structures.
		utils.MarshalUint64(b.Height),
		hashData,
		observersConfHashData,

		// Variadic length structures.
		// Header.
		utils.MarshalUint16(uint16(len(authorSignatureData))),
		utils.MarshalUint16(uint16(len(claimsData))),
		// (Last fields size should not be included.
		// On data marshalling - last field size is bounded by the bytes array right border.)

		// Body.
		authorSignatureData,
		claimsData,
		tslsData)

	return
}

func (b *ProposedBlock) UnmarshalBinary(data []byte) (err error) {
	const (
		fieldOffsetHeight                = 0
		fieldsOffsetHashData             = types.Uint64ByteSize
		fieldOffsetObserversConfHashData = fieldsOffsetHashData + types.HashSize
		fieldOffsetAuthorSignaturesSize  = fieldOffsetObserversConfHashData + types.HashSize
		fieldOffsetClaimsSize            = fieldOffsetAuthorSignaturesSize + types.Uint16ByteSize
		offsetVariadicLengthData         = fieldOffsetClaimsSize + types.Uint16ByteSize
	)

	b.Height, err = utils.UnmarshalUint64(data[fieldOffsetHeight:fieldsOffsetHashData])
	if err != nil {
		return
	}

	err = b.Hash.UnmarshalBinary(data[fieldsOffsetHashData:fieldOffsetObserversConfHashData])
	if err != nil {
		return
	}

	err = b.ObserversConfHash.UnmarshalBinary(data[fieldOffsetObserversConfHashData:fieldOffsetAuthorSignaturesSize])
	if err != nil {
		return
	}

	authorSignatureDataSegmentSize, err := utils.UnmarshalUint16(data[fieldOffsetAuthorSignaturesSize:fieldOffsetClaimsSize])
	if err != nil {
		return
	}

	claimsDataSegmentSize, err := utils.UnmarshalUint16(data[fieldOffsetClaimsSize : fieldOffsetClaimsSize+types.Uint16ByteSize])
	if err != nil {
		return
	}

	claimsDataSegmentOffset := offsetVariadicLengthData + authorSignatureDataSegmentSize
	TSLsDataOffset := claimsDataSegmentOffset + claimsDataSegmentSize

	b.AuthorSignature = &ecdsa.Signature{}
	err = b.AuthorSignature.UnmarshalBinary(data[offsetVariadicLengthData:claimsDataSegmentOffset])
	if err != nil {
		return
	}

	err = b.Claims.UnmarshalBinary(data[claimsDataSegmentOffset:TSLsDataOffset])
	if err != nil {
		return
	}

	err = b.TSLs.UnmarshalBinary(data[TSLsDataOffset:])
	if err != nil {
		return
	}

	return
}

func (b *ProposedBlock) updateHash() (err error) {
	// tsls marshalling
	claims, err := b.Claims.MarshalBinary()
	if err != nil {
		return
	}

	// tsls marshalling
	tsls, err := b.TSLs.MarshalBinary()
	if err != nil {
		return
	}

	// height marshalling
	height := utils.MarshalUint64(b.Height)

	// processing of the block
	dataSize := len(claims) + len(tsls) + len(height)
	data := make([]byte, 0, dataSize)
	data = utils.ChainByteSlices(claims, tsls, height)

	// hash generation
	b.Hash = types.NewSHA256Container(data)
	return
}

// --------------------------------------------------------------------------------------------------------------------
