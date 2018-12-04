package external

import "geo-observers-blockchain/core/common/types"

type Configuration struct {
	Observers []*Observer

	// Version of the configuration received.
	// Might be calculated locally and might be different from the revisions of other observers.
	// At the moment, this number is simply increments on each new configuration received.
	Revision uint64
}

func NewConfiguration(rev uint64, observers []*Observer) *Configuration {
	return &Configuration{
		Observers: observers,
		Revision:  rev,
	}
}

func (c *Configuration) Hash() types.SHA256Container {
	dataSize := types.HashSize * len(c.Observers)
	data := make([]byte, 0, dataSize)

	for _, o := range c.Observers {
		k := o.Hash().Bytes
		data = append(data, k[:]...)
	}

	return types.NewSHA256Container(data)
}
