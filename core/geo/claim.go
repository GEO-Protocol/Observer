package geo

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/crypto/lamport"
	"geo-observers-blockchain/core/utils"
)

type Claim struct {
	TxUUID  *types.TransactionUUID
	PubKeys *lamport.PubKeys
}

func NewClaim() *Claim {
	return &Claim{
		TxUUID:  types.NewTransactionUUID(),
		PubKeys: &lamport.PubKeys{},
	}
}

func (claim *Claim) MarshalBinary() (data []byte, err error) {
	if claim.TxUUID == nil || claim.PubKeys == nil {
		return nil, common.ErrNilInternalDataStructure
	}

	transactionUUIDBinary, err := claim.TxUUID.MarshalBinary()
	if err != nil {
		return
	}

	keysBinary, err := claim.PubKeys.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(transactionUUIDBinary, keysBinary)
	return
}

func (claim *Claim) UnmarshalBinary(data []byte) (err error) {
	const (
		offsetUUIDData = 0
		offsetKeysData = offsetUUIDData + types.TransactionUUIDSize

		minDataLength = offsetKeysData + types.Uint16ByteSize
	)

	if len(data) < minDataLength {
		return common.ErrInvalidDataFormat
	}

	err = claim.TxUUID.UnmarshalBinary(data[:types.TransactionUUIDSize])
	if err != nil {
		return
	}

	err = claim.PubKeys.UnmarshalBinary(data[offsetKeysData:])
	if err != nil {
		return
	}

	return
}

// --------------------------------------------------------------------------------------------------------------------

const (
	ClaimsMaxCount = 1024 * 16
)

type Claims struct {
	At []*Claim
}

func (c *Claims) Add(claim *Claim) error {
	if claim == nil {
		return common.ErrNilParameter
	}

	if c.Count() < ClaimsMaxCount {
		c.At = append(c.At, claim)
		return nil
	}

	return common.ErrMaxCountReached
}

func (c *Claims) Count() uint16 {
	return uint16(len(c.At))
}

// Format:
// 2B - Total claims count.
// [4B, 4B, ... 4B] - Claims sizes.
// [NB, NB, ... NB] - Claims bodies.
func (c *Claims) MarshalBinary() (data []byte, err error) {
	var (
		initialDataSize = types.Uint16ByteSize + // Total claims count.
			types.Uint16ByteSize*c.Count() // Claims sizes fields.
	)

	data = make([]byte, 0, initialDataSize)
	data = append(data, utils.MarshalUint16(c.Count())...)
	claims := make([][]byte, 0, c.Count())

	for _, claim := range c.At {
		claimBinary, err := claim.MarshalBinary()
		if err != nil {
			return nil, err
		}

		// Skip empty claim, if any.
		if len(claimBinary) == 0 {
			continue
		}

		// Append claim size directly to the data stream.
		data = append(data, utils.MarshalUint32(uint32(len(claimBinary)))...)

		// Claims would be attached to the data after all claims size fields would be written.
		claims = append(claims, claimBinary)
	}

	data = append(data, utils.ChainByteSlices(claims...)...)
	return
}

func (c *Claims) UnmarshalBinary(data []byte) (err error) {
	count, err := utils.UnmarshalUint16(data[:types.Uint16ByteSize])
	if err != nil {
		return
	}

	c.At = make([]*Claim, count, count)
	if count == 0 {
		return
	}

	var i uint16
	for i = 0; i < count; i++ {
		c.At[i] = NewClaim()
	}

	claimsSizes := make([]uint32, 0, count)

	var offset uint32 = types.Uint16ByteSize
	for i = 0; i < count; i++ {
		claimSize, err := utils.UnmarshalUint32(data[offset : offset+types.Uint32ByteSize])
		if err != nil {
			return err
		}
		if claimSize == 0 {
			err = common.ErrInvalidDataFormat
		}

		claimsSizes = append(claimsSizes, claimSize)
		offset += types.Uint32ByteSize
	}

	for i = 0; i < count; i++ {
		claim := NewClaim()
		claimSize := claimsSizes[i]

		err = claim.UnmarshalBinary(data[offset : offset+claimSize])
		if err != nil {
			return err
		}

		offset += claimSize
		c.At[i] = claim
	}

	return
}
