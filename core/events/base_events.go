package events

import "geo-observers-blockchain/core/common/errors"

type Event struct {
	componentID uint64
}

func NewEvent(id uint64) *Event {
	return &Event{componentID: id}
}

func (e *Event) ComponentID() uint64 {
	return e.componentID
}

type RequestEvent struct {
	Event

	id uint64
}

func NewRequestEvent(componentID uint64, id uint64) *RequestEvent {
	return &RequestEvent{
		Event: *NewEvent(componentID),
		id:    id,
	}
}

func (e *RequestEvent) ID() uint64 {
	return e.id
}

func (e *RequestEvent) Equals(other RequestEventer) bool {
	return e.id == other.ID() && e.componentID == other.ComponentID()
}

type ResponseEvent struct {
	RequestEvent
}

func NewResponseEvent(request *RequestEvent) *ResponseEvent {
	return &ResponseEvent{
		*request,
	}
}

func (r *ResponseEvent) Request() RequestEventer {
	return &r.RequestEvent
}

// ResultEvent represents standard Result/Error event.
type ResultEvent struct {
	Event

	Error errors.E
}

func NewErrorEvent(componentID uint64, id uint64) *ResultEvent {
	return &ResultEvent{
		Event: *NewEvent(componentID),
	}
}
