package platform

import (
	"time"
)

type Clipboard interface {
	Set(text string, ttl time.Duration) error
}

type noopClipboard struct{}

func (n noopClipboard) Set(string, time.Duration) error { return nil }

func NewClipboard() Clipboard { return noopClipboard{} }
