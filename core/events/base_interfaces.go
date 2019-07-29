package events

// Eventer implements basic entity for information exchange between components of the system.
type Eventer interface {

	// ComponentID returns ID of the component, that has generated this event.
	ComponentID() uint64
}

// RequestEventer allows requesting of some data by some component of the application.
type RequestEventer interface {
	Eventer

	// ID returns ID of the event in the scope of component.
	// Request is determined by both IDs: ComponentID of the event and ID of the request.
	ID() uint64

	// Equals returns true in case if other request has the same ComponentID and ID.
	Equals(other RequestEventer) bool
}

type ResponseEventer interface {
	Eventer

	Request() RequestEventer
	ID() uint64
}
