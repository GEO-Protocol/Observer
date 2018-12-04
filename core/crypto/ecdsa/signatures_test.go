package ecdsa

import (
	"bytes"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"math/big"
	"math/rand"
	"testing"
)

func TestSignature_UnmarshalBinary_TooShortData(t *testing.T) {
	sig := &Signature{}
	err := sig.UnmarshalBinary([]byte{0, 0, 0, 0, 0, 0})
	if err != common.ErrInvalidDataFormat {
		t.Fatal()
	}
}

// Note:
// MarshallBinary() and UnmarshalBinary methods are checked further.

// --------------------------------------------------------------------------------------------------------------------
func TestSignatures_Add_Max(t *testing.T) {
	signatures := &Signatures{}
	for i := 0; i < SIGNATURES_MAX_COUNT; i++ {
		err := signatures.Add(Signature{})
		if err != nil {
			t.Fatal()
		}
	}

	err := signatures.Add(Signature{})
	if err != common.ErrMaxCountReached {
		t.Fatal("signatures list must restrict max count of elements")
	}
}

func TestSignatures_Count(t *testing.T) {
	signatures := &Signatures{}
	if signatures.Count() != 0 {
		t.Fatal()
	}

	for i := 0; i < SIGNATURES_MAX_COUNT; i++ {
		_ = signatures.Add(Signature{})
		if signatures.Count() != uint16(i+1) {
			t.Fatal()
		}
	}
}

// Creates empty signatures list.
// Serializes it, deserializes it back and then checks data equality.
func TestSignatures_MarshallBinary_NoData(t *testing.T) {
	signatures := &Signatures{}
	restoredSignatures := &Signatures{}

	binary, _ := signatures.MarshalBinary()
	_ = restoredSignatures.UnmarshalBinary(binary)

	// Empty serialised signatures must be 2 bytes long.
	// (total count if signatures serializes as uint16)
	if len(binary) != types.Uint16ByteSize {
		t.Fatal()
	}

	ReferenceSerializedData := []byte{0, 0}
	if bytes.Compare(binary, ReferenceSerializedData) != 0 {
		t.Fatal()
	}

	if signatures.Count() != restoredSignatures.Count() {
		t.Fatal()
	}
}

// Creates signatures list with only one key included.
// Serializes it, deserializes it back and then checks data equality.
func TestSignatures_MarshallBinary_OnePubKey(t *testing.T) {

	// Reference data initialisation.
	sig := Signature{}
	sig.S = big.NewInt(12345678)
	sig.R = big.NewInt(12345678)

	signatures := &Signatures{}
	_ = signatures.Add(sig)

	// Marshalling.
	binary, _ := signatures.MarshalBinary()

	restoredSignatures := &Signatures{}
	_ = restoredSignatures.UnmarshalBinary(binary)

	// Checks
	if signatures.Count() != restoredSignatures.Count() {
		t.Fatal()
	}

	SignaturesDataIsEqual :=
		sig.S.Cmp(restoredSignatures.At[0].S) == 0 &&
			sig.R.Cmp(restoredSignatures.At[0].R) == 0

	if !SignaturesDataIsEqual {
		t.Fatal()
	}
}

// Creates signatures list with maximum possible keys included.
// Serializes it, deserializes it back and then checks data equality.
func TestSignatures_MarshallBinary_MaxElementsCount(t *testing.T) {

	// Reference data initialisation.
	signatures := &Signatures{}
	for j := 0; j < SIGNATURES_MAX_COUNT; j++ {
		sig := Signature{}
		sig.S = big.NewInt(rand.Int63())
		sig.R = big.NewInt(rand.Int63())

		err := signatures.Add(sig)
		if err != nil {
			t.Fatal()
		}
	}

	// Marshalling.
	binary, _ := signatures.MarshalBinary()

	restoredSignatures := &Signatures{}
	_ = restoredSignatures.UnmarshalBinary(binary)

	// Checks
	if restoredSignatures.Count() != SIGNATURES_MAX_COUNT {
		t.Fatal()
	}
	if restoredSignatures.Count() != signatures.Count() {
		t.Fatal()
	}

	for i, restoredSig := range restoredSignatures.At {
		SignaturesDataIsEqual :=
			restoredSig.S.Cmp(signatures.At[i].S) == 0 &&
				restoredSig.R.Cmp(signatures.At[i].R) == 0

		if !SignaturesDataIsEqual {
			t.Fatal()
		}
	}
}
