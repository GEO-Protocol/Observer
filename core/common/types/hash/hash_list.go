package hash

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
)

const (
	HashesListMaxCount = 1024
)

// todo: tests are needed.
type List struct {
	At []SHA256Container
}

func (s *List) Count() uint16 {
	return uint16(len(s.At))
}

func (s *List) Add(instance SHA256Container) error {
	if s.Count() < HashesListMaxCount {
		s.At = append(s.At, instance)
		return nil
	}

	return errors.MaxCountReached
}

// Format:
// 2B - Total hashes count.
// [NB, NB, ... NB] - hashes bodies.
func (s *List) MarshalBinary() (data []byte, err error) {
	if s.Count() > HashesListMaxCount {
		err = errors.TooLargeSequence
		return
	}

	var (
		i        uint16
		dataSize = common.Uint16ByteSize + // Total instances count.
			BytesSize*s.Count() // Instances bodies.
	)

	data = make([]byte, 0, dataSize)
	data = append(data, utils.MarshalUint16(s.Count())...)

	for i = 0; i < s.Count(); i++ {
		instanceBinary, err := s.At[i].MarshalBinary()
		if err != nil {
			return nil, err
		}

		data = append(data, instanceBinary...)
	}
	return
}

func (s *List) UnmarshalBinary(data []byte) (err error) {
	count, err := utils.UnmarshalUint16(data[:common.Uint16ByteSize])
	if err != nil {
		return errors.InvalidDataFormat
	}

	if count > HashesListMaxCount {
		return errors.TooLargeSequence
	}

	s.At = make([]SHA256Container, count, count)
	if count == 0 {
		return
	}

	var (
		i      uint16
		offset = common.Uint16ByteSize
	)

	for i = 0; i < count; i++ {
		err = s.At[i].UnmarshalBinary(data[offset : offset+BytesSize])
		if err != nil {
			return
		}

		offset += BytesSize
	}

	return
}
