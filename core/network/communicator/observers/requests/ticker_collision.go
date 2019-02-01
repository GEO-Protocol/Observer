package requests

type TimeFrameCollision struct {
	request
}

func NewTimeFrameCollision(destinationObserver uint16) *TimeFrameCollision {
	return &TimeFrameCollision{
		request: newRequest([]uint16{destinationObserver}),
	}
}

func (r *TimeFrameCollision) MarshalBinary() ([]byte, error) {
	return r.request.MarshalBinary()
}

func (r *TimeFrameCollision) UnmarshalBinary(data []byte) error {
	return r.request.UnmarshalBinary(data)
}
