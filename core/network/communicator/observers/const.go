package observers

var (
	// Range 0..63 is reserved for the future needs.

	DataTypeBlockProposal  uint8 = 64
	DataTypeBlockSignature uint8 = 65

	// Requests and responses
	DataTypeRequestTimeFrames uint8 = 128
	DataTypeResponseTimeFrame uint8 = 129
)

var (
	StreamTypeBlockProposal  = []byte{DataTypeBlockProposal}
	StreamTypeBlockSignature = []byte{DataTypeBlockSignature}

	// Requests and responses
	StreamTypeRequestTimeFrames = []byte{DataTypeRequestTimeFrames}
	StreamTypeResponseTimeFrame = []byte{DataTypeResponseTimeFrame}
)
