package geo

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/crypto/lamport"
	"geo-observers-blockchain/core/utils"
)

const (
	ClaimMemberBinarySize = common.Uint16ByteSize + lamport.PubKeyBytesSize
)

type ClaimMember struct {
	ID     uint16
	PubKey *lamport.PubKey
}

func NewClaimMember(memberID uint16) *ClaimMember {
	return &ClaimMember{
		ID:     memberID,
		PubKey: &lamport.PubKey{},
	}
}

func (member *ClaimMember) MarshalBinary() (data []byte, err error) {
	idBinary := utils.MarshalUint16(member.ID)
	pubKeyBinary, err := member.PubKey.MarshalBinary()
	if err != nil {
		return
	}

	return utils.ChainByteSlices(idBinary, pubKeyBinary), nil
}

func (member *ClaimMember) UnmarshalBinary(data []byte) (err error) {
	if len(data) < ClaimMemberBinarySize {
		return errors.InvalidDataFormat
	}

	member.ID, err = utils.UnmarshalUint16(data)
	if err != nil {
		return
	}

	member.PubKey = &lamport.PubKey{}
	return member.PubKey.UnmarshalBinary(data[common.Uint16ByteSize:])
}
