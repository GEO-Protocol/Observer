package constants

var (
	// Range 0..63 is reserved for the future needs.
	DataTypeRequestAccepted = 1

	// Requests and responses
	DataTypeRequestTimeFrames         uint8 = 70
	DataTypeResponseTimeFrame         uint8 = 71
	DataTypeRequestTSLBroadcast       uint8 = 75
	DataTypeResponseTSLApprove        uint8 = 76
	DataTypeRequestClaimBroadcast     uint8 = 80
	DataTypeResponseClaimApprove      uint8 = 81
	DataTypeRequestDigestBroadcast    uint8 = 85
	DataTypeResponseDigestApprove     uint8 = 86
	DataTypeResponseDigestReject      uint8 = 87
	DataTypeRequestBlockSigsBroadcast uint8 = 90
	DataTypeRequestChainTop           uint8 = 91
	DataTypeResponseChainTop          uint8 = 95
	DataTypeRequestBlockHashBroadcast uint8 = 96

	// Errors
	DataTypeInternalError   = 253
	DataTypeRequestRejected = 254
	DataTypeInvalidRequest  = 255
)

var (
	StreamTypeRequestTimeFrames         = []byte{DataTypeRequestTimeFrames}
	StreamTypeResponseTimeFrame         = []byte{DataTypeResponseTimeFrame}
	StreamTypeRequestTSLBroadcast       = []byte{DataTypeRequestTSLBroadcast}
	StreamTypeResponseTSLApprove        = []byte{DataTypeResponseTSLApprove}
	StreamTypeRequestClaimBroadcast     = []byte{DataTypeRequestClaimBroadcast}
	StreamTypeResponseClaimApprove      = []byte{DataTypeResponseClaimApprove}
	StreamTypeRequestDigestBroadcast    = []byte{DataTypeRequestDigestBroadcast}
	StreamTypeResponseDigestApprove     = []byte{DataTypeResponseDigestApprove}
	StreamTypeResponseDigestReject      = []byte{DataTypeResponseDigestReject}
	StreamTypeRequestBlockSigsBroadcast = []byte{DataTypeRequestBlockSigsBroadcast}
	StreamTypeRequestChainTop           = []byte{DataTypeRequestChainTop}
	StreamTypeResponseChainTop          = []byte{DataTypeResponseChainTop}
	StreamTypeRequestBlockHashBroadcast = []byte{DataTypeRequestBlockHashBroadcast}
)
