package observers

import (
	"bufio"
	"bytes"
	"fmt"
	"geo-observers-blockchain/core/chain"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/messages"
	"geo-observers-blockchain/core/requests"
	"geo-observers-blockchain/core/responses"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"strings"
	"time"
)

type Receiver struct {
	OutgoingEventsConnectionClosed chan interface{}

	Claims chan geo.Claim
	TSLs   chan geo.TransactionSignaturesList
	//BlockSignatures chan *chain.BlockSignatures
	BlockSignatures chan *messages.SignatureMessage
	BlocksProposed  chan *chain.ProposedBlock
	//BlockCandidates chan *chain.BlockSigned
	Requests  chan requests.Request
	Responses chan responses.Response
}

func NewReceiver() *Receiver {
	const ChannelBufferSize = 1

	return &Receiver{
		OutgoingEventsConnectionClosed: make(chan interface{}, 1),

		Claims: make(chan geo.Claim, ChannelBufferSize),
		TSLs:   make(chan geo.TransactionSignaturesList, ChannelBufferSize),
		//BlockSignatures: make(chan *chain.BlockSignatures, ChannelBufferSize),
		BlockSignatures: make(chan *messages.SignatureMessage, ChannelBufferSize),
		BlocksProposed:  make(chan *chain.ProposedBlock, ChannelBufferSize),
		//BlockCandidates: make(chan *chain.BlockSigned, ChannelBufferSize),
		Requests:  make(chan requests.Request, ChannelBufferSize),
		Responses: make(chan responses.Response, ChannelBufferSize),
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
	defer conn.Close()

	reader := bufio.NewReader(conn)

	for {
		dataPackage, err := r.receiveDataPackage(reader)
		if err != nil {
			if err == io.EOF {
				r.sendEvent(r.OutgoingEventsConnectionClosed, &EventConnectionClosed{
					RemoteHost: strings.Split(conn.RemoteAddr().String(), ":")[0],
					RemotePort: strings.Split(conn.RemoteAddr().String(), ":")[1],
				})
				return
			}

			errors <- err

			// In case of error - connection must be closed.
			// No more read attempt must be performed.
			return
		}

		r.log().Debug("[TX<=] ", len(dataPackage), "B received, ", conn.RemoteAddr())

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

	case DataTypeRequestTimeFrames:
		{
			request := &requests.RequestTimeFrames{}
			err = request.UnmarshalBinary(data[1:])
			if err != nil {
				return
			}

			r.Requests <- request
			return
		}

	case DataTypeResponseTimeFrame:
		{
			r.log().Info("Time Frame received")

			response := &responses.ResponseTimeFrame{}
			err = response.UnmarshalBinary(data[1:])
			if err != nil {
				return
			}

			// Time frame response MUST be extended with time of receiving.
			// It would be used further for time offset corrections.
			response.Received = time.Now()

			r.Responses <- response
			return
		}

	default:
		return common.ErrUnexpectedDataType
	}
}

func (r *Receiver) sendEvent(channel chan interface{}, event interface{}) {
	select {
	case channel <- event:
		{
		}
	default:
		// todo: log error
	}
}

func (r *Receiver) log() *log.Entry {
	return log.WithFields(log.Fields{"subsystem": "receiver"})
}
