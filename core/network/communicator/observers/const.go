package observers

var (
	// Range 0..63 is reserved for the future needs.

	DataTypeBlockProposal  uint8 = 64
	DataTypeBlockSignature uint8 = DataTypeBlockProposal + 1
)

var (
	StreamTypeBlockProposal  = []byte{DataTypeBlockProposal}
	StreamTypeBlockSignature = []byte{DataTypeBlockSignature}
)
