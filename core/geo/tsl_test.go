package geo

import (
	"bytes"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/crypto/lamport"
	"math/rand"
	"testing"
)

func TestTransactionSignaturesList_MarshalBinaryEmptyInternal(t *testing.T) {
	tsl := &TSL{}
	_, err := tsl.MarshalBinary()
	if err != common.ErrNilInternalDataStructure {
		t.Fatal()
	}
}

func TestTransactionSignaturesList_UnmarshalBinary_EmptyData(t *testing.T) {
	tsl := NewTSL()
	err := tsl.UnmarshalBinary([]byte{})
	if err != common.ErrInvalidDataFormat {
		t.Fatal()
	}
}

func TestTransactionSignaturesList_UnmarshalBinary_TooShortData(t *testing.T) {
	tsl := NewTSL()
	err := tsl.UnmarshalBinary([]byte{0, 0, 0, 0, 0, 0})
	if err != common.ErrInvalidDataFormat {
		t.Fatal()
	}
}

// Note:
// MarshallBinary() and UnmarshalBinary methods are checked further.

//--------------------------------------------------------------------------------------------------------------------

func TestTransactionSignaturesLists_Add_NilParameter(t *testing.T) {
	tsls := &TSLs{}
	err := tsls.Add(nil)
	if err != common.ErrNilParameter {
		t.Fatal()
	}
}

func TestTransactionSignaturesLists_Add_Max(t *testing.T) {
	tsls := &TSLs{}
	for i := 0; i < TSLsMaxCount; i++ {
		err := tsls.Add(NewTSL())
		if err != nil {
			t.Fatal()
		}
	}

	err := tsls.Add(NewTSL())
	if err != common.ErrMaxCountReached {
		t.Fatal("tsls list must restrict max count of elements")
	}
}

func TestTransactionSignaturesLists_Count(t *testing.T) {
	tsls := &TSLs{}
	if tsls.Count() != 0 {
		t.Fatal()
	}

	for i := 0; i < TSLsMaxCount; i++ {
		_ = tsls.Add(NewTSL())
		if tsls.Count() != uint16(i+1) {
			t.Fatal()
		}
	}
}

// Creates empty transactions signatures list.
// Serializes it, deserializes it back and then checks data equality.
func TestTransactionSignaturesLists_MarshallBinary_NoData(t *testing.T) {
	tsls := &TSLs{}
	restoredTSLs := &TSLs{}

	binary, _ := tsls.MarshalBinary()
	_ = restoredTSLs.UnmarshalBinary(binary)

	// Empty serialised tsls must be 2 bytes long.
	// (total count if tsls serializes as uint16)
	if len(binary) != types.Uint16ByteSize {
		t.Fatal()
	}

	ReferenceSerializedData := []byte{0, 0}
	if bytes.Compare(binary, ReferenceSerializedData) != 0 {
		t.Fatal()
	}

	if tsls.Count() != restoredTSLs.Count() {
		t.Fatal()
	}
}

// Creates transactions signatures list with only one TSL included.
// Serializes it, deserializes it back and then checks data equality.
func TestTransactionSignaturesLists_MarshallBinary_OneTSL_1024Signatures(t *testing.T) {

	// Reference data initialisation.
	tsl := NewTSL()
	tsl.TxUUID.Bytes = [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 1, 2, 3, 4, 5, 6}
	for i := 0; i < 1024; i++ {
		sig := &lamport.Signature{}
		_, err := rand.Read(sig.Bytes[:])
		if err != nil {
			t.Fatal()
		}

		_ = tsl.Members.Add(sig)
	}

	tsls := &TSLs{}
	_ = tsls.Add(tsl)

	// Marshalling.
	binary, _ := tsls.MarshalBinary()

	restoredTSLs := &TSLs{}
	_ = restoredTSLs.UnmarshalBinary(binary)

	// Checks
	if tsls.Count() != restoredTSLs.Count() {
		t.Fatal()
	}

	// Transaction UUID
	transactionUUIDsAreEqual := bytes.Compare(
		tsls.At[0].TxUUID.Bytes[:],
		restoredTSLs.At[0].TxUUID.Bytes[:]) == 0

	if !transactionUUIDsAreEqual {
		t.Fatal()
	}

	// At count
	pubKeysCountAreEqual := restoredTSLs.At[0].Members.Count() == restoredTSLs.At[0].Members.Count()
	if !pubKeysCountAreEqual {
		t.Fatal()
	}

	// At data
	for i, sig := range tsls.At[0].Members.At {
		restoredSig := restoredTSLs.At[0].Members.At[i]
		sigNIsEqual := bytes.Compare(sig.Bytes[:], restoredSig.Bytes[:]) == 0
		if !sigNIsEqual {
			t.Fatal()
		}
	}
}

// Creates transactions signatures list with maximum possible TSLsHashes included.
// Serializes it, deserializes it back and then checks data equality.
//
// WARN: Consumes more than 182MB of memory.
func TestTransactionSignaturesLists_MarshallBinary_MaxElementsCount(t *testing.T) {

	// Reference data initialisation.
	TSLs := &TSLs{}
	for j := 0; j < TSLsMaxCount; j++ {
		TSL := NewTSL()
		_, err := rand.Read(TSL.TxUUID.Bytes[:])
		if err != nil {
			t.Fatal()
		}

		sig := &lamport.Signature{}
		_, err = rand.Read(sig.Bytes[:])
		if err != nil {
			t.Fatal()
		}
		_ = TSL.Members.Add(sig)

		err = TSLs.Add(TSL)
		if err != nil {
			t.Fatal()
		}
	}

	// Marshalling.
	binary, _ := TSLs.MarshalBinary()

	restoredTSLs := &TSLs{}
	_ = restoredTSLs.UnmarshalBinary(binary)

	// Checks
	if restoredTSLs.Count() != TSLsMaxCount {
		t.Fatal()
	}
	if restoredTSLs.Count() != TSLs.Count() {
		t.Fatal()
	}

	for i, restoredTSL := range restoredTSLs.At {
		transactionUUIDsAreEqual := bytes.Compare(
			restoredTSL.TxUUID.Bytes[:],
			TSLs.At[i].TxUUID.Bytes[:]) == 0

		if !transactionUUIDsAreEqual {
			t.Fatal()
		}

		for j, sig := range restoredTSL.Members.At {
			sigNIsEqual := bytes.Compare(sig.Bytes[:], TSLs.At[i].Members.At[j].Bytes[:]) == 0
			if !sigNIsEqual {
				t.Fatal()
			}
		}
	}
}
