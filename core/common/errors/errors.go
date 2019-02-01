package errors

import (
	"errors"
	"runtime/debug"
)

type E interface {
	Error() error
	StackTrace() string
	IsFatal() bool
}

type StackTrace struct {
	err   error
	trace []byte
	fatal bool
}

func New(message string) (err E) {
	err = &StackTrace{
		err:   errors.New(message),
		trace: debug.Stack(),
	}
	return
}

func Fatal(message string) (err E) {
	err = &StackTrace{
		err:   errors.New(message),
		trace: debug.Stack(),
		fatal: true,
	}
	return
}

func AppendStackTrace(e error) (err E) {
	err = &StackTrace{
		err:   e,
		trace: debug.Stack(),
	}
	return
}

func (e *StackTrace) IsFatal() bool {
	return e.fatal
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
	// Configuration
	InvalidObserverIndex = errors.New("invalid observer index")

	// Common
	SuspiciousOperation = errors.New("suspicious operation")
	ExpectationFailed   = errors.New("expecteation failed")
	NoConsensus         = errors.New("no consensus")
	ValidationFailed    = errors.New("validation failed")
	TimeoutFired        = errors.New("timeout fired ")

	// Memory error
	InvalidCopyOperation     = errors.New("invalid copy operation")
	MaxCountReached          = errors.New("max elements count has been reached")
	NilInternalDataStructure = errors.New("attempt to reach nil object")
	NilParameter             = errors.New("nil parameter occurred")
	InvalidParameter         = errors.New("invalid parameter")

	// Channels
	ChannelTransferringFailed = errors.New("attempt to send info to the channel failed")

	// Events
	UnexpectedEvent = errors.New("unexpected event occurred")

	// Sequences
	EmptySequence    = errors.New("empty sequence")
	NotFound         = errors.New("not found")
	NoData           = errors.New("no data")
	TooLargeSequence = errors.New("too large sequence")
	Collision        = errors.New("collision detected")

	// Bytes flow errors
	InvalidDataFormat  = errors.New("invalid data format occurred")
	BufferDiscarding   = errors.New("buffer can't be discarded")
	UnexpectedDataType = errors.New("unexpected data type occurred in incoming data stream")

	// chain
	InvalidBlockHeight = errors.New("invalid block height")
	InvalidChainHeight = errors.New("invalid chain height")

	// Blocks Producer
	AttemptToGenerateRedundantBlock    = errors.New("attempt to generate redundant block proposal")
	DifferentBlocksGenerated           = errors.New("different block was generated")
	TSLsPoolReadFailed                 = errors.New("can't fetch block ready tsls from pool")
	ClaimsPoolReadFailed               = errors.New("can't fetch block ready claims from pool")
	InvalidBlockCandidateDigest        = errors.New("invalid block candidate digest")
	InvalidTimeFrame                   = errors.New("invalid time frame")
	InvalidBlockCandidateDigestApprove = errors.New("invalid block candidate digest approve")
	InvalidBlockSignatures             = errors.New("invalid block signatures")

	// GEO Nodes receiver
	HashIntegrityCheckFailed = errors.New("hash integrity check failed")
)

var (
	// Composer
	NoResponseReceived = errors.New("no ChainTop response received")
	SyncFailed         = errors.New("sync failed")
)
