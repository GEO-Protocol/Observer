package constants

var (
	// Range 0..63 is reserved for the future needs.
	DataTypeRequestAccepted = 1

	// Requests and responses
	DataTypeRequestTimeFrames uint8 = 128
	DataTypeResponseTimeFrame uint8 = 129

	DataTypeRequestTSLBroadcast uint8 = 130
	DataTypeResponseTSLApprove  uint8 = 131

	DataTypeRequestClaimBroadcast uint8 = 132
	DataTypeResponseClaimApprove  uint8 = 133

	DataTypeRequestDigestBroadcast uint8 = 134
	DataTypeResponseDigestApprove  uint8 = 135

	DataTypeRequestBlockSignaturesBroadcast uint8 = 136

	DataTypeRequestChainTop  uint8 = 137
	DataTypeResponseChainTop uint8 = 138

	DataTypeRequestBlockHashBroadcast uint8 = 139

	DataTypeRequestTimeFrameCollision uint8 = 140

	// Errors
	DataTypeInternalError   = 253
	DataTypeRequestRejected = 254
	DataTypeInvalidRequest  = 255
)

var (
	// Requests and responses
	StreamTypeRequestTimeFrames = []byte{DataTypeRequestTimeFrames}
	StreamTypeResponseTimeFrame = []byte{DataTypeResponseTimeFrame}

	StreamTypeRequestTSLBroadcast = []byte{DataTypeRequestTSLBroadcast}
	StreamTypeResponseTSLApprove  = []byte{DataTypeResponseTSLApprove}

	StreamTypeRequestClaimBroadcast = []byte{DataTypeRequestClaimBroadcast}
	StreamTypeResponseClaimApprove  = []byte{DataTypeResponseClaimApprove}

	StreamTypeRequestDigestBroadcast = []byte{DataTypeRequestDigestBroadcast}
	StreamTypeResponseDigestApprove  = []byte{DataTypeResponseDigestApprove}

	StreamTypeRequestBlockSignaturesBroadcast = []byte{DataTypeRequestBlockSignaturesBroadcast}

	StreamTypeRequestChainTop  = []byte{DataTypeRequestChainTop}
	StreamTypeResponseChainTop = []byte{DataTypeResponseChainTop}

	StreamTypeRequestBlockHashBroadcast = []byte{DataTypeRequestBlockHashBroadcast}

	StreamTypeRequestTimeFrameCollision = []byte{DataTypeRequestTimeFrameCollision}
)
