package geo

import (
	"bufio"
	"encoding"
	"fmt"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/network/communicator/geo/api/v0"
	geoRequests "geo-observers-blockchain/core/network/communicator/geo/api/v0/common"
	"geo-observers-blockchain/core/network/communicator/observers/constants"
	"geo-observers-blockchain/core/settings"
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

// Communicator is used to receive requests from GEO Nodes and transfer relevant responses backward.
// Uses TCP protocol for communication.
// handles each one request in it's own goroutine.
type Communicator struct {
	Requests chan geoRequests.Request

	errorsFlow chan<- error
}

// Communicator creates new instance of Communicator
// with default requests queue set to 256 slots.
func New() *Communicator {
	return &Communicator{
		Requests: make(chan geoRequests.Request, 256),
	}
}

// Run launches communicator and communication flow.
// ToDo: Replace `errorsFlow chan<- error` by `errorsFlow chan<- errors.E`
func (c *Communicator) Run(globalErrorFlow chan<- error) {
	c.errorsFlow = globalErrorFlow

	netInterface := fmt.Sprint(settings.Conf.Nodes.Network.Host, ":", settings.Conf.Nodes.Network.Port)
	listener, err := net.Listen("tcp", netInterface)
	if err != nil {
		c.errorsFlow <- err
		return
	}

	//noinspection GoUnhandledErrorResult
	defer listener.Close()

	c.log().WithFields(log.Fields{
		"Host": settings.Conf.Nodes.Network.Host,
		"Port": settings.Conf.Nodes.Network.Port,
	}).Info("Started")

	for {
		conn, err := listener.Accept()
		if err != nil {
			// todo: throw fatal error
			c.errorsFlow <- err
			return
		}

		go c.handleConnection(conn)
	}
}

// handleConnection tries to parse incoming data, create request and send it for the further processing.
// Drops the connection with corresponding error in case of failure.
// Closes connection after each request processed.
func (c *Communicator) handleConnection(conn net.Conn) {
	defer conn.Close()

	data, err := c.receiveData(conn)
	if err != nil {
		c.respondError(conn, constants.DataTypeInvalidRequest)
		return
	}

	request, e := v0.ParseRequest(data)
	if e != nil {
		c.respondError(conn, constants.DataTypeInvalidRequest)
		return
	}

	c.handleRequest(conn, request)
}

// handleRequest sends request for further dispatching by the core.
// Reports error in global errors flow in case if core is busy and request can't be transferred via channel.
func (c *Communicator) handleRequest(conn net.Conn, request geoRequests.Request) {
	select {
	case c.Requests <- request:
		if settings.OutputNetworkGEOCommunicatorDebug {
			c.log().WithField("Type", reflect.TypeOf(request).String()).Debug("Transferred to core")
		}
		c.handleResponseIfAny(conn, request)

	default:
		c.errorsFlow <- errors.ChannelTransferringFailed
		c.respondError(conn, constants.DataTypeRequestRejected)
	}
}

// handleResponseIfAny checks if request assumes response and if so - tries to send it to the client.
// ToDo: Handle case when claim or TSL wasn't accepted for any reason.
//       Send appropriate response to the client.
func (c *Communicator) handleResponseIfAny(conn net.Conn, request geoRequests.Request) {
	processResponseSending := func(response encoding.BinaryMarshaler) {
		binaryData, err := response.MarshalBinary()
		if err != nil {
			c.errorsFlow <- err
			return
		}

		e := c.sendData(conn, binaryData)
		if e != nil {
			c.errorsFlow <- e.Error()
			return
		}
	}

	processError := func(err error) {
		c.errorsFlow <- err
		c.respondError(conn, constants.DataTypeInternalError)
	}

	if request.ResponseChannel() == nil {
		// Request does not assumes any response.
		c.sendCode(conn, constants.DataTypeRequestAccepted)
		return
	}

	select {
	case response := <-request.ResponseChannel():
		if settings.OutputNetworkGEOCommunicatorDebug {
			c.log().WithField("Type", reflect.TypeOf(response).String()).Debug("Transferred to core")
		}
		processResponseSending(response)

	case err := <-request.ErrorsChannel():
		processError(err)

	case <-time.After(time.Second * 10):
		processError(errors.NoResponseReceived)
	}
}

// receiveData fetches data from the socket and tires to combine it into a message.
func (c *Communicator) receiveData(conn net.Conn) (data []byte, e errors.E) {
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
			c.logIngress(len(data), conn)
			return data, nil
		}
	}
}

func (c *Communicator) sendData(conn net.Conn, data []byte) (e errors.E) {
	dataSize := len(data)
	dataSizeBinary := utils.MarshalUint32(uint32(dataSize))
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

	c.logEgress(totalBytesSent, conn)
	return
}

func (c *Communicator) respondError(conn net.Conn, code int) {
	_, _ = conn.Write([]byte{0, 0, 0, 1, byte(code)})
	c.logEgress(5, conn)
}

func (c *Communicator) sendCode(conn net.Conn, code int) {
	_, _ = conn.Write([]byte{0, 0, 0, 1, byte(code)})
	c.logEgress(5, conn)
}

func (c *Communicator) logIngress(bytesReceived int, conn net.Conn) {
	c.log().Debug("[TX<=] ", bytesReceived, "B, ", conn.RemoteAddr())
}

func (c *Communicator) logEgress(bytesSent int, conn net.Conn) {
	c.log().Debug("[TX=>] ", bytesSent, "B, ", conn.RemoteAddr())
}

func (c *Communicator) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Network/GEO"})
}
