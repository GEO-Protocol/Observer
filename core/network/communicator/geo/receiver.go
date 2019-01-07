package geo

import (
	"bufio"
	"bytes"
	"fmt"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/geo"
	"geo-observers-blockchain/core/utils"
	log "github.com/sirupsen/logrus"
	"golang.org/x/crypto/blake2b"
	"net"
)

const (
	// Traffic status codes
	TransferStatusOK    = 0
	TransferStatusError = 1
)

type Receiver struct {
	IncomingClaims chan *geo.Claim
	IncomingTSLs   chan *geo.TransactionSignaturesList
}

func NewReceiver() *Receiver {
	const channelBufferSize = 1

	return &Receiver{
		IncomingClaims: make(chan *geo.Claim, channelBufferSize),
		IncomingTSLs:   make(chan *geo.TransactionSignaturesList, channelBufferSize),
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
	defer conn.Close()

	handleTransferError := func(err error) {
		_, _ = conn.Write([]byte{TransferStatusError})
		r.logEgress(1, conn)
		globalErrorsFlow <- err
	}

	handleTransferSuccess := func() {
		_, _ = conn.Write([]byte{TransferStatusOK})
		r.logEgress(1, conn)
	}

	reader := bufio.NewReader(conn)

	message, err := r.receiveData(reader)
	if err != nil {
		handleTransferError(err)
		return
	}
	r.logIngress(len(message), conn)

	hash := message[:blake2b.Size256]
	data := message[blake2b.Size256:]
	if r.checkIntegrity(hash, data) {
		handleTransferError(errors.HashIntegrityCheckFailed)
		return
	}

	err = r.parseAndRouteData(message)
	if err != nil {
		handleTransferError(err)
		return
	}

	handleTransferSuccess()
}

func (r *Receiver) receiveData(reader *bufio.Reader) (data []byte, err error) {
	packageSizeMarshaled := make([]byte, common.Uint32ByteSize, common.Uint32ByteSize)
	bytesRead, err := reader.Read(packageSizeMarshaled)
	if err != nil {
		return
	}
	if bytesRead != common.Uint32ByteSize {
		return nil, errors.BufferDiscarding
	}

	messageSize, err := utils.UnmarshalUint32(packageSizeMarshaled)
	if err != nil {
		return
	}

	var offset uint32 = 0
	data = make([]byte, messageSize, messageSize)
	for {
		bytesReceived, err := reader.Read(data[offset:])
		if err != nil {
			return nil, err
		}

		offset += uint32(bytesReceived)
		if offset == messageSize {
			return data[common.Uint32ByteSize:], nil
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
	case DataTypeTransactionsSignaturesList:
		{
			tsl := &geo.TransactionSignaturesList{}
			err = tsl.UnmarshalBinary(data[1:])
			if err != nil {
				return
			}

			select {
			case r.IncomingTSLs <- tsl:
				{
					return
				}

			default:
				return errors.ChannelTransferringFailed
			}
		}

	case DataTypeClaim:
		{
			claim := &geo.Claim{}
			err = claim.UnmarshalBinary(data[1:])
			if err != nil {
				return
			}

			select {
			case r.IncomingClaims <- claim:
				{
					return
				}

			default:
				return errors.ChannelTransferringFailed
			}
		}

	default:
		return errors.UnexpectedDataType
	}
}

// todo: fix blake2b AVX/AVX2 usage.
//       At this moment, (golang 1.11) internal x/blake2b library patching is needed.
//       More details: https://github.com/golang/go/issues/25098
func (r *Receiver) checkIntegrity(hash, data []byte) bool {
	h := blake2b.Sum256(data)
	return bytes.Compare(hash, h[:]) == 0
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
