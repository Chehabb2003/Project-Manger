package vault

type Policy struct {
	LockTimeout      int64  `json:"lock_timeout_ms"`
	ClipboardTimeout int64  `json:"clipboard_timeout_ms"`
	RehashTargetM    uint32 `json:"rehash_target_m"`
	RehashTargetT    uint32 `json:"rehash_target_t"`
	RehashTargetP    uint8  `json:"rehash_target_p"`
}

func DefaultPolicy() Policy {
	return Policy{
		LockTimeout:      5 * 60 * 1000,
		ClipboardTimeout: 25 * 1000,
		RehashTargetM:    1024 * 1024,
		RehashTargetT:    3,
		RehashTargetP:    4,
	}
}
