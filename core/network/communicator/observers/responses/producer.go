package responses

import (
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/crypto/ecdsa"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
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
	err = r.response.UnmarshalBinary(data[:common.Uint16ByteSize])
	if err != nil {
		return
	}

	return r.Signature.UnmarshalBinary(data[common.Uint16ByteSize:])
}
