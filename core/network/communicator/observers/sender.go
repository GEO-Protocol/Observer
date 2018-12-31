package observers

import (
	"fmt"
	"geo-observers-blockchain/core/chain"
	errors2 "geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/network/communicator/observers/constants"
	"geo-observers-blockchain/core/network/communicator/observers/requests"
	"geo-observers-blockchain/core/network/communicator/observers/responses"
	"geo-observers-blockchain/core/network/external"
	"geo-observers-blockchain/core/network/messages"
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

// todo: encrypt traffic between observers
type Sender struct {
	Settings         *settings.Settings
	ObserversConfRep *external.Reporter

	OutgoingProposedBlocks   chan *chain.ProposedBlockData
	OutgoingBlocksSignatures chan *messages.SignatureMessage
	OutgoingRequests         chan requests.Request
	OutgoingResponses        chan responses.Response

	IncomingEvents chan interface{}

	connections *ConnectionsMap
}

func NewSender(conf *settings.Settings, observersConfigurationReporter *external.Reporter) *Sender {
	const kChannelBufferSize = 1

	return &Sender{
		Settings:         conf,
		ObserversConfRep: observersConfigurationReporter,

		OutgoingProposedBlocks:   make(chan *chain.ProposedBlockData, kChannelBufferSize),
		OutgoingBlocksSignatures: make(chan *messages.SignatureMessage, kChannelBufferSize),
		OutgoingRequests:         make(chan requests.Request, kChannelBufferSize),
		OutgoingResponses:        make(chan responses.Response, kChannelBufferSize),

		IncomingEvents: make(chan interface{}, kChannelBufferSize),

		connections: NewConnectionsMap(time.Minute * 10),
	}
}

func (s *Sender) Run(host string, port uint16, errors chan<- error) {
	// Report Ok
	errors <- nil

	s.waitAndSendInfo(errors)
}

func (s *Sender) waitAndSendInfo(errors chan<- error) {

	processSending := func() {
		select {
		case proposedBlock := <-s.OutgoingProposedBlocks:
			s.processBlockProposalSending(proposedBlock, errors)

		case blockSignature := <-s.OutgoingBlocksSignatures:
			s.processBlockSignatureSending(blockSignature, errors)

		case request := <-s.OutgoingRequests:
			s.processRequestSending(request, errors)

		case response := <-s.OutgoingResponses:
			s.processResponseSending(response, errors)

		// Events
		case event := <-s.IncomingEvents:
			s.processIncomingEvent(event, errors)

			// ...
			// other cases

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
	currentObserverNumber, err := s.ObserversConfRep.GetCurrentObserverNumber()
	if err != nil {
		return
	}
	request.SetObserverNumber(currentObserverNumber)
	s.log().Debug(currentObserverNumber)

	data, err := request.MarshalBinary()
	if err != nil {
		errors <- utils.Wrap(ErrInvalidSendingAttempt, err.Error())
		return
	}

	send := func(streamType []byte, destinationObservers []uint16) {
		data = markAs(data, streamType)
		s.sendDataToObservers(data, destinationObservers, errors)
		s.log().Debug(reflect.TypeOf(request).String(), " sent")
	}

	// todo: add positional number to the data

	switch request.(type) {
	case *requests.RequestSynchronisationTimeFrames:
		send(constants.StreamTypeRequestTimeFrames, nil)

	case *requests.RequestPoolInstanceBroadcast:
		switch request.(*requests.RequestPoolInstanceBroadcast).Instance.(type) {
		case *geo.TransactionSignaturesList:
			send(
				constants.StreamTypeRequestTSLBroadcast,
				request.(*requests.RequestPoolInstanceBroadcast).DestinationObservers())

		case *geo.Claim:
			send(
				constants.StreamTypeRequestClaimBroadcast,
				request.(*requests.RequestPoolInstanceBroadcast).DestinationObservers())

		default:
			// todo: report error here
			s.log().Debug("unexpected broadcast request occurred")
		}

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
		data = markAs(data, streamType)
		s.sendDataToObservers(data, destinationObservers, errors)
		s.log().Debug(reflect.TypeOf(response).String(), " sent")
	}

	switch response.(type) {
	case *responses.ResponseTimeFrame:
		send(
			constants.StreamTypeResponseTimeFrame,
			[]uint16{response.Request().ObserverNumber()})

	case *responses.ResponseClaimApprove:
		send(
			constants.StreamTypeResponseClaimApprove,
			[]uint16{response.Request().ObserverNumber()})

	case *responses.ResponseTSLApprove:
		send(
			constants.StreamTypeResponseTSLApprove,
			[]uint16{response.Request().ObserverNumber()})

	default:
		errors <- utils.Error(
			"observers sender",
			"unexpected response type occurred")
		s.log().Debug("Unexpected response type occurred")
	}
}

// todo: consider sending blocks as a stream: claim after claim with on the fly validation on the receivers part
// (defence from the invalid data sending )
func (s *Sender) processBlockProposalSending(block *chain.ProposedBlockData, errors chan<- error) {
	if block == nil {
		errors <- errors2.NilParameter
		return
	}

	data, err := block.MarshalBinary()
	if err != nil {
		errors <- utils.Wrap(ErrInvalidSendingAttempt, err.Error())
		return
	}

	// Mark data as block proposal and send it.
	data = append(constants.StreamTypeBlockProposal, data...)

	// todo: crate function for observers selection
	s.sendDataToObservers(data, nil, errors)
}

func (s *Sender) processBlockSignatureSending(message *messages.SignatureMessage, errors chan<- error) {
	if message == nil {
		errors <- errors2.NilParameter
		return
	}

	data, err := message.MarshalBinary()
	if err != nil {
		errors <- utils.Wrap(ErrInvalidSendingAttempt, err.Error())
		return
	}

	// todo: append block hash to have possibility to collect signatures
	//  for various proposed blocks on the receiver's side.

	// Mark data as block proposal and send it.
	data = append(constants.StreamTypeBlockSignature, data...)
	s.sendDataToObservers(data, []uint16{message.AddresseeObserverIndex}, errors)
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

	observersConf, err := s.ObserversConfRep.GetCurrentConfiguration()
	if err != nil {
		errors <- ErrInvalidObserversConfiguration
		return
	}

	if observersIndexes == nil {
		// Bytes must be sent to all observers.
		for _, observer := range observersConf.Observers {

			//if !s.settings.Debug {
			// If not debug - prevent sending the data to itself.
			// In debug mode it might be useful to send blocks to itself,
			// to test whole network cycle in one executable process.
			if observer.Host == s.Settings.Observers.Network.Host {
				if observer.Port == s.Settings.Observers.Network.Port {
					continue
				}
			}
			//}

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
	if observer.Host == s.Settings.Observers.Network.Host {
		if observer.Port == s.Settings.Observers.Network.Port {
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
		additionalInfo := log.Fields{"host": o.Host, "port": o.Port}
		s.log().WithFields(additionalInfo).Errorln("Can't connect to remote observer.")

		return nil, ErrObserverConnectionRefused
	}

	if s.Settings.Debug {
		additionalInfo := log.Fields{"host": o.Host, "port": o.Port}
		s.log().WithFields(additionalInfo).Info("Connected to remote observer.")
	}

	s.connections.Set(o, conn)
	return s.connections.Get(o)
}

func (s *Sender) logEgress(bytesSent int, conn net.Conn) {
	s.log().Debug("[TX=>] ", bytesSent, "B, ", conn.RemoteAddr())
}

func (s *Sender) log() *log.Entry {
	return log.WithFields(log.Fields{"subsystem": "senderObservers"})
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
