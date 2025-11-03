package sync

type BootstrapPacket struct {
	EphemeralPub []byte
	DeviceID     string
	Signature    []byte
}

// TODO: implement QR bootstrap and E2E VRK transfer
