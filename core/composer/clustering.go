package composer

import (
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"geo-observers-blockchain/core/responses"
	"sort"
)

// observersCluster holds list of observers IDs
// paired with last block hash, that is common for all of them.
type observersCluster struct {
	LastBlockHash    *hash.SHA256Container
	ObserversIndexes []uint16
}

func newObserversCluster(hash *hash.SHA256Container, firstObserverIndex uint16) *observersCluster {
	return &observersCluster{
		LastBlockHash:    hash,
		ObserversIndexes: []uint16{firstObserverIndex},
	}
}

// todo: tests needed
// todo: ensure responses from remote observers are signed.
//       Otherwise spoofing is possible.
func (c *Composer) performObserversResponsesClustering() (
	observersClusters []*observersCluster, observersResponses map[uint16]*responses.ChainInfo, e errors.E) {

	receivedResponsesCount := len(c.IncomingResponses.ChainInfo)
	if receivedResponsesCount == 0 {
		e = errors.AppendStackTrace(errors.NoResponseReceived)
		return
	}

	observersResponses = make(map[uint16]*responses.ChainInfo)
	observersClusters = make([]*observersCluster, 0, 0)

	for i := 0; i < receivedResponsesCount; i++ {
		// No hang is possible,channel length is checked previously.
		response := <-c.IncomingResponses.ChainInfo

		// Get or create cluster
		var cluster *observersCluster = nil
		for _, presentCluster := range observersClusters {
			if presentCluster.LastBlockHash.Compare(response.LastBlockHash) {
				cluster = presentCluster
				break
			}
		}

		if cluster == nil {
			// There is no cluster with such block hash. Create new one.
			cluster = newObserversCluster(response.LastBlockHash, response.ObserverIndex())
			observersClusters = append(observersClusters, cluster)

		} else {
			// Cluster found. Append current observer to it.
			cluster.ObserversIndexes = append(cluster.ObserversIndexes, response.ObserverIndex())
		}

		observersResponses[response.ObserverIndex()] = response
	}

	sort.Slice(
		observersClusters,
		func(i, j int) bool {
			return len(observersClusters[i].ObserversIndexes) < len(observersClusters[j].ObserversIndexes)
		})

	return
}
