package vault

import (
	"encoding/json"
	"errors"
	"os"
)

func readHeader(path string) (Header, error) {
	var h Header
	b, err := os.ReadFile(path)
	if err != nil {
		return h, err
	}
	if err := json.Unmarshal(b, &h); err != nil {
		return h, err
	}
	return h, nil
}

func writeHeader(path string, h Header) error {
	b, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, b, 0600)
}

var ErrNotUnlocked = errors.New("vault: not unlocked")
