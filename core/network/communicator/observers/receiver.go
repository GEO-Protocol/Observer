package observers

import (
	"bufio"
	"bytes"
	"fmt"
	//"geo-observers-blockchain/core/chain"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/observers/constants"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"io"
	"net"
	"reflect"
	"strings"
	"time"
)

type Receiver struct {
	OutgoingEventsConnectionClosed chan interface{}

	Claims chan geo.Claim
	TSLs   chan geo.TSL
	//BlockSignatures chan *chain.BlockSignatures
	//BlockSignatures chan *messages.SignatureMessage
	//BlocksProposed  chan *chain.ProposedBlockData
	//BlockCandidates chan *chain.BlockSigned
	Requests  chan requests.Request
	Responses chan responses.Response
}

func NewReceiver() *Receiver {
	const ChannelBufferSize = 1

	return &Receiver{
		OutgoingEventsConnectionClosed: make(chan interface{}, 1),

		Claims: make(chan geo.Claim, ChannelBufferSize),
		TSLs:   make(chan geo.TSL, ChannelBufferSize),
		//BlockSignatures: make(chan *chain.BlockSignatures, ChannelBufferSize),
		//BlockSignatures: make(chan *messages.SignatureMessage, ChannelBufferSize),
		//BlocksProposed:  make(chan *chain.ProposedBlockData, ChannelBufferSize),
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

	r.log().WithFields(log.Fields{
		"Host": host,
		"Port": port,
	}).Info("Started")

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

		r.logIngress(len(dataPackage), conn)

		err = r.parseAndRouteData(dataPackage)
		if err != nil {
			errors <- err

			// In case of error - connection must be closed.
			// No more read attempt must be performed.
			r.log().Debug("connection will now be closed")
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
		return nil, errors.BufferDiscarding
	}

	packageSize, err := utils.UnmarshalUint32(packageSizeMarshaled)
	if err != nil {
		return
	}

	var offset uint32 = 0
	data = make([]byte, packageSize, packageSize)
	for {
		bytesReceived, err := reader.Read(data[offset:])
		if err != nil {
			return nil, err
		}

		offset += uint32(bytesReceived)
		if offset == packageSize {
			return data, nil
		}
	}
}

func (r *Receiver) parseAndRouteData(data []byte) (err error) {

	processRequest := func(request requests.Request) (err error) {
		r.log().WithFields(log.Fields{
			"Type": reflect.TypeOf(request).String(),
		}).Debug("Received")

		err = request.UnmarshalBinary(data[1:])
		if err != nil {
			return
		}

		select {
		case r.Requests <- request:
		default:
			// WARN: do not return error!
			// Otherwise the connection would be closed.
			r.log().WithFields(log.Fields{
				"Type": reflect.TypeOf(r).String(),
			}).Error("Can't transfer r.Requests")
		}

		return
	}

	processResponse := func(response responses.Response, customHandler func(responses.Response)) (err error) {
		r.log().WithFields(log.Fields{
			"Type": reflect.TypeOf(response).String(),
		}).Debug("Received")

		err = response.UnmarshalBinary(data[1:])
		if err != nil {
			return
		}

		if customHandler != nil {
			customHandler(response)
		}

		select {
		case r.Responses <- response:
		default:
			err = errors.ChannelTransferringFailed
		}

		return
	}

	reader := bytes.NewReader(data)
	dataTypeHeader, err := reader.ReadByte()
	if err != nil {
		return
	}

	switch uint8(dataTypeHeader) {

	// Timer
	case constants.DataTypeRequestTimeFrames:
		return processRequest(&requests.SynchronisationTimeFrames{})

	case constants.DataTypeResponseTimeFrame:
		return processResponse(&responses.TimeFrame{}, func(response responses.Response) {
			// Time frame response MUST be extended with time of receiving.
			// It would be used further for time offset corrections.
			response.(*responses.TimeFrame).Received = time.Now()
		})

	// Pools instances
	case constants.DataTypeRequestClaimBroadcast:
		return processRequest(&requests.PoolInstanceBroadcast{})

	case constants.DataTypeRequestTSLBroadcast:
		return processRequest(&requests.PoolInstanceBroadcast{})

	case constants.DataTypeResponseClaimApprove:
		return processResponse(&responses.ClaimApprove{}, nil)

	case constants.DataTypeResponseTSLApprove:
		return processResponse(&responses.TSLApprove{}, nil)

	// Block producer
	case constants.DataTypeRequestDigestBroadcast:
		return processRequest(&requests.CandidateDigestBroadcast{})

	case constants.DataTypeResponseDigestApprove:
		return processResponse(&responses.CandidateDigestApprove{}, nil)

	case constants.DataTypeRequestBlockSignaturesBroadcast:
		return processRequest(&requests.BlockSignaturesBroadcast{})

	case constants.DataTypeRequestChainTop:
		return processRequest(&requests.ChainTop{})

	case constants.DataTypeResponseChainTop:
		return processResponse(&responses.ChainTop{}, nil)

	default:
		return errors.UnexpectedDataType
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

func (r *Receiver) logIngress(bytesReceived int, conn net.Conn) {
	r.log().WithFields(log.Fields{
		"Bytes":     bytesReceived,
		"Addressee": conn.RemoteAddr(),
	}).Debug("[TX <=]")
}

func (r *Receiver) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Network/Observers/Communicator"})
}
