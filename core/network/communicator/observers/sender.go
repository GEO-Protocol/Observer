package observers

import (
	"fmt"
	errors2 "geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/observers/constants"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/settings"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"net"
	"reflect"
	"time"
)

// todo: move this to constants errors
var (
	ErrEmptyData                     = utils.Error("sender", "invalid data set")
	ErrCycleDataSending              = utils.Error("sender", "attempt to send the data to itself prevented")
	ErrPartialWriteOccurred          = utils.Error("sender", "partial write occurred")
	ErrInvalidSendingAttempt         = utils.Error("sender", "invalid data sending attempt")
	ErrObserverConnectionRefused     = utils.Error("sender", "can't connect to remote observer")
	ErrInvalidObserversConfiguration = utils.Error("sender", "can't connect to remote observer")
	ErrInvalidObserverIndex          = utils.Error("sender", "invalid observer index")
)

// todo: move observers list related logic to the sender
//       (response.observersList)
type Sender struct {
	OutgoingRequests  chan requests.Request
	OutgoingResponses chan responses.Response
	IncomingEvents    chan interface{}

	settings    *settings.Settings
	reporter    *external.Reporter
	connections *ConnectionsMap
}

func NewSender(conf *settings.Settings, observersConfReporter *external.Reporter) *Sender {
	return &Sender{
		OutgoingRequests:  make(chan requests.Request, 16),
		OutgoingResponses: make(chan responses.Response, 16),
		IncomingEvents:    make(chan interface{}, 1),

		settings:    conf,
		reporter:    observersConfReporter,
		connections: NewConnectionsMap(time.Minute * 10),
	}
}

func (s *Sender) Run(host string, port uint16, errors chan<- error) {
	// Report Ok
	errors <- nil
	s.log().Info("Started")

	s.waitAndSendInfo(errors)
}

func (s *Sender) waitAndSendInfo(errors chan<- error) {

	processSending := func() {
		select {
		case request := <-s.OutgoingRequests:
			s.processRequestSending(request, errors)

		case response := <-s.OutgoingResponses:
			s.processResponseSending(response, errors)

		case event := <-s.IncomingEvents:
			s.processIncomingEvent(event, errors)
		}
	}

	for {
		processSending()
	}
}

// todo: remove global errors flow
func (s *Sender) processRequestSending(request requests.Request, errors chan<- error) {

	if request == nil {
		errors <- errors2.NilParameter
		return
	}

	// Mark request with observer's number,
	// so the responder would know to which observer to respond.
	currentObserverIndex, err := s.reporter.GetCurrentObserverIndex()
	if err != nil {
		return
	}
	request.SetObserverIndex(currentObserverIndex)

	data, err := request.MarshalBinary()
	if err != nil {
		errors <- utils.Wrap(ErrInvalidSendingAttempt, err.Error())
		return
	}

	send := func(streamType []byte, destinationObservers []uint16) {
		s.log().WithFields(log.Fields{
			"Type": reflect.TypeOf(request).String(),
		}).Debug("Enqueued")

		data = markAs(data, streamType)
		s.sendDataToObservers(data, destinationObservers, errors)
	}

	// todo: add positional number to the data

	switch request.(type) {
	case *requests.SynchronisationTimeFrames:
		send(constants.StreamTypeRequestTimeFrames, nil)

	case *requests.PoolInstanceBroadcast:
		switch request.(*requests.PoolInstanceBroadcast).Instance.(type) {
		case *geo.TransactionSignaturesList:
			send(
				constants.StreamTypeRequestTSLBroadcast,
				request.(*requests.PoolInstanceBroadcast).DestinationObservers())

		case *geo.Claim:
			send(
				constants.StreamTypeRequestClaimBroadcast,
				request.(*requests.PoolInstanceBroadcast).DestinationObservers())

		default:
			// todo: report error here
			s.log().Debug("unexpected broadcast request occurred")
		}

	case *requests.CandidateDigestBroadcast:
		send(constants.StreamTypeRequestDigestBroadcast, nil)

	case *requests.BlockSignaturesBroadcast:
		send(constants.StreamTypeRequestBlockSignaturesBroadcast, nil)

	default:
		errors <- utils.Error(
			"observers sender",
			"unexpected request type occurred")
		s.log().Debug("Unexpected request type occurred")
	}
}

func (s *Sender) processResponseSending(response responses.Response, errors chan<- error) {
	if response == nil {
		errors <- errors2.NilParameter
		return
	}

	data, err := response.MarshalBinary()
	if err != nil {
		errors <- utils.Wrap(ErrInvalidSendingAttempt, err.Error())
		return
	}

	send := func(streamType []byte, destinationObservers []uint16) {
		s.log().WithFields(log.Fields{
			"Type": reflect.TypeOf(response).String(),
		}).Debug("Enqueued")

		data = markAs(data, streamType)
		s.sendDataToObservers(data, destinationObservers, errors)
	}

	switch response.(type) {
	case *responses.TimeFrame:
		send(
			constants.StreamTypeResponseTimeFrame,
			[]uint16{response.Request().ObserverIndex()})

	case *responses.ClaimApprove:
		send(
			constants.StreamTypeResponseClaimApprove,
			[]uint16{response.Request().ObserverIndex()})

	case *responses.TSLApprove:
		send(
			constants.StreamTypeResponseTSLApprove,
			[]uint16{response.Request().ObserverIndex()})

	case *responses.CandidateDigestApprove:
		send(
			constants.StreamTypeResponseDigestApprove,
			[]uint16{response.Request().ObserverIndex()})

	default:
		errors <- utils.Error(
			"observers sender",
			"unexpected response type occurred")
		s.log().Debug("Unexpected response type occurred")
	}
}

