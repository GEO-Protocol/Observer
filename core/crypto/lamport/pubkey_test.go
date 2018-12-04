package lamport

import (
	"bytes"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"math/rand"
	"testing"
)

func TestPubKey_UnmarshalBinary_TooShortData(t *testing.T) {
	key := &PubKey{}
	err := key.UnmarshalBinary([]byte{0, 0, 0, 0, 0, 0})
	if err != common.ErrInvalidDataFormat {
		t.Fatal()
	}
}

// Note:
// MarshallBinary() and UnmarshalBinary methods are checked further.

// --------------------------------------------------------------------------------------------------------------------

func TestPubKeys_Add_NilParameter(t *testing.T) {
	keys := &PubKeys{}
	err := keys.Add(nil)
	if err != common.ErrNilParameter {
		t.Fatal()
	}
}

func TestPubKeys_Add_Max(t *testing.T) {
	keys := &PubKeys{}
	for i := 0; i < PubKeysMaxCount; i++ {
		err := keys.Add(&PubKey{})
		if err != nil {
			t.Fatal()
		}
	}

	err := keys.Add(&PubKey{})
	if err != common.ErrMaxCountReached {
		t.Fatal("keys list must restrict max count of elements")
	}
}

func TestPubKeys_Count(t *testing.T) {
	keys := &PubKeys{}
	if keys.Count() != 0 {
		t.Fatal()
	}

	for i := 0; i < PubKeysMaxCount; i++ {
		_ = keys.Add(&PubKey{})
		if keys.Count() != uint16(i+1) {
			t.Fatal()
		}
	}
}

// Creates empty public keys list.
// Serializes it, deserializes it back and then checks data equality.
func TestPubKeys_MarshallBinary_NoData(t *testing.T) {
	keys := &PubKeys{}
	restoredTSLs := &PubKeys{}

	binary, _ := keys.MarshalBinary()
	_ = restoredTSLs.UnmarshalBinary(binary)

	// Empty serialised keys must be 2 bytes long.
	// (total count if keys serializes as uint16)
	if len(binary) != types.Uint16ByteSize {
		t.Fatal()
	}

	ReferenceSerializedData := []byte{0, 0}
	if bytes.Compare(binary, ReferenceSerializedData) != 0 {
		t.Fatal()
	}

	if keys.Count() != restoredTSLs.Count() {
		t.Fatal()
	}
}

// Creates pub keys list with only one key included.
// Serializes it, deserializes it back and then checks data equality.
func TestPubKeys_MarshallBinary_OnePubKey(t *testing.T) {

	// Reference data initialisation.
	key := &PubKey{}
	_, err := rand.Read(key.Bytes[:])
	if err != nil {
		t.Fatal()
	}

	pubKeys := &PubKeys{}
	_ = pubKeys.Add(key)

	// Marshalling.
	binary, _ := pubKeys.MarshalBinary()

	restoredPubKeys := &PubKeys{}
	_ = restoredPubKeys.UnmarshalBinary(binary)

	// Checks
	if pubKeys.Count() != restoredPubKeys.Count() {
		t.Fatal()
	}

	// Keys data
	keysDataIsEqual := bytes.Compare(key.Bytes[:], restoredPubKeys.At[0].Bytes[:]) == 0
	if !keysDataIsEqual {
		t.Fatal()
	}
}

// Creates public keys list with maximum possible keys included.
// Serializes it, deserializes it back and then checks data equality.
func TestPubKeys_MarshallBinary_MaxElementsCount(t *testing.T) {

	// Reference data initialisation.
	keys := &PubKeys{}
	for j := 0; j < PubKeysMaxCount; j++ {
		key := &PubKey{}
		_, err := rand.Read(key.Bytes[:])
		if err != nil {
			t.Fatal()
		}

		err = keys.Add(key)
		if err != nil {
			t.Fatal()
		}
	}

	// Marshalling.
	binary, _ := keys.MarshalBinary()

	restoredKeys := &PubKeys{}
	_ = restoredKeys.UnmarshalBinary(binary)

	// Checks
	if restoredKeys.Count() != PubKeysMaxCount {
		t.Fatal()
	}
	if restoredKeys.Count() != keys.Count() {
		t.Fatal()
	}

	for i, restoredKey := range restoredKeys.At {
		pubKeysNAreEqual := bytes.Compare(restoredKey.Bytes[:], keys.At[i].Bytes[:]) == 0
		if !pubKeysNAreEqual {
			t.Fatal()
		}
	}
}
