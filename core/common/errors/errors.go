package errors

import (
	"errors"
	"runtime/debug"
)

type StackTraceError interface {
	Error() error
	StackTrace() string
}

type StackTrace struct {
	err   error
	trace []byte
}

func WithStackTrace(e error) (err StackTraceError) {
	err = &StackTrace{
		err:   e,
		trace: debug.Stack(),
	}
	return
}

func (e *StackTrace) Error() error {
	return e.err
}

func (e *StackTrace) StackTrace() string {
	return string(e.trace)
}

// --------------------------------------------------------------------------------------------------------------------

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
	ExpectationFailed   = errors.New("expecteation failed")

	// Memory error
	InvalidCopyOperation     = errors.New("invalid copy operation")
	MaxCountReached          = errors.New("max elements count has been reached")
	NilInternalDataStructure = errors.New("attempt to reach nil object")
	NilParameter             = errors.New("nil parameter occurred")

	// Channels
	ChannelTransferringFailed = errors.New("attempt to send info to the channel failed")

	// Events
	UnexpectedEvent = errors.New("unexpected event occurred")

	// Sequences
	EmptySequence    = errors.New("empty sequence")
	NotFound         = errors.New("not found")
	TooLargeSequence = errors.New("too large sequence")
	Collision        = errors.New("collision detected")

	// Bytes flow errors
	InvalidDataFormat  = errors.New("invalid data format occurred")
	BufferDiscarding   = errors.New("buffer can't be discarded")
	UnexpectedDataType = errors.New("unexpected data type occurred in incoming data stream")

	// chain
	InvalidBlockHeight = errors.New("invalid block height")

	// Blocks Producer
	AttemptToGenerateRedundantBlock    = errors.New("attempt to generate redundant block proposal")
	DifferentBlocksGenerated           = errors.New("different block was generated")
	TSLsPoolReadFailed                 = errors.New("can't fetch block ready tsls from pool")
	ClaimsPoolReadFailed               = errors.New("can't fetch block ready claims from pool")
	InvalidBlockCandidateDigest        = errors.New("invalid block candidate digest")
	InvalidBlockCandidateDigestApprove = errors.New("invalid block candidate digest approve")
	InvalidBlockSignatures             = errors.New("invalid block signatures")

	// GEO Nodes receiver
	HashIntegrityCheckFailed = errors.New("hash integrity check failed")
)
