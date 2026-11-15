package service

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Hasher băm và kiểm tra mật khẩu
type Hasher interface {
	Hash(pw string) (string, error)
	Verify(hash, pw string) bool
}

// Argon2id theo tham số khuyến nghị của OWASP (19 MiB, 2 vòng, 1 luồng). Chuỗi lưu
// theo định dạng PHC: $argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
type Argon2id struct {
	Memory  uint32
	Time    uint32
	Threads uint8
}

func NewArgon2id() Argon2id { return Argon2id{Memory: 19 * 1024, Time: 2, Threads: 1} }

func (a Argon2id) Hash(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(pw), salt, a.Time, a.Memory, a.Threads, 32)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, a.Memory, a.Time, a.Threads, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

func (a Argon2id) Verify(hash, pw string) bool {
	m, t, p, salt, key, err := parseArgon(hash)
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, t, m, p, uint32(len(key)))
	return subtle.ConstantTimeCompare(got, key) == 1
}

func parseArgon(s string) (m, t uint32, p uint8, salt, key []byte, err error) {
	parts := strings.Split(s, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return 0, 0, 0, nil, nil, errors.New("not an argon2id hash")
	}
	if _, err = fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return
	}
	b64 := base64.RawStdEncoding
	if salt, err = b64.DecodeString(parts[4]); err != nil {
		return
	}
	key, err = b64.DecodeString(parts[5])
	return
}
