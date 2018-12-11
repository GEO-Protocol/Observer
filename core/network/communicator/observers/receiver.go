package observers

import (
	"bufio"
	"bytes"
	"fmt"
	"geo-observers-blockchain/core/chain"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/messages"
	"geo-observers-blockchain/core/utils"
	"net"
)

type Receiver struct {
	Claims chan geo.Claim
	TSLs   chan geo.TransactionSignaturesList
	//BlockSignatures chan *chain.BlockSignatures
	BlockSignatures chan *messages.SignatureMessage
	BlocksProposed  chan *chain.ProposedBlock
	//BlockCandidates chan *chain.BlockSigned
}

func NewReceiver() *Receiver {
	const kChannelBufferSize = 1

	return &Receiver{
		Claims: make(chan geo.Claim, kChannelBufferSize),
		TSLs:   make(chan geo.TransactionSignaturesList, kChannelBufferSize),
		//BlockSignatures: make(chan *chain.BlockSignatures, kChannelBufferSize),
		BlockSignatures: make(chan *messages.SignatureMessage, kChannelBufferSize),
		BlocksProposed:  make(chan *chain.ProposedBlock, kChannelBufferSize),
		//BlockCandidates: make(chan *chain.BlockSigned, kChannelBufferSize),
	}
}

func (r *Receiver) Run(host string, port uint16, errors chan<- error) {
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
			continue
		}

		go r.handleConnection(conn, errors)
	}
}

func (r *Receiver) handleConnection(conn net.Conn, errors chan<- error) {
	// todo: uncoment
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		//
		//
		//t := make([]byte, 1024, 1024)
		//k, err := reader.Read(t)
		//
		//
		////k, err := conn.Read(t)
		//fmt.Println(err, k, conn.RemoteAddr())
		//time.Sleep(time.Second * 2)

		dataPackage, err := r.receiveDataPackage(reader)
		if err != nil {
			errors <- err

			// In case of error - connection must be closed.
			// No more read attempt must be performed.
			return
		}

		err = r.parseAndRouteData(dataPackage)
		if err != nil {
			errors <- err

			// In case of error - connection must be closed.
			// No more read attempt must be performed.
			return
		}
	}
}

func (r *Receiver) receiveDataPackage(reader *bufio.Reader) (data []byte, err error) {
	const kPackageSizeHeaderBytes = 4
	packageSizeMarshaled := make([]byte, kPackageSizeHeaderBytes, kPackageSizeHeaderBytes)
	bytesRead, err := reader.Read(packageSizeMarshaled)
	if err != nil {
		return
	}
	if bytesRead != kPackageSizeHeaderBytes {
		return nil, common.ErrBufferDiscarding
	}

	packageSize, err := utils.UnmarshalUint32(packageSizeMarshaled)
	if err != nil {
		return
	}

	data = make([]byte, packageSize, packageSize)
	for {
		_, err := reader.Read(data)
		if err != nil {
			return nil, err
		}

		if len(data) == int(packageSize) {
			return data, nil
		}
	}
}

func (r *Receiver) parseAndRouteData(data []byte) (err error) {
	reader := bytes.NewReader(data)
	dataTypeHeader, err := reader.ReadByte()
	if err != nil {
		return
	}

	switch uint8(dataTypeHeader) {
	case DataTypeBlockProposal:
		{
			block := &chain.ProposedBlock{}
			err = block.UnmarshalBinary(data[1:])
			if err != nil {
				return
			}

			r.BlocksProposed <- block
			return
		}

	case DataTypeBlockSignature:
		{
			message := &messages.SignatureMessage{}
			err = message.UnmarshalBinary(data[1:])
			if err != nil {
				return
			}

			r.BlockSignatures <- message
			return
		}

	default:
		return common.ErrUnexpectedDataType
	}
}