func (s *Sender) processIncomingEvent(event interface{}, errors chan<- error) {
	switch event.(type) {
	case *EventConnectionClosed:
		{
			s.connections.DeleteByRemoteHost(event.(*EventConnectionClosed).RemoteHost)
		}
	}
}

// Note: several errors might occur during data sending.
// There are cases when sending should not be cancelled due to an error,
// but the error itself must be reported.
// For this cases, errors channel is propagated to the method.
func (s *Sender) sendDataToObservers(data []byte, observersIndexes []uint16, errors chan<- error) {
	if len(data) == 0 {
		errors <- ErrEmptyData
		return
	}

	observersConf, err := s.reporter.GetCurrentConfiguration()
	if err != nil {
		errors <- ErrInvalidObserversConfiguration
		return
	}

	if observersIndexes == nil {
		// Bytes must be sent to all observers.
		for _, observer := range observersConf.Observers {

			// If not debug - prevent sending the data to itself.
			// In debug mode it might be useful to send blocks to itself,
			// to test whole network cycle in one executable process.
			if observer.Host == s.settings.Observers.Network.Host {
				if observer.Port == s.settings.Observers.Network.Port {
					continue
				}
			}

			err := s.sendDataToObserver(observer, data)
			if err != nil {
				errors <- err
				continue
			}
		}

	} else {
		for _, index := range observersIndexes {
			if len(observersConf.Observers) < int(index+1) {
				errors <- ErrInvalidObserverIndex
				continue
			}

			observer := observersConf.Observers[index]
			err := s.sendDataToObserver(observer, data)
			if err != nil {
				errors <- err
				continue
			}
		}
	}
}

func (s *Sender) sendDataToObserver(observer *external.Observer, data []byte) (err error) {
	send := func(conn *ConnectionWrapper, data []byte) (err error) {
		// Writing data total size.
		dataSize := len(data)
		dataSizeMarshaled := utils.MarshalUint32(uint32(dataSize))
		bytesWritten, err := conn.Writer.Write(dataSizeMarshaled)
		if err != nil {
			return
		}
		if bytesWritten != len(dataSizeMarshaled) {
			err = ErrPartialWriteOccurred
			return
		}

		// todo: split the data to chunks of 2-4Kb in size
		// todo: send them sequentially.

		// Writing data.
		//conn.Connection.(net.TCPConn).SetNoDelay(true)
		bytesWritten, err = conn.Writer.Write(data)
		if err != nil {
			return
		}

		// todo: resend the rest (enhancement).
		if bytesWritten != len(data) {
			err = ErrPartialWriteOccurred
			return
		}

		err = conn.Writer.Flush()
		if err != nil {
			return
		}

		s.logEgress(len(data), conn.Connection)
		return nil
	}

	if len(data) == 0 {
		return ErrEmptyData
	}

	//if !s.settings.Debug {
	// If not debug - prevent sending the data to itself.
	// In debug mode it might be useful to send blocks to itself,
	// to test whole network cycle in one executable process.
	if observer.Host == s.settings.Observers.Network.Host {
		if observer.Port == s.settings.Observers.Network.Port {
			return ErrCycleDataSending
		}
	}

	conn, err := s.connections.Get(observer)
	if err != nil {

		switch err {
		case ErrNoObserver:
			{
				// In case if there is no connection to observer - it should be created.
				conn, err = s.connectToObserver(observer)
				if err != nil {
					return
				}
			}

		default:
			return
		}
	}

	err = send(conn, data)
	if err != nil {
		conn, err = s.connectToObserver(observer)
		if err != nil {
			return
		}

		err = send(conn, data)
	}

	return
}

func (s *Sender) connectToObserver(o *external.Observer) (connection *ConnectionWrapper, err error) {
	conn, err := net.Dial("tcp", fmt.Sprint(o.Host, ":", o.Port))
	if err != nil {
		// No connection is possible to some of observers.
		// Connection error would be reported, but it would not contain any connection details,
		// so it would be very difficult to know which o can't be reached.
		//
		// To make it possible - some additional log information is printed here.

		s.log().WithFields(log.Fields{
			"Host": o.Host,
			"Port": o.Port,
		}).Warn("Remote obs. connection refused")

		return nil, ErrObserverConnectionRefused
	}

	if s.settings.Debug {
		s.log().WithFields(log.Fields{
			"Host": o.Host,
			"Port": o.Port,
		}).Info("Connected to remote observer.")
	}

	s.connections.Set(o, conn)
	return s.connections.Get(o)
}

func (s *Sender) logEgress(bytesSent int, conn net.Conn) {
	s.log().WithFields(log.Fields{
		"Bytes":     bytesSent,
		"Addressee": conn.RemoteAddr(),
	}).Debug("[TX =>]")
}

func (s *Sender) log() *log.Entry {
	return log.WithFields(log.Fields{"prefix": "Network/Observers/Sender"})
}

// markAs prefixes "data" with "prefix" that specifies type of data,
// so it might de decoded on the receiver's side.
func markAs(data, prefix []byte) []byte {
	return append(prefix, data...)
}

// allObservers is a syntax sugar for marking message/request
// addressed to all observers from current configuration.
func allObservers() []uint16 {
	return nil
}
