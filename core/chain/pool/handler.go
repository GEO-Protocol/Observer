package pool

import (
	"encoding"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/network/external"
	"time"
)

var (
	// Specifies how often pool will try to broadcast
	// its items to the external observers.
	kTimeoutItemsSynchronisation = time.Duration(time.Second * 2)
)

// todo: tests needed
type Handler struct {
	// Instances, that was received directly from the GEO Nodes.
	// At this moment, there are 2 types of instances possible: claims and TSLs.
	// both are served by this pool implementation via interfaces hierarchy.
	IncomingInstances chan instance

	// Requests and responses,
	// that was received from external observers.
	IncomingRequestsInstanceBroadcast  chan *requests.RequestPoolInstanceBroadcast
	IncomingResponsesInstanceBroadcast chan *responses.ResponsePoolInstanceBroadcastApprove

	// Requests and responses that should be scheduled
	// for the sending to the external observers.
	OutgoingRequestsInstanceBroadcast  chan *requests.RequestPoolInstanceBroadcast
	OutgoingResponsesInstanceBroadcast chan *responses.ResponsePoolInstanceBroadcastApprove

	internalEventsBus chan interface{}

	// If true - than pool has items that must be synchronised with other observers.
	// Otherwise - all items of the pool has collected majority of (or more) positive votes.
	hasUnapprovedItems bool

	// Instances collection handler.
	pool *Pool

	// External observers configuration reporter.
	reporter *external.Reporter
}

func NewHandler(reporter *external.Reporter) *Handler {
	return &Handler{
		// Instances flow might be really intensive
		// in case if huge amount of nodes are trying to communicate to the observers.
		// The number of slots in this channel specifies how many instances
		// might be processed during one internal events processing round.
		IncomingInstances: make(chan instance, 64),

		// The same as IncomingInstances,
		// but for receiving claims and TSLs from the other observers.
		IncomingRequestsInstanceBroadcast:  make(chan *requests.RequestPoolInstanceBroadcast, 64),
		OutgoingRequestsInstanceBroadcast:  make(chan *requests.RequestPoolInstanceBroadcast, 64),
		IncomingResponsesInstanceBroadcast: make(chan *responses.ResponsePoolInstanceBroadcastApprove, 64),
		OutgoingResponsesInstanceBroadcast: make(chan *responses.ResponsePoolInstanceBroadcastApprove, 64),

		internalEventsBus: make(
			chan interface{},
			1), // one event per round might be processed, no need for more.

		pool:     NewPool(),
		reporter: reporter,
	}
}

func (h *Handler) Run(globalErrorsFlow chan<- error) {

	processErrorIfAny := func(err error) {
		if err != nil {
			globalErrorsFlow <- err
		}
	}

	for {
		conf, err := h.reporter.GetCurrentConfiguration()
		if err != nil {
			processErrorIfAny(err)
			return
		}

		select {
		case instance := <-h.IncomingInstances:
			processErrorIfAny(
				h.processNewInstance(instance, conf))

		case newInstanceRequest := <-h.IncomingRequestsInstanceBroadcast:
			processErrorIfAny(
				h.processNewInstanceRequest(newInstanceRequest, conf))

		case newInstanceResponse := <-h.IncomingResponsesInstanceBroadcast:
			processErrorIfAny(
				h.processNewInstanceResponse(newInstanceResponse, conf))

		case _ = <-time.After(kTimeoutItemsSynchronisation):
			processErrorIfAny(
				h.processItemsSynchronisation())
		}
	}
}

// BlockReadyInstances returns channel with items of the pool,
// that has been approved to be synchronised by the majority of the observers pools.
// This instances are ready to be included into the next block.
//
// Results are returned in channel to make it possible
// to call this method safely from other goroutines.
func (h *Handler) BlockReadyInstances() (channel chan instances, err error) {
	channel = make(chan instances, 1)
	event := &BlockReadyInstancesRequest{
		ResultsChannel: channel,
	}

	select {
	case h.internalEventsBus <- event:

	default:
		err = errors.ChannelTransferringFailed
	}

	return
}

