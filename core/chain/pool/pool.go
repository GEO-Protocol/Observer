package pool

import (
	"encoding"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/common/types/hash"
	"time"
)

type instance interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

type instances struct {
	At []instance
}

type Record struct {
	Instance instance

	//Approves collected from external observers.
	Approves [common.ObserversMaxCount]bool

	// Time of last attempt to send this Record to the external observers.
	LastSyncAttempt time.Time
}

func (r *Record) IsMajorityApprovesCollected() bool {
	var (
		positiveVotesPresent = 0
		negativeVotesPresent = 0
	)

	for _, vote := range r.Approves {
		if vote == true {
			positiveVotesPresent++
			if positiveVotesPresent >= common.ObserversConsensusCount {
				return true
			}

		} else {
			negativeVotesPresent++
			if negativeVotesPresent >= common.ObserversMaxCount-common.ObserversConsensusCount {
				return false
			}
		}
	}

	return false
}

type Pool struct {
	index map[hash.SHA256Container]*Record
}

func NewPool() *Pool {
	return &Pool{
		index: make(map[hash.SHA256Container]*Record),
	}
}

func (pool *Pool) Add(instance instance) (record *Record, err error) {
	data, err := instance.MarshalBinary()
	if err != nil {
		return
	}

	key := hash.NewSHA256Container(data)
	_, err = pool.ByHash(&key)
	if err != errors.NotFound {
		// Exactly the same item is already present in the pool.
		// It must not be replaced by the new value, to prevent votes dropping.
		err = errors.Collision
		return
	}

	record = &Record{
		Instance: instance,
	}

	pool.index[key] = record
	return
}

func (pool *Pool) Remove(hash *hash.SHA256Container) {
	delete(pool.index, *hash)
}

func (pool *Pool) ByHash(hash *hash.SHA256Container) (record *Record, err error) {
	record, isPresent := pool.index[*hash]
	if !isPresent {
		return nil, errors.NotFound
	}

	return
}
