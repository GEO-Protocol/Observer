package responses

import (
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/crypto/ecdsa"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/utils"
)

// CandidateDigestApprove is emitted by the observer in case when is is agree with next block candidate digest.
// todo: check each time is received that block hash is the same as candidate digest
//       (pool)
type CandidateDigestApprove struct {
	*response

	//BlockHash hash.SHA256Container
	Signature ecdsa.Signature
}

func NewCandidateDigestApprove(
	r requests.Request, observerNumber uint16,
	signature ecdsa.Signature) *CandidateDigestApprove {

	return &CandidateDigestApprove{
		response: newResponse(r, observerNumber),
		//BlockHash: blockHash,
		Signature: signature,
	}
}

func (r *CandidateDigestApprove) MarshalBinary() (data []byte, err error) {
	responseBinary, err := r.response.MarshalBinary()
	if err != nil {
		return
	}

	signatureBinary, err := r.Signature.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(responseBinary, signatureBinary)
	return
}

func (r *CandidateDigestApprove) UnmarshalBinary(data []byte) (err error) {
	r.response = &response{}
	err = r.response.UnmarshalBinary(data[:ResponseDefaultBytesLength])
	if err != nil {
		return
	}

	return r.Signature.UnmarshalBinary(data[ResponseDefaultBytesLength:])
}

// todo: add comment
type CandidateDigestReject struct {
	*response

	Hash hash.SHA256Container
}

func NewCandidateDigestReject(
	r requests.Request) *CandidateDigestReject {

	return &CandidateDigestReject{
		response: newResponse(r, r.ObserverIndex()),
		Hash:     r.(*requests.CandidateDigestBroadcast).Digest.BlockHash,
	}
}

func (r *CandidateDigestReject) MarshalBinary() (data []byte, err error) {
	responseBinary, err := r.response.MarshalBinary()
	if err != nil {
		return
	}

	hashBinary, err := r.Hash.MarshalBinary()
	if err != nil {
		return
	}

	data = utils.ChainByteSlices(responseBinary, hashBinary)
	return
}

func (r *CandidateDigestReject) UnmarshalBinary(data []byte) (err error) {
	r.response = &response{}
	err = r.response.UnmarshalBinary(data[:ResponseDefaultBytesLength])
	if err != nil {
		return
	}

	return r.Hash.UnmarshalBinary(data[ResponseDefaultBytesLength:])
}
