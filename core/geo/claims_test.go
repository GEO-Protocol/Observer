package geo

import (
	"bytes"
	"crypto/rand"
	"geo-observers-blockchain/core/common"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/crypto/lamport"
	"testing"
)

func TestClaim_MarshalBinary_EmptyInternal(t *testing.T) {
	claim := &Claim{}
	_, err := claim.MarshalBinary()
	if err != common.ErrNilInternalDataStructure {
		t.Fatal()
	}
}

func TestClaim_UnmarshalBinary_EmptyData(t *testing.T) {
	claim := NewClaim()
	err := claim.UnmarshalBinary([]byte{})
	if err != common.ErrInvalidDataFormat {
		t.Fatal()
	}
}

// Note:
// MarshallBinary() and UnmarshalBinarymethods are checked further.

// --------------------------------------------------------------------------------------------------------------------

func TestClaims_Add_NilParameter(t *testing.T) {
	claims := &Claims{}
	err := claims.Add(nil)
	if err != common.ErrNilParameter {
		t.Fatal()
	}
}

func TestClaims_Add_Max(t *testing.T) {
	claims := &Claims{}
	for i := 0; i < ClaimsMaxCount; i++ {
		err := claims.Add(NewClaim())
		if err != nil {
			t.Fatal()
		}
	}

	err := claims.Add(&Claim{})
	if err != common.ErrMaxCountReached {
		t.Fatal("claims list must restrict max count of elements")
	}
}

func TestClaims_Count(t *testing.T) {
	claims := &Claims{}
	if claims.Count() != 0 {
		t.Fatal()
	}

	for i := 0; i < ClaimsMaxCount; i++ {
		_ = claims.Add(NewClaim())
		if claims.Count() != uint16(i+1) {
			t.Fatal()
		}
	}
}

// Creates empty claims list.
// Serializes it, deserializes it back and then checks data equality.
func TestClaims_MarshallBinary_NoData(t *testing.T) {
	claims := &Claims{}
	restoredClaims := &Claims{}

	binary, _ := claims.MarshalBinary()
	_ = restoredClaims.UnmarshalBinary(binary)

	// Empty serialised claims must be 2 bytes long.
	// (total count if claims serializes as uint16)
	if len(binary) != types.Uint16ByteSize {
		t.Fatal()
	}

	ReferenceSerializedData := []byte{0, 0}
	if bytes.Compare(binary, ReferenceSerializedData) != 0 {
		t.Fatal()
	}

	if claims.Count() != restoredClaims.Count() {
		t.Fatal()
	}
}

// Creates claims list with only one claim included.
// Serializes it, deserializes it back and then checks data equality.
func TestClaims_MarshallBinary_OneClaim_1024PubKeys(t *testing.T) {

	// Reference data initialisation.
	claim := NewClaim()
	claim.TxUUID.Bytes = [16]byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 1, 2, 3, 4, 5, 6}
	for i := 0; i < 1024; i++ {
		pubKey := &lamport.PubKey{}
		_, err := rand.Read(pubKey.Bytes[:])
		if err != nil {
			t.Fatal()
		}

		_ = claim.PubKeys.Add(pubKey)
	}

	claims := &Claims{}
	_ = claims.Add(claim)

	// Marshalling.
	binary, _ := claims.MarshalBinary()

	restoredClaims := &Claims{}
	_ = restoredClaims.UnmarshalBinary(binary)

	// Checks
	if claims.Count() != restoredClaims.Count() {
		t.Fatal()
	}

	// Transaction UUID
	transactionUUIDsAreEqual := bytes.Compare(
		claims.At[0].TxUUID.Bytes[:],
		restoredClaims.At[0].TxUUID.Bytes[:]) == 0

	if !transactionUUIDsAreEqual {
		t.Fatal()
	}

	// Pub keys count
	pubKeysCountAreEqual := restoredClaims.At[0].PubKeys.Count() == restoredClaims.At[0].PubKeys.Count()
	if !pubKeysCountAreEqual {
		t.Fatal()
	}

	// PubKeys data
	for i, pubKey := range claims.At[0].PubKeys.At {
		restoredPubKey := restoredClaims.At[0].PubKeys.At[i]
		pubKeyNIsEqual := bytes.Compare(pubKey.Bytes[:], restoredPubKey.Bytes[:]) == 0
		if !pubKeyNIsEqual {
			t.Fatal()
		}
	}
}

// Creates claims list with maximum possible claims included.
// Serializes it, deserializes it back and then checks data equality.
//
// WARN: Consumes more than 256MB of memory.
func TestClaims_MarshallBinary_MaxElementsCount(t *testing.T) {

	// Reference data initialisation.
	claims := &Claims{}
	for j := 0; j < ClaimsMaxCount; j++ {
		claim := NewClaim()
		_, err := rand.Read(claim.TxUUID.Bytes[:])
		if err != nil {
			t.Fatal()
		}

		pubKey := &lamport.PubKey{}
		_, err = rand.Read(pubKey.Bytes[:])
		if err != nil {
			t.Fatal()
		}
		_ = claim.PubKeys.Add(pubKey)

		err = claims.Add(claim)
		if err != nil {
			t.Fatal()
		}
	}

	// Marshalling.
	binary, _ := claims.MarshalBinary()

	restoredClaims := &Claims{}
	_ = restoredClaims.UnmarshalBinary(binary)

	// Checks
	if restoredClaims.Count() != ClaimsMaxCount {
		t.Fatal()
	}
	if restoredClaims.Count() != claims.Count() {
		t.Fatal()
	}

	for i, restoredClaim := range restoredClaims.At {
		transactionUUIDsAreEqual := bytes.Compare(
			restoredClaim.TxUUID.Bytes[:],
			claims.At[i].TxUUID.Bytes[:]) == 0

		if !transactionUUIDsAreEqual {
			t.Fatal()
		}

		for j, key := range restoredClaim.PubKeys.At {
			pubKeyNIsEqual := bytes.Compare(key.Bytes[:], claims.At[i].PubKeys.At[j].Bytes[:]) == 0
			if !pubKeyNIsEqual {
				t.Fatal()
			}
		}
	}
}
