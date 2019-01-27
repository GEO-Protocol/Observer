package pool

import (
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/common/types/transactions"
)

type EventBlockReadyInstancesRequest struct {
	Results chan *instances
	Errors  chan error
}

type EventBlockReadyInstancesByHashesRequest struct {
	Instances chan *instances
	Errors    chan error
	Hashes    []hash.SHA256Container
}

type EventItemsDroppingRequest struct {
	Errors chan error
	Hashes []hash.SHA256Container
}

type EventInstanceIsPresentRequest struct {
	Errors chan error
	Result chan bool
	TxID   *transactions.TxID
}
