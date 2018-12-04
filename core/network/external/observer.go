package external

import (
	"crypto/ecdsa"
	"geo-observers-blockchain/core/common/types"
	"geo-observers-blockchain/core/utils"
)

type Observer struct {
	PubKey *ecdsa.PublicKey
	Host   string
	Port   uint16
}

// todo: add pub key
func NewObserver(host string, port uint16, pubKey *ecdsa.PublicKey) *Observer {
	return &Observer{
		PubKey: pubKey,
		Host:   host,
		Port:   port,
	}
}

func (o *Observer) Hash() types.SHA256Container {
	xData := o.PubKey.X.Bytes()
	yData := o.PubKey.Y.Bytes()
	hostData := []byte(o.Host)
	portData := utils.MarshalUint16(o.Port)

	dataSize := len(xData) +
		len(yData) +
		len(hostData) +
		len(portData)

	data := make([]byte, 0, dataSize)
	return types.NewSHA256Container(data)
}
