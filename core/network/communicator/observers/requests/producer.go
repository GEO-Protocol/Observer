package requests

import (
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
)

// CandidateDigestBroadcast is used for broadcasting
// of new candidate blocks digests to the external observers.
type CandidateDigestBroadcast struct {
	request
	Digest *block.Digest
}

func NewCandidateDigestBroadcast(digest *block.Digest) *CandidateDigestBroadcast {
	return &CandidateDigestBroadcast{
		request: newRequest(nil),
		Digest:  digest,
	}
}

func (r *CandidateDigestBroadcast) MarshalBinary() (data []byte, err error) {
	requestBinary, err := r.request.MarshalBinary()
	if err != nil {
		return
	}

	digestBinary, err := r.Digest.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(requestBinary, digestBinary)
	return
}

func (r *CandidateDigestBroadcast) UnmarshalBinary(data []byte) (err error) {
	err = r.request.UnmarshalBinary(data[0:common.Uint16ByteSize])
	if err != nil {
		return
	}

	r.Digest = &block.Digest{}
	return r.Digest.UnmarshalBinary(data[common.Uint16ByteSize:])
}

// --------------------------------------------------------------------------------------------------------------------

type BlockSignaturesBroadcast struct {
	request
	Signatures *signatures.IndexedObserversSignatures
}

func NewBlockSignaturesBroadcast(signatures *signatures.IndexedObserversSignatures) *BlockSignaturesBroadcast {
	return &BlockSignaturesBroadcast{
		request:    newRequest(nil),
		Signatures: signatures,
	}
}

func (r *BlockSignaturesBroadcast) MarshalBinary() (data []byte, err error) {
	requestBinary, err := r.request.MarshalBinary()
	if err != nil {
		return
	}

	signaturesBinary, err := r.Signatures.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(requestBinary, signaturesBinary)
	return
}

func (r *BlockSignaturesBroadcast) UnmarshalBinary(data []byte) (err error) {
	err = r.request.UnmarshalBinary(data[0:common.Uint16ByteSize])
	if err != nil {
		return
	}

	r.Signatures = signatures.NewIndexedObserversSignatures(settings.ObserversMaxCount)
	return r.Signatures.UnmarshalBinary(data[common.Uint16ByteSize:])
}
