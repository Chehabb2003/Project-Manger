package sync

type BootstrapPacket struct {
	EphemeralPub []byte
	DeviceID     string
	Signature    []byte
}
