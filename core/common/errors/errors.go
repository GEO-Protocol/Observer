package errors

import (
	"errors"
)

func ExpandWithStackTrace(e error) (err error) {
	// todo: add stacktrace collection
	return e
}

func SendErrorIfAny(err error, errors chan error) {
	select {
	case errors <- err:
		return

	default:
		// todo: add logging here
		return
	}
}

var (
	// Common
	SuspiciousOperation = errors.New("suspicious operation")

	// Memory error
	InvalidCopyOperation     = errors.New("invalid copy operation")
	MaxCountReached          = errors.New("max elements count has been reached")
	NilInternalDataStructure = errors.New("attempt to reach nil object")
	NilParameter             = errors.New("nil parameter occurred")

	// Channels
	ChannelTransferringFailed = errors.New("attempt to send info to the channel failed")

	// Sequences
	EmptySequence    = errors.New("empty sequence")
	NotFound         = errors.New("not found")
	TooLargeSequence = errors.New("too large sequence")
	Collision        = errors.New("collision detected")

	// Bytes flow errors
	InvalidDataFormat  = errors.New("invalid data format occurred")
	BufferDiscarding   = errors.New("buffer can't be discarded")
	UnexpectedDataType = errors.New("unexpected data type occurred in incoming data stream")

	// Chain
	InvalidBlockHeight = errors.New("invalid block height")

	// Blocks Producer
	ErrAttemptToGenerateRedundantProposedBlock = errors.New("attempt to generate redundant block proposal")

	// GEO Nodes receiver
	HashIntegrityCheckFailed = errors.New("hash integrity check failed")
)
