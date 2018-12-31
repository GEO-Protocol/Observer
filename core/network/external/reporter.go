package external

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"geo-observers-blockchain/core/common/errors"
	"geo-observers-blockchain/core/settings"
	"io/ioutil"
	"os"
)

var (
	configuration *Configuration = nil
	number        int32          = -1
)

// todo: implement
// todo: comments
type Reporter struct {
	settings *settings.Settings
}

func NewReporter(settings *settings.Settings) *Reporter {
	return &Reporter{
		settings: settings,
	}
}

// todo: cache the results internally
func (r *Reporter) GetCurrentConfiguration() (*Configuration, error) {
	configuration = r.temptStaticConfiguration()

	for i, observer := range configuration.Observers {
		if observer.Host == r.settings.Observers.GNS.Host && observer.Port == r.settings.Observers.GNS.Port {
			configuration.CurrentObserverIndex = uint16(i)
		}
	}

	return r.temptStaticConfiguration(), nil
}

// todo: cache the results
func (r *Reporter) GetCurrentObserverNumber() (uint16, error) {
	if number >= 0 {
		return uint16(number), nil
	}

	conf := r.temptStaticConfiguration()
	for i, observer := range conf.Observers {
		if observer.Host == r.settings.Observers.Network.Host {
			if observer.Port == r.settings.Observers.Network.Port {
				number = int32(i)
				return uint16(number), nil
			}
		}
	}

	return 0, errors.NilParameter
}

// todo: sort observers in strict order!
func (r *Reporter) temptStaticConfiguration() *Configuration {
	if configuration == nil {
		observers := make([]*Observer, 0, 4)
		observers = append(observers, NewObserver("127.0.0.1", 3000, r.tempLoadPublicKey(1)))
		observers = append(observers, NewObserver("127.0.0.1", 3001, r.tempLoadPublicKey(2)))
		observers = append(observers, NewObserver("127.0.0.1", 3002, r.tempLoadPublicKey(3)))
		observers = append(observers, NewObserver("127.0.0.1", 3003, r.tempLoadPublicKey(4)))

		configuration = NewConfiguration(0, r.sortObservers(observers))
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
