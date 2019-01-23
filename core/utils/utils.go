package utils

import (
	"bytes"
	"encoding/binary"
)

var (
	ErrInvalidData = Error("utils", "invalid data occurred")
)

func MarshalUint16(u uint16) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, u)
	return buf.Bytes()
}

func UnmarshalUint16(data []byte) (u uint16, err error) {
	if len(data) < 2 {
		err = ErrInvalidData
		return
	}

	buf := bytes.NewReader(data)
	err = binary.Read(buf, binary.LittleEndian, &u)
	return
}

func MarshalUint32(u uint32) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, u)
	return buf.Bytes()
}

func UnmarshalUint32(data []byte) (u uint32, err error) {
	if len(data) != 4 {
		err = ErrInvalidData
		return
	}

	buf := bytes.NewReader(data)
	err = binary.Read(buf, binary.LittleEndian, &u)
	return
}

func MarshalUint64(u uint64) []byte {
	buf := new(bytes.Buffer)
	_ = binary.Write(buf, binary.LittleEndian, u)
	return buf.Bytes()
}

func UnmarshalUint64(data []byte) (u uint64, err error) {
	if len(data) != 8 {
		err = ErrInvalidData
		return
	}

	buf := bytes.NewReader(data)
	err = binary.Read(buf, binary.LittleEndian, &u)
	return
}

func ChainByteSlices(slices ...[]byte) []byte {
	totalSize := 0
	for _, slice := range slices {
		totalSize += len(slice)
	}

	container := make([]byte, 0, totalSize)
	for _, slice := range slices {
		container = append(container, slice...)
	}

	return container
}

func CatchPanic(err error) {
	if r := recover(); r != nil {
		switch k := r.(type) {
		case error:
			{
				err = k
			}

		default:
			panic(r)
		}
	}
}

func PanicOnError(err error) {
	if err != nil {
		panic(err)
	}
}
