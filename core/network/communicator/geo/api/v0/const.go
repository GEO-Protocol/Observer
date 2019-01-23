package v0

const (
	ProtocolVersion = 0
)

const (
	// Reserved system constants.
	// ...

	// Global chain info
	ReqChainLastBlockNumber = 32

	// TSLs.
	ReqTSLAppend = 64
	_

	ReqTSLGet = 66

	ReqTSLIsPresent = 68

	// Claims.
	ReqClaimAppend = 128
	_

	ReqClaimGet = 130

	ReqClaimIsPresent = 132
)
