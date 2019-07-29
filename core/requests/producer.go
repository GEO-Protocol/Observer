package requests

import (
	"geo-observers-blockchain/core/chain/block"
	"geo-observers-blockchain/core/chain/signatures"
	"geo-observers-blockchain/core/common/types/hash"
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
	err = r.request.UnmarshalBinary(data[0:RequestDefaultBytesLength])
	if err != nil {
		return
	}

	r.Digest = &block.Digest{}
	return r.Digest.UnmarshalBinary(data[RequestDefaultBytesLength:])
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
	err = r.request.UnmarshalBinary(data[0:RequestDefaultBytesLength])
	if err != nil {
		return
	}

	r.Signatures = signatures.NewIndexedObserversSignatures(settings.ObserversMaxCount)
	return r.Signatures.UnmarshalBinary(data[RequestDefaultBytesLength:])
}

// --------------------------------------------------------------------------------------------------------------------

type BlockHashBroadcast struct {
	request
	Hash *hash.SHA256Container
}

func NewBlockHashBroadcast(hash *hash.SHA256Container) *BlockHashBroadcast {
	return &BlockHashBroadcast{
		request: newRequest(nil),
		Hash:    hash,
	}
}

func (r *BlockHashBroadcast) MarshalBinary() (data []byte, err error) {
	requestBinary, err := r.request.MarshalBinary()
	if err != nil {
		return
	}

	hashBinary, err := r.Hash.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(requestBinary, hashBinary)
	return
}

func (r *BlockHashBroadcast) UnmarshalBinary(data []byte) (err error) {
	err = r.request.UnmarshalBinary(data[0:RequestDefaultBytesLength])
	if err != nil {
		return
	}

	r.Hash = &hash.SHA256Container{}
	return r.Hash.UnmarshalBinary(data[RequestDefaultBytesLength:])
}

// --------------------------------------------------------------------------------------------------------------------

type CollisionNotification struct {
	request
}

func NewCollisionNotification(destinationObserver uint16) *CollisionNotification {
	return &CollisionNotification{
		request: newRequest([]uint16{destinationObserver}),
	}
}

func (r *CollisionNotification) MarshalBinary() ([]byte, error) {
	return r.request.MarshalBinary()
}

func (r *CollisionNotification) UnmarshalBinary(data []byte) error {
	return r.request.UnmarshalBinary(data)
}
