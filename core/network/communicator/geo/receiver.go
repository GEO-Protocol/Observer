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

type Receiver struct {
	Requests chan geoRequests.Request
}

func NewReceiver() *Receiver {
	return &Receiver{
		Requests: make(chan geoRequests.Request, 256),
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
			return
		}

		go r.handleConnection(conn, errors)
	}
}

func (r *Receiver) handleConnection(conn net.Conn, globalErrorsFlow chan<- error) {
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

func (r *Receiver) handleRequest(conn net.Conn, request geoRequests.Request, globalErrorsFlow chan<- error) {
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

func (r *Receiver) handleResponseIfAny(conn net.Conn, request geoRequests.Request, globalErrorsFlow chan<- error) {
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

func (r *Receiver) receiveData(conn net.Conn) (data []byte, e errors.E) {
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

	messageSize, err := utils.UnmarshalUint32(messageSizeBinary)
	if err != nil {
		e = errors.AppendStackTrace(errors.InvalidDataFormat)
		return
	}

	_, _ = reader.Discard(4)
	var offset uint32 = 0
	data = make([]byte, messageSize, messageSize)
	for {
		bytesReceived, err := reader.Read(data[offset:])
		if err != nil {
			return nil, errors.AppendStackTrace(err)
		}

		offset += uint32(bytesReceived)
		if offset == messageSize {
			r.logIngress(len(data), conn)
			return data, nil
		}
	}
}

func (r *Receiver) sendData(conn net.Conn, data []byte) (e errors.E) {
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

func (r *Receiver) logIngress(bytesReceived int, conn net.Conn) {
	r.log().Debug("[TX<=] ", bytesReceived, "B, ", conn.RemoteAddr())
}

func (r *Receiver) logEgress(bytesSent int, conn net.Conn) {
	r.log().Debug("[TX=>] ", bytesSent, "B, ", conn.RemoteAddr())
}

func (r *Receiver) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Network/GEO"})
}
