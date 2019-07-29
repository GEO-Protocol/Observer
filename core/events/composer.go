package events

import (
	"geo-observers-blockchain/core/common/errors"
)

type SyncFinished struct {
	Error errors.E
}
