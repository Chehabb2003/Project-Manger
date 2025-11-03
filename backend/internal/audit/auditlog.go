package audit

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type Entry struct {
	TS   int64  `json:"ts"`
	What string `json:"what"`
	Hash string `json:"hash"`
}

type Log struct {
	lastHash []byte
	entries  []Entry
}

func New() *Log { return &Log{} }

func (l *Log) Append(what string) Entry {
	h := sha256.New()
	h.Write(l.lastHash)
	h.Write([]byte(what))
	sum := h.Sum(nil)
	l.lastHash = sum
	e := Entry{TS: time.Now().Unix(), What: what, Hash: hex.EncodeToString(sum)}
	l.entries = append(l.entries, e)
	return e
}

func (l *Log) Verify() error {
	var prev []byte
	for _, e := range l.entries {
		h := sha256.New()
		h.Write(prev)
		h.Write([]byte(e.What))
		sum := h.Sum(nil)
		if hex.EncodeToString(sum) != e.Hash {
			return fmt.Errorf("audit chain broken")
		}
		prev = sum
	}
	return nil
}

func (l *Log) Entries() []Entry { return append([]Entry(nil), l.entries...) }
