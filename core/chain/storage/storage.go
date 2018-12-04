package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

var (
	ErrorInvalidOffset       = errors.New("invalid offset of the file")
	ErrorInvalidRecordHeader = errors.New("invalid record header")
	ErrorInvalidWrite        = errors.New("invalid write operation")
	ErrorInvalidRead         = errors.New("invalid read operation")
	ErrorInvalidIndex        = errors.New("invalid index")
	ErrorInactiveStorage     = errors.New("keystore is inactive")
)

const (
	kRecordHeaderSize = 4 // bytes, uint32
)

// todo: tests needed
type AppendOnlyStorage struct {
	file         *os.File
	isActive     bool
	recordsCount uint64

	// Records index.
	// Key - record number.
	// Value - address (offset) of the record in data file.
	index map[uint64]int64
}

func Open(filename string) (*AppendOnlyStorage, error) {
	storage := &AppendOnlyStorage{}

	var err error
	storage.file, err = os.Open(filename)
	if err != nil {
		return nil, err
	}

	err = storage.resetIndex()
	if err != nil {
		return nil, err
	}

	storage.index = make(map[uint64]int64)
	storage.isActive = true
	return storage, nil
}

func (c *AppendOnlyStorage) Append(data []byte) error {

	if !c.isActive {
		return ErrorInactiveStorage
	}

	// Content and header must be written in ACID manner.
	// There is a non-zero probability of partial write of the data
	// in case if it would be written a slice of bytes.
	//
	// To prevent it and provide strong guarantee of data integrity -
	// data is written in 2 phases:
	// 1. Empty header + data;
	// 2. Real header.
	//
	// Empty header is a sequence from four zeroes.
	// In case if keystore would find e record with a 0 size -
	// it will escape this record as broken (stripped away).
	//
	// In case of unsuccessful/partial content write - header would remain zeroed.
	// When such a record would be discovered -
	// keystore would simply replace it's content by newly inserted records.
	//
	// The record itself begins to be valid only after non zero header is written before IT.
	// On most systems writing of 4B is an atomic operation.

	// Ensure writing is done always from the end.
	_, err := c.file.Seek(0, 2)
	if err != nil {
		return err
	}

	// Empty record header.
	bw, err := c.file.Write([]byte{0, 0, 0, 0})
	if err != nil {
		return err
	}
	if bw != kRecordHeaderSize {
		return ErrorInvalidWrite
	}

	// Content.
	recordSize := int32(len(data))

	bw, err = c.file.Write(data)
	if err != nil {
		return err
	}
	if int32(bw) != recordSize {
		return ErrorInvalidWrite
	}

	// Sync to be sure that content is on the disk.
	err = c.file.Sync()
	if err != nil {
		return err
	}

	// Real header.
	lastRecordOffset, err := c.file.Seek(int64(recordSize)*-1, 1)
	if err != nil {
		return err
	}

	err = binary.Write(c.file, binary.LittleEndian, &recordSize)
	if err != nil {
		return err
	}

	// Ensure header is present.
	err = c.file.Sync()
	if err != nil {
		return err
	}

	c.recordsCount += 1
	c.index[c.recordsCount] = lastRecordOffset
	return nil
}

func (c *AppendOnlyStorage) Get(index uint64) (buffer []byte, err error) {
	if !c.isActive {
		return nil, ErrorInactiveStorage
	}

	if index > c.recordsCount {
		return nil, ErrorInvalidIndex
	}

	offset, ok := c.index[index]
	if !ok {
		return nil, ErrorInvalidIndex
	}

	_, err = c.file.Seek(offset, 0)
	if err != nil {
		return nil, err
	}

	var recordSize int32 = 0
	err = binary.Read(c.file, binary.LittleEndian, &recordSize)
	if err != nil {
		return nil, err
	}

	buffer = make([]byte, recordSize)
	k, err := c.file.Read(buffer)
	if int32(k) != recordSize {
		return nil, ErrorInvalidRead
	}

	return buffer, nil
}

func (c *AppendOnlyStorage) Cut(fromIndex uint64) error {
	if !c.isActive {
		return ErrorInactiveStorage
	}

	if fromIndex > c.recordsCount {
		return ErrorInvalidIndex
	}

	offset, ok := c.index[fromIndex]
	if !ok {
		return ErrorInvalidIndex
	}

	err := c.file.Truncate(offset)
	if err != nil {
		return err
	}

	c.recordsCount = 0
	shortenedIndex := make(map[uint64]int64)
	for k, v := range c.index {
		if k < fromIndex {
			shortenedIndex[k] = v
			c.recordsCount += 1
		}
	}

	c.index = shortenedIndex
	return nil
}

func (c *AppendOnlyStorage) Count() (uint64, error) {
	if !c.isActive {
		return 0, ErrorInactiveStorage
	}

	return c.recordsCount, nil
}

func (c *AppendOnlyStorage) Close() error {
	if !c.isActive {
		return ErrorInactiveStorage
	}

	c.isActive = false
	return c.file.Close()
}

func (c *AppendOnlyStorage) resetIndex() error {
	offset, err := c.file.Seek(0, 0)
	if err != nil {
		return err
	}

	if offset != 0 {
		return ErrorInvalidOffset
	}

	recordHeader := make([]byte, kRecordHeaderSize)
	var (
		recordSize  int32 = 0
		currentSeek int64 = 0
	)

	for {
		bytesRead, err := c.file.Read(recordHeader)
		if err != nil {
			if err == io.EOF {
				return nil

			} else {
				return err

			}
		}
		if bytesRead != kRecordHeaderSize {
			return ErrorInvalidRecordHeader
		}

		buf := bytes.NewReader(recordHeader)
		err = binary.Read(buf, binary.LittleEndian, &recordSize)
		if err != nil {
			return err
		}
		if recordSize == 0 {
			// In case if next record size was read well, but it is marked as zero-length -
			// then it means that this records must be recognized as invalid.
			// Empty record header may occur on unsuccessful Append() call.
			// Please, see doc. for Append() for the details.
			return nil
		}

		bytesSeek, err := c.file.Seek(int64(recordSize), 1)
		if err != nil {
			return err
		}

		if bytesSeek != int64(recordSize) {
			return ErrorInvalidOffset
		}

		c.index[c.recordsCount] = currentSeek

		c.recordsCount += 1
		currentSeek += int64(recordSize) + kRecordHeaderSize
	}
}
