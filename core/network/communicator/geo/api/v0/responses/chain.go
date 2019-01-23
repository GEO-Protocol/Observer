package responses

import "geo-observers-blockchain/core/utils"

type LastBlockHeight struct {
	Height uint64
}

func (r *LastBlockHeight) MarshalBinary() (data []byte, err error) {
	return utils.MarshalUint64(r.Height), nil
}

// --------------------------------------------------------------------------------------------------------------------
