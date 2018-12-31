package constants

var (
	// Range 0..63 is reserved for the future needs.

	DataTypeBlockProposal  uint8 = 64
	DataTypeBlockSignature uint8 = 65

	// Requests and responses
	DataTypeRequestTimeFrames uint8 = 128
	DataTypeResponseTimeFrame uint8 = 129

	DataTypeRequestTSLBroadcast uint8 = 130
	DataTypeResponseTSLApprove  uint8 = 131

	DataTypeRequestClaimBroadcast uint8 = 132
	DataTypeResponseClaimApprove  uint8 = 133
)

var (
	StreamTypeBlockProposal  = []byte{DataTypeBlockProposal}
	StreamTypeBlockSignature = []byte{DataTypeBlockSignature}

	// Requests and responses
	StreamTypeRequestTimeFrames = []byte{DataTypeRequestTimeFrames}
	StreamTypeResponseTimeFrame = []byte{DataTypeResponseTimeFrame}

	StreamTypeRequestTSLBroadcast = []byte{DataTypeRequestTSLBroadcast}
	StreamTypeResponseTSLApprove  = []byte{DataTypeResponseTSLApprove}

	StreamTypeRequestClaimBroadcast = []byte{DataTypeRequestClaimBroadcast}
	StreamTypeResponseClaimApprove  = []byte{DataTypeResponseClaimApprove}
)
