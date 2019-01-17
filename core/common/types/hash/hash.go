package hash

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"geo-observers-blockchain/core/common/errors"
)

const (
	BytesSize = 32
)

// todo: replace by sha256 by blake2b 512
// todo: add project specific salt
// todo: think about custom round-specific salt,
//       to prevent usage or pre generated hash tables to brute the hashes.

type SHA256Container struct {
	Bytes [BytesSize]byte
}

func NewSHA256Container(data []byte) SHA256Container {
	return SHA256Container{
		Bytes: sha256.Sum256(data),
	}
}

func (h *SHA256Container) Compare(other *SHA256Container) bool {
	return bytes.Compare(other.Bytes[:], h.Bytes[:]) == 0
}

func (h *SHA256Container) MarshalBinary() (data []byte, err error) {
	return h.Bytes[:BytesSize], nil
}

func (h *SHA256Container) UnmarshalBinary(data []byte) error {
	if copy(h.Bytes[:], data[:BytesSize]) == BytesSize {
		return nil

	} else {
		return errors.InvalidCopyOperation

	}
}

func (h *SHA256Container) Hex() string {
	return "0x" + hex.EncodeToString(h.Bytes[:])
}
