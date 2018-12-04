package ecdsa

import (
	e "crypto/ecdsa"
	"geo-observers-blockchain/core/utils"
)

func PublicKeyMarshallBinary(key *e.PublicKey) (data []byte, err error) {
	xData := key.X.Bytes()
	yData := key.Y.Bytes()
	totalDataSize := len(xData) + len(yData)

	data = utils.ChainByteSlices(
		utils.MarshalUint16(uint16(totalDataSize)), xData, yData)

	return
}
