package network

import (
	"geo-observers-blockchain/core/chain"
	"geo-observers-blockchain/core/chain/geo"
)

const (
	ChannelBufferSize = 1
)

// Todo: public keys of observers are available for public.
// This makes it possible to use reverse ssl, and to encrypt traffic between the observers.

type Receiver struct {
	Claims          chan<- geo.Claim
	TSLs            chan<- geo.TransactionSignaturesList
	BlockSignatures chan<- chain.BlockSignatures
	BlocksProposed  chan<- chain.BlockProposal
	BlockCandidates chan<- chain.SignedBlock
}

type Broadcaster struct {
	Claims <-chan geo.Claim
}

type Communicator struct {
}

func NewCommunicator() *Communicator {
	return &Communicator{
		Claims:          make(chan<- geo.Claim, ChannelBufferSize),
		TSLs:            make(chan<- geo.TransactionSignaturesList, ChannelBufferSize),
		BlockSignatures: make(chan<- chain.BlockSignatures, ChannelBufferSize),
		BlocksProposed:  make(chan<- chain.BlockProposal, ChannelBufferSize),
		BlockCandidates: make(chan<- chain.SignedBlock, ChannelBufferSize),
	}
}
