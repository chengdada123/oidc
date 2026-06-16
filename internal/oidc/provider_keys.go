package oidcbridge

var BrokerKID string
var BrokerPublicKeyPEM string
var BrokerPrivateKeyPEM string

func init() {
	kp, err := GenerateBrokerKeypair()
	if err != nil {
		panic(err)
	}
	BrokerKID = kp.KID
	BrokerPublicKeyPEM = kp.PublicPEM
	BrokerPrivateKeyPEM = kp.PrivatePEM
}
