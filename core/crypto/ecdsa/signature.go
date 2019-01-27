package ecdsa

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
	"math/big"
)

type Signature struct {
	R *big.Int
	S *big.Int
}

func (s *Signature) MarshalBinary() (data []byte, err error) {
	if s.R == nil || s.S == nil {
		return nil, errors.NilInternalDataStructure
	}

	var (
		rDataBinary = s.R.Bytes()
		rSizeBinary = utils.MarshalUint16(uint16(len(rDataBinary)))

		sDataBinary = s.S.Bytes()
		sSizeBinary = utils.MarshalUint16(uint16(len(sDataBinary)))
	)

	return utils.ChainByteSlices(
		rSizeBinary, sSizeBinary, rDataBinary, sDataBinary), nil
}

func (s *Signature) UnmarshalBinary(data []byte) (err error) {
	if data == nil {
		return errors.InvalidDataFormat
	}

	const (
		sizeFieldLength = 2
		rSizeOffset     = 0
		sSizeOffset     = rSizeOffset + sizeFieldLength
		dataOffset      = sSizeOffset + sizeFieldLength
	)

	rSize, err := utils.UnmarshalUint16(data[rSizeOffset:sSizeOffset])
	if err != nil {
		return
	}
	if rSize == 0 {
		err = errors.InvalidDataFormat
		return
	}

	sSize, err := utils.UnmarshalUint16(data[sSizeOffset:dataOffset])
	if err != nil {
		return
	}
	if sSize == 0 {
		err = errors.InvalidDataFormat
		return
	}

	if uint16(len(data)) < rSize+sSize+(sizeFieldLength*2) {
		err = errors.InvalidDataFormat
		return
	}

	s.S = big.NewInt(0)
	s.R = big.NewInt(0)

	s.R.SetBytes(data[dataOffset : dataOffset+rSize])
	s.S.SetBytes(data[dataOffset+rSize : dataOffset+rSize+sSize])
	return
}

//---------------------------------------------------------------------------------------------------------------------

var (
	SignaturesMaxCount = settings.ObserversMaxCount
)

type Signatures struct {
	At []Signature
}

func (s *Signatures) Count() uint16 {
	return uint16(len(s.At))
}

func (s *Signatures) Add(sig Signature) error {
	if s.Count() < uint16(SignaturesMaxCount) {
		s.At = append(s.At, sig)
		return nil
	}

	return errors.MaxCountReached
}

// Format:
// 2B - Total signatures count.
// [2B, 2B, ... 2B] - At sizes.
// [NB, NB, ... NB] - At bodies.
func (s *Signatures) MarshalBinary() (data []byte, err error) {
	var (
		initialDataSize = common.Uint16ByteSize + // Total signatures count.
			common.Uint16ByteSize*s.Count() // At sizes fields.
	)

	data = make([]byte, 0, initialDataSize)
	data = append(data, utils.MarshalUint16(s.Count())...)
	signaturesBodies := make([][]byte, 0, s.Count())

	for _, signature := range s.At {
		sigBinary, err := signature.MarshalBinary()
		if err != nil {
			return nil, err
		}

		// Skip empty signature, if any.
		if len(sigBinary) == 0 {
			continue
		}

		// Append signature size directly to the data stream.
		data = append(utils.MarshalUint16(uint16(len(sigBinary))))

		// ClaimsHashes would be attached to the data after all signaturesBodies size fields would be written.
		signaturesBodies = append(signaturesBodies, sigBinary)
	}

	data = append(data, utils.ChainByteSlices(signaturesBodies...)...)
	return
}

func (s *Signatures) UnmarshalBinary(data []byte) (err error) {
	count, err := utils.UnmarshalUint16(data[:common.Uint16ByteSize])
	if err != nil {
		return
	}

	s.At = make([]Signature, count, count)
	if count == 0 {
		return
	}

	signaturesSizes := make([]uint16, 0, count)

	var i uint16
	var offset uint16 = common.Uint16ByteSize
	for i = 0; i < count; i++ {
		signatureSize, err := utils.UnmarshalUint16(data[offset : offset+common.Uint16ByteSize])
		if err != nil {
			return err
		}
		if signatureSize == 0 {
			err = errors.InvalidDataFormat
		}

		signaturesSizes = append(signaturesSizes, signatureSize)
	}

	offset = common.Uint16ByteSize
	for i = 0; i < count; i++ {
		signatureSize := signaturesSizes[i]
		err = s.At[i].UnmarshalBinary(data[offset : offset+signatureSize])
		if err != nil {
			return
		}

		offset += signatureSize
	}

	return
}
