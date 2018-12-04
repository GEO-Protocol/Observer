package geo

import (
	"fmt"
	"geo-observers-blockchain/core/chain/geo"
	"net"
)

const (
	ChannelBufferSize = 1
)

type NodesCommunicator struct {
	Claims chan<- geo.Claim
	TSLs   chan<- geo.TransactionSignaturesList
}

func NewNodesCommunicator() *NodesCommunicator {
	return &NodesCommunicator{
		Claims: make(chan<- geo.Claim, ChannelBufferSize),
		TSLs:   make(chan<- geo.TransactionSignaturesList, ChannelBufferSize),
	}
}

func (r *NodesCommunicator) Run(host string, port uint16, errors chan<- error) {
	listener, err := net.Listen("tcp", fmt.Sprint(host, ":", port))
	if err != nil {
		errors <- err
		return
	}

	//noinspection GoUnhandledErrorResult
	defer listener.Close()

	// Inform outer scope that initialisation was performed well
	// and no errors has been occurred.
	errors <- nil

	for {
		conn, err := listener.Accept()
		if err != nil {
			errors <- err
			return
		}

		go r.handleRequest(conn)
	}
}

func (r *NodesCommunicator) handleRequest(conn net.Conn) {
	// todo: implement

	//noinspection GoUnhandledErrorResult
	conn.Close()
}
