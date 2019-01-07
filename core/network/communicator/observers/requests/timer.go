package requests

// SynchronisationTimeFrames is used for time frame synchronisation purposes.
// It is emitted by the observer as a request for information about
// current state of the ticker on other observers.
type SynchronisationTimeFrames struct {
	request
}

func (r *SynchronisationTimeFrames) MarshalBinary() ([]byte, error) {
	return r.request.MarshalBinary()
}

func (r *SynchronisationTimeFrames) UnmarshalBinary(data []byte) error {
	return r.request.UnmarshalBinary(data)
}
