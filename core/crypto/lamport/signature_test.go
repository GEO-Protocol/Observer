package lamport

import (
	"bytes"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
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

func TestSignatures_Add_NilParameter(t *testing.T) {
	signatures := &Signatures{}
	err := signatures.Add(nil)
	if err != common.ErrNilParameter {
		t.Fatal()
	}
}

func TestSignatures_Add_Max(t *testing.T) {
	signatures := &Signatures{}
	for i := 0; i < SignaturesMaxCount; i++ {
		err := signatures.Add(&Signature{})
		if err != nil {
			t.Fatal()
		}
	}

	err := signatures.Add(&Signature{})
	if err != common.ErrMaxCountReached {
		t.Fatal("signatures list must restrict max count of elements")
	}
}

func TestSignatures_Count(t *testing.T) {
	signatures := &Signatures{}
	if signatures.Count() != 0 {
		t.Fatal()
	}

	for i := 0; i < SignaturesMaxCount; i++ {
		_ = signatures.Add(&Signature{})
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
	sig := &Signature{}
	_, err := rand.Read(sig.Bytes[:])
	if err != nil {
		t.Fatal()
	}

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

	signaturesDataIsEqual := bytes.Compare(sig.Bytes[:], restoredSignatures.At[0].Bytes[:]) == 0
	if !signaturesDataIsEqual {
		t.Fatal()
	}
}

// Creates signatures list with maximum possible keys included.
// Serializes it, deserializes it back and then checks data equality.
func TestSignatures_MarshallBinary_MaxElementsCount(t *testing.T) {

	// Reference data initialisation.
	signatures := &Signatures{}
	for j := 0; j < SignaturesMaxCount; j++ {
		sig := &Signature{}
		_, err := rand.Read(sig.Bytes[:])
		if err != nil {
			t.Fatal()
		}

		err = signatures.Add(sig)
		if err != nil {
			t.Fatal()
		}
	}

	// Marshalling.
	binary, _ := signatures.MarshalBinary()

	restoredSignatures := &Signatures{}
	_ = restoredSignatures.UnmarshalBinary(binary)

	// Checks
	if restoredSignatures.Count() != SignaturesMaxCount {
		t.Fatal()
	}
	if restoredSignatures.Count() != signatures.Count() {
		t.Fatal()
	}

	for i, restoredSig := range restoredSignatures.At {
		pubKeysNAreEqual := bytes.Compare(restoredSig.Bytes[:], signatures.At[i].Bytes[:]) == 0
		if !pubKeysNAreEqual {
			t.Fatal()
		}
	}
}
