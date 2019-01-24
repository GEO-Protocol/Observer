package chain

import "geo-observers-blockchain/core/common/errors"

type EventSynchronizationRequested struct {
	Chain *Chain
}

type EventSynchronizationFinished struct {
	Error errors.E
}