// processNewInstance handles newly received claim or TSL from the GEO node:
// validates it for the correctness, adds to the pool and
// tries to broadcast the instance to the rest of observers.
func (h *Handler) processNewInstance(i instance, conf *external.Configuration) (err error) {
	// todo: add instance validation here.
	//       (attach crypto-backend, that is able to process lamport signatures)

	record, err := h.pool.Add(i)
	if err != nil {
		return
	}

	// Mark record as approved by the observers,
	// that has added it to the pool.
	record.Approves[conf.CurrentObserverIndex] = true

	// Setting this flag to true indicates that pool must try to
	// sync it's items with the rest observers ASAP.
	h.hasUnapprovedItems = true

	return h.requestRecordBroadcast(record)
}

// processNewInstanceRequest handles newly received claim or TSL from the external observer:
// validates it and adds to the pool. No further broadcast is done.
// It is optimistically assumed, that the original sender observer has
// also sent a copy of this info the rest of observers.
// In case if no - it would be obvious on block generation stage.
func (h *Handler) processNewInstanceRequest(
	r *requests.RequestPoolInstanceBroadcast, conf *external.Configuration) (err error) {

	if r.ObserverNumber() == conf.CurrentObserverIndex {
		err = errors.SuspiciousOperation
		return
	}

	// todo: add instance validation here.
	//       (attach crypto-backend, that is able to process lamport signatures)

	data, err := r.Instance.(encoding.BinaryMarshaler).MarshalBinary()
	if err != nil {
		return
	}

	key := hash.NewSHA256Container(data)
	record, err := h.pool.ByHash(&key)
	if err == errors.NotFound {

		// By default, received record should not be found in the pool.
		// In this case - it must be created and optimistically marked as approved by all observers
		// (it is assumed, that original observer has sent the same info to all other observers).
		record, err = h.pool.Add(r.Instance.(instance))
		if err != nil {
			return
		}

		for i := range record.Approves {
			record.Approves[i] = true
		}

	} else if err == errors.Collision {
		// In case if record is present - than it seems that request has been received
		// from the observer, that repeats it's request.
		// In this case - only vote of this observer must be rewritten.
		record.Approves[r.ObserverNumber()] = true
	}

	// Send approve to the observer, that has generated the request.
	response := responses.NewResponsePoolInstanceBroadcastApprove(
		r, conf.CurrentObserverIndex, &key)

	select {
	case h.OutgoingResponsesInstanceBroadcast <- response:
		return

	default:
		err = errors.ChannelTransferringFailed
		return
	}
}

// processNewInstanceResponse handles responses from external observers to the requests for instances approves.
func (h *Handler) processNewInstanceResponse(
	r *responses.ResponsePoolInstanceBroadcastApprove, conf *external.Configuration) (err error) {

	record, err := h.pool.ByHash(r.Hash)
	if err != nil {
		return
	}

	record.Approves[r.ObserverNumber()] = true
	return
}

// processItemsSynchronisation is launched from time to time.
// Checks if pool has items that need to be synchronized with some external observers.
// For each such item processItemsSynchronisation tries to perform synchronization flow.
func (h *Handler) processItemsSynchronisation() (err error) {
	if h.hasUnapprovedItems == false {
		return
	}

	anyItemsAreNotInSync := false
	for _, record := range h.pool.index {
		if record.IsMajorityApprovesCollected() == false {
			anyItemsAreNotInSync = true
			err = h.requestRecordBroadcast(record)
			if err != nil {
				return
			}
		}
	}

	if anyItemsAreNotInSync {
		h.hasUnapprovedItems = true
	}

	return
}

// requestRecordBroadcast checks which observers has not approved which items,
// and sens them to corresponding observers.
func (h *Handler) requestRecordBroadcast(record *Record) (err error) {
	record.LastSyncAttempt = time.Now()

	destinationObservers := make([]uint16, 0, common.ObserversMaxCount)
	for i := 0; i < common.ObserversMaxCount; i++ {
		if record.Approves[i] == false {
			destinationObservers = append(destinationObservers, uint16(i))
		}
	}

	request := requests.NewRequestPoolInstanceBroadcast(destinationObservers, record.Instance)
	select {
	case h.OutgoingRequestsInstanceBroadcast <- request:

	default:
		err = errors.ChannelTransferringFailed
	}

	return
}
