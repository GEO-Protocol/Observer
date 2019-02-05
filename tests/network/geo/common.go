package geo

import (
	"bufio"
	"encoding"
	"fmt"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/utils"
	"geo-observers-blockchain/tests"
	"net"
	"testing"
	"time"
)

func ConnectToObserver(t *testing.T, observerIndex int) (conn net.Conn) {
	observer := tests.Observers[observerIndex]
	conn, err := net.Dial("tcp", fmt.Sprint(observer.Host, ":", observer.GEOPort))
	if err != nil {
		t.Fatal("could not connect to observer: ", err)
	}

	return
}

func GetResponse(t *testing.T, response encoding.BinaryUnmarshaler, conn net.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(time.Second * 3))
	reader := bufio.NewReader(conn)

	messageSizeBinary := make([]byte, 4, 4)
	bytesRead, err := reader.Read(messageSizeBinary)
	if err != nil {
		t.Fatal(err)
	}

	if bytesRead != common.Uint32ByteSize {
		t.Fatal("Invalid message header received")
	}

	messageSize, err := utils.UnmarshalUint32(messageSizeBinary)
	if err != nil {
		t.Fatal(err)
	}

	var offset uint32 = 0
	data := make([]byte, messageSize, messageSize)
	for {
		bytesReceived, err := reader.Read(data[offset:])
		if err != nil {
			t.Fatal(err)
		}

		offset += uint32(bytesReceived)
		if offset == messageSize {
			err := response.UnmarshalBinary(data)
			if err != nil {
				t.Error()
				return
			}

			return
		}
	}
}

func SendRequest(t *testing.T, request encoding.BinaryMarshaler, conn net.Conn) {
	requestBinary, err := request.MarshalBinary()
	if err != nil {
		t.Error()
	}

	SendData(t, conn, requestBinary)
}

func SendData(t *testing.T, conn net.Conn, data []byte) {
	err := SendDataOrReportError(conn, data)
	if err != nil {
		t.Error(err)
	}
}

func SendDataOrReportError(conn net.Conn, data []byte) (err error) {
	data = append([]byte{0}, data...) // protocol header

	var (
		dataLength       = uint32(len(data))
		dataLengthBinary = utils.MarshalUint32(dataLength)
	)

	_, err = conn.Write(dataLengthBinary)
	if err != nil {
		return
	}

	_, err = conn.Write(data)
	if err != nil {
		return
	}

	return
}
