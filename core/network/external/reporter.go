package external

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/crypto/keystore"
	"io/ioutil"
	"math"
	"os"
)

var (
	configuration *Configuration = nil
	number        int32          = -1
)

// todo: comments
type Reporter struct {
	keystore *keystore.KeyStore
}

func NewReporter(keystore *keystore.KeyStore) (reporter *Reporter) {
	reporter = &Reporter{
		keystore: keystore,
	}

	return
}

// todo: cache the results internally
func (r *Reporter) GetCurrentConfiguration() (conf *Configuration, err error) {
	conf = r.temptStaticConfiguration()
	conf.CurrentObserverIndex, err = r.GetCurrentObserverIndex()
	return
}

// todo: cache the results
func (r *Reporter) GetCurrentObserverIndex() (uint16, error) {
	if number >= 0 {
		return uint16(number), nil
	}

	conf := r.temptStaticConfiguration()
	for i, observer := range conf.Observers {
		if r.keystore.IsEqualPubKey(observer.PubKey) {
			number = int32(i)
			return uint16(number), nil
		}
	}

	return math.MaxUint16, errors.NilParameter
}

// todo: sort observers in strict order!
func (r *Reporter) temptStaticConfiguration() *Configuration {
	if configuration == nil {
		//if settings.Conf.Debug {
		//	observers := make([]*Observer, 0, 4)
		//	observers = append(observers, NewObserver("127.0.0.1", 3000, r.tempLoadPublicKey(0)))
		//	observers = append(observers, NewObserver("127.0.0.1", 3001, r.tempLoadPublicKey(1)))
		//	observers = append(observers, NewObserver("127.0.0.1", 3002, r.tempLoadPublicKey(2)))
		//	observers = append(observers, NewObserver("127.0.0.1", 3003, r.tempLoadPublicKey(3)))
		//
		//	configuration = NewConfiguration(0, r.sortObservers(observers))
		//
		//} else {
		observers := make([]*Observer, 0, 4)
		observers = append(observers, NewObserver("68.183.146.232", 3000, r.tempLoadPublicKey(0)))
		observers = append(observers, NewObserver("46.101.51.158", 3000, r.tempLoadPublicKey(1)))
		observers = append(observers, NewObserver("206.189.84.116", 3000, r.tempLoadPublicKey(2)))
		observers = append(observers, NewObserver("159.89.115.33", 3000, r.tempLoadPublicKey(3)))

		configuration = NewConfiguration(0, r.sortObservers(observers))
		//}
	}

	return configuration
}

// todo: document strict order is needed for block generation order
func (r *Reporter) sortObservers(observers []*Observer) []*Observer {
	// todo: implement strict sorting based on hash of the observer
	return observers
}

func (r *Reporter) tempLoadPublicKey(obsNumber int) *ecdsa.PublicKey {
	keyFile, err := os.Open(fmt.Sprint(obsNumber, "_p521.key"))
	if err != nil {
		panic(err)
	}

	pemEncoded, err := ioutil.ReadAll(keyFile)
	if err != nil {
		panic(err)
	}

	block, _ := pem.Decode([]byte(pemEncoded))
	pkey, err := x509.ParseECPrivateKey(block.Bytes)

	pkey.Curve = elliptic.P521()
	return &ecdsa.PublicKey{Curve: elliptic.P521(), X: pkey.PublicKey.X, Y: pkey.PublicKey.Y}
}
