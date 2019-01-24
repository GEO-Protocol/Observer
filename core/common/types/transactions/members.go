package transactions

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/utils"
)

const (
	MembersMaxCount      = common.GEOTransactionMaxParticipantsCount
	MembersMinBinarySize = common.Uint16ByteSize + MemberBinarySize
)

type Members struct {
	At []*Member
}

func (members *Members) Add(member *Member) error {
	if member == nil {
		return errors.NilParameter
	}

	if members.Count() < MembersMaxCount {
		members.At = append(members.At, member)
		return nil
	}

	return errors.MaxCountReached
}

func (members *Members) Count() uint16 {
	return uint16(len(members.At))
}

func (members *Members) MarshalBinary() (data []byte, err error) {
	totalMembersCount := len(members.At)
	if totalMembersCount > MembersMaxCount {
		err = errors.MaxCountReached
		return
	}

	totalBinarySize :=
		totalMembersCount*MemberBinarySize +
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

func (members *Members) UnmarshalBinary(data []byte) (err error) {
	if len(data) < MembersMinBinarySize {
		return errors.InvalidDataFormat
	}

	totalMembersCount, err := utils.UnmarshalUint16(data)
	if err != nil {
		return
	}

	if totalMembersCount > MembersMaxCount {
		return errors.InvalidDataFormat
	}

	members.At = make([]*Member, 0, int(totalMembersCount))
	for offset := common.Uint16ByteSize; offset < len(data); offset += MemberBinarySize {
		if len(data)-offset < MemberBinarySize {
			return errors.InvalidDataFormat
		}

		member := &Member{}
		membersData := data[offset : offset+MemberBinarySize]
		err = member.UnmarshalBinary(membersData)
		if err != nil {
			return
		}

		members.At = append(members.At, member)
	}

	return
}
