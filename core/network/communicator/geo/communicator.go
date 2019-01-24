package geo

import (
	"bufio"
	"encoding"
	"fmt"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0"
	geoRequests "geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"net"
	"reflect"
	"time"
)

const (
	MaxMessageSize = 1024 * 1024 * 32 // 32 MB
	MinMessageSize = 1
)

type Communicator struct {
	Requests chan geoRequests.Request
}

func New() *Communicator {
	return &Communicator{
		Requests: make(chan geoRequests.Request, 256),
	}
}

// todo: replace `globalErrorsFlow chan<- error` by `globalErrorsFlow chan<- errors.E`
func (r *Communicator) Run(host string, port uint16, globalErrorsFlow chan<- error) {
	listener, err := net.Listen("tcp", fmt.Sprint(host, ":", port))
	if err != nil {
		globalErrorsFlow <- err
		return
	}

	//noinspection GoUnhandledErrorResult
	defer listener.Close()

	// Inform outer scope that initialisation was performed well
	// and no errors has been occurred.
	globalErrorsFlow <- nil

	r.log().WithFields(log.Fields{
		"Host": host,
		"Port": port,
	}).Info("Started")

	for {
		conn, err := listener.Accept()
		if err != nil {
			// todo: throw fatal error
			globalErrorsFlow <- err
			return
		}

		go r.handleConnection(conn, globalErrorsFlow)
	}
}

func (r *Communicator) handleConnection(conn net.Conn, globalErrorsFlow chan<- error) {
	processError := func(err errors.E) {
		conn.Close()
	}

	message, err := r.receiveData(conn)
	if err != nil {
		processError(err)
		return
	}

	request, e := v0.ParseRequest(message)
	if e != nil {
		processError(e)
		return
	}

	go r.handleRequest(conn, request, globalErrorsFlow)
}

func (r *Communicator) handleRequest(conn net.Conn, request geoRequests.Request, globalErrorsFlow chan<- error) {
	defer conn.Close()

	select {
	case r.Requests <- request:
		r.handleResponseIfAny(conn, request, globalErrorsFlow)
		r.log().WithFields(log.Fields{
			"Type": reflect.TypeOf(request).String(),
		}).Debug("Transferred to core")

	default:
		globalErrorsFlow <- errors.ChannelTransferringFailed
	}
}

func (r *Communicator) handleResponseIfAny(conn net.Conn, request geoRequests.Request, globalErrorsFlow chan<- error) {
	processResponseSending := func(response encoding.BinaryMarshaler) {
		binaryData, err := response.MarshalBinary()
		if err != nil {
			globalErrorsFlow <- err
			return
		}

		e := r.sendData(conn, binaryData)
		if e != nil {
			globalErrorsFlow <- e.Error()
			return
		}
	}

	if request.ResponseChannel() == nil {
		// Request does not assumes any response.
		return
	}

	select {
	case response := <-request.ResponseChannel():
		r.log().WithFields(log.Fields{
			"Type": reflect.TypeOf(response).String(),
		}).Debug("Enqueued for sending")
		processResponseSending(response)

	case <-time.After(time.Second * 2):
		globalErrorsFlow <- errors.NoResponseReceived
	}
}

func (r *Communicator) receiveData(conn net.Conn) (data []byte, e errors.E) {
	reader := bufio.NewReader(conn)

	messageSizeBinary := []byte{0, 0, 0, 0}
	bytesRead, err := reader.Read(messageSizeBinary)
	if err != nil {
		e = errors.AppendStackTrace(err)
		return
	}
	if bytesRead != common.Uint32ByteSize {
		e = errors.AppendStackTrace(errors.InvalidDataFormat)
		return
	}
	_, _ = reader.Discard(4)

	messageSize, err := utils.UnmarshalUint32(messageSizeBinary)
	if err != nil {
		e = errors.AppendStackTrace(errors.InvalidDataFormat)
		return
	}
	if messageSize > MaxMessageSize || messageSize < MinMessageSize {
		e = errors.AppendStackTrace(errors.InvalidDataFormat)
		return
	}

	var offset uint32 = 0
	data = make([]byte, messageSize, messageSize)
	for {
		bytesReceived, err := reader.Read(data[offset:])
		if err != nil {
			return nil, errors.AppendStackTrace(err)
		}
		if bytesReceived == 0 {
			return nil, errors.AppendStackTrace(errors.NoData)
		}

		offset += uint32(bytesReceived)
		if offset == messageSize {
			r.logIngress(len(data), conn)
			return data, nil
		}
	}
}

func (r *Communicator) sendData(conn net.Conn, data []byte) (e errors.E) {
	dataSize := len(data)
	dataSizeBinary := utils.MarshalUint64(uint64(dataSize))
	data = append(dataSizeBinary, data...)

	totalBytesSent := 0
	for {
		if totalBytesSent == len(data) {
			break
		}

		bytesWritten, err := conn.Write(data[totalBytesSent:])
		if err != nil {
			return errors.AppendStackTrace(err)
		}

		totalBytesSent += bytesWritten
	}

	r.logEgress(totalBytesSent, conn)
	return
}

func (r *Communicator) logIngress(bytesReceived int, conn net.Conn) {
	r.log().Debug("[TX<=] ", bytesReceived, "B, ", conn.RemoteAddr())
}

func (r *Communicator) logEgress(bytesSent int, conn net.Conn) {
	r.log().Debug("[TX=>] ", bytesSent, "B, ", conn.RemoteAddr())
}

func (r *Communicator) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Network/GEO"})
}
