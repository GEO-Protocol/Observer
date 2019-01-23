package geo

import (
	"bytes"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
	"sort"
)

const (
	TSLsMaxCount = common.GEOTransactionMaxParticipantsCount
)

type TSLs struct {
	At []*TSL
}

func (t *TSLs) Add(tsl *TSL) error {
	if tsl == nil {
		return errors.NilParameter
	}

	if t.Count() < TSLsMaxCount {
		t.At = append(t.At, tsl)
		return nil
	}

	return errors.MaxCountReached
}

func (t *TSLs) Count() uint16 {
	return uint16(len(t.At))
}

// todo: tests needed
func (t *TSLs) Sort() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
			return
		}
	}()

	sort.Slice(t.At, func(i, j int) bool {
		aBinaryData, err := t.At[i].MarshalBinary()
		if err != nil {
			panic(err)
		}

		bBinaryData, err := t.At[j].MarshalBinary()
		if err != nil {
			panic(err)
		}

		return bytes.Compare(aBinaryData, bBinaryData) == -1
	})

	return
}

// Format:
// 2B - Total TSLs count.
// [2B, 2B, ... 2B] - TSLs sizes.
// [NB, NB, ... NB] - TSLs bodies.
func (t *TSLs) MarshalBinary() (data []byte, err error) {
	var (
		initialDataSize = common.Uint16ByteSize + // Total TSLs count.
			common.Uint32ByteSize*t.Count() + // TSLs sizes fields.
			TSLMinBinarySize*t.Count() // TSLs bodies.
	)

	// todo: add internal data size prediction and allocate memory at once.
	data = make([]byte, 0, initialDataSize)
	data = append(data, utils.MarshalUint16(t.Count())...)

	TSLs := make([][]byte, 0, t.Count())
	for _, TSL := range t.At {
		TSLBinary, err := TSL.MarshalBinary()
		if err != nil {
			return nil, err
		}

		// Append TSL size directly to the data stream.
		data = append(data, utils.MarshalUint32(uint32(len(TSLBinary)))...)

		// TSL body would be attached to the data after all TSLs size fields would be written.
		TSLs = append(TSLs, TSLBinary)
	}

	data = append(data, utils.ChainByteSlices(TSLs...)...)
	return
}

func (t *TSLs) UnmarshalBinary(data []byte) (err error) {
	count, err := utils.UnmarshalUint16(data[:common.Uint16ByteSize])
	if err != nil {
		return
	}

	if count > TSLsMaxCount {
		return errors.InvalidDataFormat
	}

	t.At = make([]*TSL, count, count)
	if count == 0 {
		return
	}

	TSLsSizes := make([]uint32, 0, t.Count())

	var i uint16
	var offset uint32 = common.Uint16ByteSize
	for i = 0; i < count; i++ {
		TSLSize, err := utils.UnmarshalUint32(data[offset : offset+common.Uint32ByteSize])
		if err != nil {
			return err
		}
		if TSLSize == 0 {
			err = errors.InvalidDataFormat
		}

		TSLsSizes = append(TSLsSizes, TSLSize)
		offset += common.Uint32ByteSize
	}

	for i = 0; i < count; i++ {
		TSLSize := TSLsSizes[i]

		TSL := NewTSL()
		err := TSL.UnmarshalBinary(data[offset : offset+TSLSize])
		if err != nil {
			return err
		}

		t.At[i] = TSL
		offset += TSLSize
	}

	return
}
