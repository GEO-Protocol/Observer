package requests

// RequestSynchronisationTimeFrames is used time frame synchronisation purposes.
// It is emitted by the observer as a request for information about
// current state of the ticker on other observers.
type RequestSynchronisationTimeFrames struct {
	request
}

func (r *RequestSynchronisationTimeFrames) MarshalBinary() ([]byte, error) {
	return r.request.MarshalBinary()
}

func (r *RequestSynchronisationTimeFrames) UnmarshalBinary(data []byte) error {
	return r.request.UnmarshalBinary(data)
}
