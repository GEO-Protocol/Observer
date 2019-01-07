package pool

import "geo-observers-blockchain/core/common/types/hash"

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
