package geo

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/crypto/lamport"
	"geo-observers-blockchain/core/utils"
)

const (
	TSLMemberBinarySize = common.Uint16ByteSize + lamport.SignatureBytesSize
)

type TSLMember struct {
	ID        uint16
	Signature *lamport.Signature
}

func NewTSLMember(memberID uint16) *TSLMember {
	return &TSLMember{
		ID:        memberID,
		Signature: &lamport.Signature{},
	}
}

func (member *TSLMember) MarshalBinary() (data []byte, err error) {
	idBinary := utils.MarshalUint16(member.ID)
	signatureBinary, err := member.Signature.MarshalBinary()
	if err != nil {
		return
	}

	return utils.ChainByteSlices(idBinary, signatureBinary), nil
}

func (member *TSLMember) UnmarshalBinary(data []byte) (err error) {
	if len(data) < TSLMemberBinarySize {
		return errors.InvalidDataFormat
	}

	member.ID, err = utils.UnmarshalUint16(data)
	if err != nil {
		return
	}

	member.Signature = &lamport.Signature{}
	return member.Signature.UnmarshalBinary(data[common.Uint16ByteSize:])
}
