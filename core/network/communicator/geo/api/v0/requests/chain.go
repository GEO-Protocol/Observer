package requests

type LastBlockHeight struct {
	*requestWithResponse
}

func (r *LastBlockHeight) UnmarshalBinary(data []byte) (err error) {
	r.requestWithResponse = newRequestWithResponse()
	return
}

// --------------------------------------------------------------------------------------------------------------------
