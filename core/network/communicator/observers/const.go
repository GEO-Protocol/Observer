package observers

var (
	// Range 0..63 is reserved for the future needs.

	DataTypeBlockProposal uint8 = 64
)

var (
	StreamTypeBlockProposal = []byte{DataTypeBlockProposal}
)
