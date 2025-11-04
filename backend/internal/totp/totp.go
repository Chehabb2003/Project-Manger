package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"strings"
	"time"
)

const (
	DefaultStep   = 30 * time.Second
	DefaultDigits = 6
	secretSize    = 20 // 160-bit secret
)

func GenerateSecret() (string, error) {
	secret := make([]byte, secretSize)
	if _, err := rand.Read(secret); err != nil {
		return "", err
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secret)
	return enc, nil
}

func Verify(code, secret string, when time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != DefaultDigits {
		return false
	}
	secretBytes, err := decodeSecret(secret)
	if err != nil {
		return false
	}
	defer zero(secretBytes)

	step := int64(DefaultStep / time.Second)
	if step <= 0 {
		step = 30
	}
	counter := when.Unix() / step
	for i := int64(-1); i <= 1; i++ {
		cur := counter + i
		if cur < 0 {
			continue
		}
		if computeCode(secretBytes, uint64(cur)) == code {
			return true
		}
	}
	return false
}

func ProvisionURI(account, issuer, secret string) string {
	escapedAccount := strings.ReplaceAll(account, " ", "")
	escapedIssuer := strings.ReplaceAll(issuer, " ", "")
	period := int(DefaultStep / time.Second)
	if period <= 0 {
		period = 30
	}
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=%d&period=%d",
		urlEscape(escapedIssuer), urlEscape(escapedAccount), secret, urlEscape(escapedIssuer), DefaultDigits, period)
}

func computeCode(secret []byte, counter uint64) string {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)

	mac := hmac.New(sha1.New, secret)
	mac.Write(buf[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0F
	trunc := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7FFFFFFF
	code := trunc % 1000000
	return fmt.Sprintf("%0*d", DefaultDigits, code)
}

func decodeSecret(secret string) ([]byte, error) {
	secret = strings.ToUpper(strings.TrimSpace(secret))
	decoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	return decoder.DecodeString(secret)
}

func zero(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func urlEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			b.WriteRune(r)
			continue
		}
		for _, bt := range []byte(string(r)) {
			b.WriteString(fmt.Sprintf("%%%02X", bt))
		}
	}
	return b.String()
}
