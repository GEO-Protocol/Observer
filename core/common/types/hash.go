package types

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	HashSize = 32
)

type SHA256Container struct {
	Bytes [HashSize]byte
}

func NewSHA256Container(data []byte) SHA256Container {
	return SHA256Container{
		Bytes: sha256.Sum256(data),
	}
}

func (h *SHA256Container) MarshalBinary() (data []byte, err error) {
	return h.Bytes[:HashSize], nil
}

func (h *SHA256Container) UnmarshalBinary(data []byte) error {
	if copy(h.Bytes[:], data[:HashSize]) == HashSize {
		return nil

	} else {
		return ErrorInvalidCopyOperation

	}
}

func (h *SHA256Container) Hex() string {
	return "0x" + hex.EncodeToString(h.Bytes[:])
}
