package crypto

// Zero overwrites a byte slice in memory with zeros.
// This version works on all operating systems.
func Zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}
