package requests

type RequestTimeFrames struct {
	request
}

func (r *RequestTimeFrames) MarshalBinary() ([]byte, error) {
	return r.request.MarshalBinary()
}

func (r *RequestTimeFrames) UnmarshalBinary(data []byte) error {
	return r.request.UnmarshalBinary(data)
}
