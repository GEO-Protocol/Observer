package block

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/utils"
)

type Body struct {
	Index               uint64
	ExternalChainHeight uint64
	AuthorObserverIndex uint16
	Hash                hash.SHA256Container // todo: replace by BLAKE2b
	ObserversConfHash   hash.SHA256Container // todo: replace by BLAKE2b
	Claims              *geo.Claims
	TSLs                *geo.TransactionSignaturesLists
}

func (body *Body) SortInternalSequences() (err error) {
	err = body.Claims.Sort()
	if err != nil {
		return
	}

	err = body.TSLs.Sort()
	if err != nil {
		return
	}

	return
}

func (body *Body) UpdateHash(previousBlockHash hash.SHA256Container) (err error) {
	binaryHeight := utils.MarshalUint64(body.Index)
	generatedHash := hash.NewSHA256Container(binaryHeight)

	binaryExternalChainHeight := utils.MarshalUint64(body.ExternalChainHeight)
	generatedHash = hash.NewSHA256Container(
		utils.ChainByteSlices(generatedHash.Bytes[:], binaryExternalChainHeight))

	binaryAuthorObserverIndex := utils.MarshalUint16(body.AuthorObserverIndex)
	generatedHash = hash.NewSHA256Container(
		utils.ChainByteSlices(generatedHash.Bytes[:], binaryAuthorObserverIndex))

	generatedHash = hash.NewSHA256Container(
		utils.ChainByteSlices(generatedHash.Bytes[:], previousBlockHash.Bytes[:]))

	generatedHash = hash.NewSHA256Container(
		utils.ChainByteSlices(generatedHash.Bytes[:], body.ObserversConfHash.Bytes[:]))

	for _, claim := range body.Claims.At {
		data, err := claim.MarshalBinary()
		if err != nil {
			return err
		}

		generatedHash = hash.NewSHA256Container(
			utils.ChainByteSlices(generatedHash.Bytes[:], data))
	}

	for _, tsl := range body.TSLs.At {
		data, err := tsl.MarshalBinary()
		if err != nil {
			return err
		}

		generatedHash = hash.NewSHA256Container(
			utils.ChainByteSlices(generatedHash.Bytes[:], data))
	}

	body.Hash = generatedHash
	return
}

// WARN!
// GenerateDigest does not call SortInternalSequences() and UpdateHash()
// and doest not check if them was called in the past.
// This methods MUST be called before calling GenerateDigest.
func (body *Body) GenerateDigest() (digest *Digest, err error) {
	if body.AuthorObserverIndex >= common.ObserversMaxCount {
		err = errors.ExpectationFailed
		return
	}

	digest = &Digest{}
	digest.Index = body.Index
	digest.ExternalChainHeight = body.ExternalChainHeight
	digest.AuthorObserverIndex = body.AuthorObserverIndex
	digest.ObserversConfHash = body.ObserversConfHash
	digest.BlockHash = body.Hash

	for _, claim := range body.Claims.At {
		data, err := claim.MarshalBinary()
		if err != nil {
			return nil, err
		}

		digest.ClaimsHashes.At = append(
			digest.ClaimsHashes.At, hash.NewSHA256Container(data))
	}

	for _, tsl := range body.TSLs.At {
		data, err := tsl.MarshalBinary()
		if err != nil {
			return nil, err
		}

		digest.TSLsHashes.At = append(
			digest.TSLsHashes.At, hash.NewSHA256Container(data))
	}

	return
}

func (body *Body) MarshalBinary() (data []byte, err error) {
	observersConfHashData, err := body.ObserversConfHash.MarshalBinary()
	if err != nil {
		return
	}

	claimsData, err := body.Claims.MarshalBinary()
	if err != nil {
		return
	}

	tslsData, err := body.TSLs.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(
		utils.MarshalUint64(body.Index),
		utils.MarshalUint64(body.ExternalChainHeight),
		utils.MarshalUint16(body.AuthorObserverIndex),
		observersConfHashData,

		utils.MarshalUint16(uint16(len(claimsData))),
		claimsData,
		tslsData)

	return
}

func (body *Body) UnmarshalBinary(data []byte) (err error) {
	const (
		offsetHeight                = 0
		offsetExternalChainHeight   = offsetHeight + common.Uint64ByteSize
		offsetAuthorPosition        = offsetExternalChainHeight + common.Uint64ByteSize
		offsetObserversConfHashData = offsetAuthorPosition + common.Uint16ByteSize
		fieldOffsetClaimsSize       = offsetObserversConfHashData + hash.BytesSize
		offsetVariadicLengthData    = fieldOffsetClaimsSize + common.Uint16ByteSize
	)

	body.Index, err = utils.UnmarshalUint64(data[offsetHeight:offsetExternalChainHeight])
	if err != nil {
		return
	}

	body.ExternalChainHeight, err = utils.UnmarshalUint64(data[offsetExternalChainHeight:offsetAuthorPosition])
	if err != nil {
		return
	}

	body.AuthorObserverIndex, err = utils.UnmarshalUint16(data[offsetAuthorPosition:offsetObserversConfHashData])
	if err != nil {
		return
	}

	err = body.ObserversConfHash.UnmarshalBinary(data[offsetObserversConfHashData:fieldOffsetClaimsSize])
	if err != nil {
		return
	}

	claimsDataSegmentSize, err := utils.UnmarshalUint16(data[fieldOffsetClaimsSize:offsetVariadicLengthData])
	if err != nil {
		return
	}

	claimsDataSegmentOffset := uint16(offsetVariadicLengthData)
	TSLsDataOffset := claimsDataSegmentOffset + claimsDataSegmentSize

	body.Claims = &geo.Claims{}
	err = body.Claims.UnmarshalBinary(data[claimsDataSegmentOffset:TSLsDataOffset])
	if err != nil {
		return
	}

	body.TSLs = &geo.TransactionSignaturesLists{}
	err = body.TSLs.UnmarshalBinary(data[TSLsDataOffset:])
	if err != nil {
		return
	}

	return
}
