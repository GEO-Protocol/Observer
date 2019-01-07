package external

import "geo-observers-blockchain/core/common/types/hash"

type Configuration struct {
	Observers []*Observer

	CurrentObserverIndex uint16

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

func (c *Configuration) Hash() hash.SHA256Container {
	dataSize := hash.BytesSize * len(c.Observers)
	data := make([]byte, 0, dataSize)

	for _, o := range c.Observers {
		k := o.Hash().Bytes
		data = append(data, k[:]...)
	}

	return hash.NewSHA256Container(data)
}

// CurrentExternalChainHeight returns current block number of the external blockchain.
// It is implemented as a method of current configuration because this number would change relatively often,
// but the observers configuration - only once a week, so there is no need to regenerate configuration each time
// when external blockchain height has been increased.
func (c *Configuration) CurrentExternalChainHeight() uint64 {
	// todo: implement proxy method calling middleware
	return 1
}
