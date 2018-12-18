package common

import (
	"errors"
)

func WrapError(err error, message string) error {
	return errors.New(message + " -> " + err.Error())
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
	// Memory error
	ErrInvalidCopyOperation     = errors.New("invalid copy operation")
	ErrMaxCountReached          = errors.New("max elements count has been reached")
	ErrNilInternalDataStructure = errors.New("attempt to reach nil object")
	ErrNilParameter             = errors.New("nil parameter occurred")

	// Channels
	ErrChannelTransferringFailed = errors.New("attempt to send info to the channel failed")

	// Sequences
	ErrEmptySequence = errors.New("empty sequence")

	// Bytes flow errors
	ErrInvalidDataFormat  = errors.New("invalid data format occurred")
	ErrBufferDiscarding   = errors.New("buffer can't be discarded")
	ErrUnexpectedDataType = errors.New("unexpected data type occurred in incoming data stream")

	// Chain
	ErrInvalidBlockHeight = errors.New("invalid block height")

	// Blocks Producer
	ErrAttemptToGenerateRedundantProposedBlock = errors.New("attempt to generate redundant block proposal")
)
