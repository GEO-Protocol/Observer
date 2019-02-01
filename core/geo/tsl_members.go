package geo

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
)

var (
	TSLMembersMaxCount      = settings.GEOTransactionMaxParticipantsCount
	TSLMembersMinBinarySize = common.Uint16ByteSize + TSLMemberBinarySize
)

type TSLMembers struct {
	At []*TSLMember
}

func (members *TSLMembers) Add(member *TSLMember) error {
	if member == nil {
		return errors.NilParameter
	}

	if members.Count() < uint16(TSLMembersMaxCount) {
		members.At = append(members.At, member)
		return nil
	}

	return errors.MaxCountReached
}

func (members *TSLMembers) Count() uint16 {
	return uint16(len(members.At))
}

func (members *TSLMembers) MarshalBinary() (data []byte, err error) {
	totalMembersCount := len(members.At)
	if totalMembersCount > TSLMembersMaxCount {
		err = errors.MaxCountReached
		return
	}

	totalBinarySize :=
		totalMembersCount*ClaimMemberBinarySize +
			common.Uint16ByteSize // members count

	data = make([]byte, 0, totalBinarySize)
	data = utils.ChainByteSlices(data, utils.MarshalUint16(uint16(totalMembersCount)))

	for _, member := range members.At {
		memberBinary, err := member.MarshalBinary()
		if err != nil {
			return nil, err
		}

		data = utils.ChainByteSlices(data, memberBinary)
	}

	return
}

func (members *TSLMembers) UnmarshalBinary(data []byte) (err error) {
	if len(data) < TSLMembersMinBinarySize {
		return errors.InvalidDataFormat
	}

	totalMembersCount, err := utils.UnmarshalUint16(data)
	if err != nil {
		return
	}

	if totalMembersCount > uint16(TSLMembersMaxCount) {
		return errors.InvalidDataFormat
	}

	members.At = make([]*TSLMember, 0, int(totalMembersCount))
	for offset := common.Uint16ByteSize; offset < len(data); offset += ClaimMemberBinarySize {
		if len(data)-offset < ClaimMemberBinarySize {
			return errors.InvalidDataFormat
		}

		member := &TSLMember{}
		membersData := data[offset : offset+ClaimMemberBinarySize]
		err = member.UnmarshalBinary(membersData)
		if err != nil {
			return
		}

		members.At = append(members.At, member)
	}

	return
}
