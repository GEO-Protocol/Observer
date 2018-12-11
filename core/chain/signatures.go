package chain

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/crypto/ecdsa"
	"geo-observers-blockchain/core/utils"
)

// IndexedObserversSignatures provides list of
// signatures along with corresponding observers indexes.
//
// Keeps order of observers and preserves position of
// each signature on serialization/deserialization.
//
// Note:
// This structure differs from "Signatures".
// "Signatures" does not serializes observers indexes.
type IndexedObserversSignatures struct {
	Signatures []*ecdsa.Signature
}

func NewIndexedObserversSignatures(count int) *IndexedObserversSignatures {
	return &IndexedObserversSignatures{
		Signatures: make([]*ecdsa.Signature, count, count),
	}
}

// todo: [enhance] think about little bit compact binary format.
func (s *IndexedObserversSignatures) MarshalBinary() (data []byte, err error) {
	data = utils.MarshalUint16(uint16(len(s.Signatures)))
	for index, sig := range s.Signatures {
		if sig == nil {
			continue
		}

		signatureBinary, err := sig.MarshalBinary()
		if err != nil {
			return nil, err
		}

		data = append(data,
			utils.ChainByteSlices(
				utils.MarshalUint16(uint16(index)),
				utils.MarshalUint16(uint16(len(signatureBinary))),
				signatureBinary)...)
	}
	return
}

func (s *IndexedObserversSignatures) UnmarshalBinary(data []byte) (err error) {
	if data == nil {
		return common.ErrInvalidDataFormat
	}

	count, err := utils.UnmarshalUint16(data[:types.Uint16ByteSize])
	if err != nil {
		return
	}

	s.Signatures = make([]*ecdsa.Signature, count, 0)

	var (
		offset        = types.Uint16ByteSize
		i      uint16 = 0
	)
	for i = 0; i < count; i++ {
		index, err := utils.UnmarshalUint16(data[offset : offset+types.Uint16ByteSize])
		offset += types.Uint16ByteSize
		if err != nil {
			return err
		}

		size, err := utils.UnmarshalUint16(data[offset : offset+types.Uint16ByteSize])
		offset += types.Uint16ByteSize
		if err != nil {
			return err
		}

		sig := &ecdsa.Signature{}
		err = sig.UnmarshalBinary(data[offset : offset+int(size)])
		if err != nil {
			return err
		}

		s.Signatures[index] = sig
	}
	return
}
