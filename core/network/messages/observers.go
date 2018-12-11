package messages

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/crypto/ecdsa"
	"geo-observers-blockchain/core/utils"
)

type SignatureMessage struct {
	Signature              *ecdsa.Signature
	AddresseeObserverIndex uint16
}

func (s *SignatureMessage) MarshalBinary() (data []byte, err error) {
	data = utils.MarshalUint16(s.AddresseeObserverIndex)

	sigData, err := s.Signature.MarshalBinary()
	if err != nil {
		return
	}

	data = append(data, sigData...)
	return
}

func (s *SignatureMessage) UnmarshalBinary(data []byte) (err error) {
	if data == nil {
		return common.ErrInvalidDataFormat
	}

	s.AddresseeObserverIndex, err = utils.UnmarshalUint16(data[:types.Uint16ByteSize])
	if err != nil {
		return
	}

	s.Signature = &ecdsa.Signature{}
	return s.Signature.UnmarshalBinary(data[types.Uint16ByteSize:])
}

// --------------------------------------------------------------------------------------------------------------------
