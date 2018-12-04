package lamport

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/utils"
)

const (
	PubKeyBytesSize = 1024 * 16
)

type PubKey struct {
	Bytes [PubKeyBytesSize]byte
}

func (h *PubKey) MarshalBinary() (data []byte, err error) {
	return h.Bytes[:PubKeyBytesSize], nil
}

func (h *PubKey) UnmarshalBinary(data []byte) error {
	if len(data) < PubKeyBytesSize {
		return common.ErrInvalidDataFormat
	}

	if copy(h.Bytes[:], data[:PubKeyBytesSize]) == PubKeyBytesSize {
		return nil

	} else {
		return common.ErrInvalidCopyOperation

	}
}

// --------------------------------------------------------------------------------------------------------------------

const (
	PubKeysMaxCount = common.GEO_TransactionMaxParticipant
)

type PubKeys struct {
	At []*PubKey
}

func (p *PubKeys) Add(key *PubKey) error {
	if key == nil {
		return common.ErrNilParameter
	}

	if p.Count() < PubKeysMaxCount {
		p.At = append(p.At, key)
		return nil
	}

	return common.ErrMaxCountReached
}

func (p *PubKeys) Count() uint16 {
	return uint16(len(p.At))
}

func (p *PubKeys) MarshalBinary() (data []byte, err error) {
	var (
		totalBinarySize = p.Count()*PubKeyBytesSize +
			types.Uint16ByteSize
	)

	data = make([]byte, 0, totalBinarySize)
	data = append(data, utils.MarshalUint16(p.Count())...)

	for _, key := range p.At {
		keyBinary, err := key.MarshalBinary()
		if err != nil {
			return nil, err
		}

		data = append(data, keyBinary...)
	}

	return
}

func (p *PubKeys) UnmarshalBinary(data []byte) (err error) {
	count, err := utils.UnmarshalUint16(data[:types.Uint16ByteSize])
	if err != nil {
		return
	}

	p.At = make([]*PubKey, count, count)
	if count == 0 {
		return
	}

	var offset uint32 = types.Uint16ByteSize
	for i := 0; i < int(count); i++ {
		pubKey := &PubKey{}
		err = pubKey.UnmarshalBinary(data[offset : offset+PubKeyBytesSize])
		if err != nil {
			return err
		}

		p.At[i] = pubKey
		offset += PubKeyBytesSize
	}

	return
}
