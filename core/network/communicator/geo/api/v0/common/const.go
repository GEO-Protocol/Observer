package common

const (
	ProtocolVersion = 0
)

const (
	// Reserved system constants.
	// ...

	// Global chain info
	ReqChainLastBlockNumber = 32

	// TSLs.
	ReqTSLAppend    = 64
	ReqTSLIsPresent = 66
	ReqTSLGet       = 68

	// Claims
	ReqClaimAppend    = 128
	ReqClaimIsPresent = 130
)
