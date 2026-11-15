// Package privacy mã hoá dữ liệu cá nhân trong ứng dụng (NFR-3.1): AES-256-GCM cho giá
// trị, HMAC-SHA-256 cho khoá tra cứu (vd email) và mã ẩn danh cho nhật ký kiểm toán.
// Không phụ thuộc module nào; module nhận *Cipher qua Deps.
package privacy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// Config đọc khoá từ file (deploy/secrets, tạo bằng make init). Mỗi file chứa 32 byte
// mã base64. Đổi khoá thì dữ liệu cũ không đọc được: chưa hỗ trợ xoay khoá.
type Config struct {
	EncKey  string `env:"PRIVACY_ENC_KEY_FILE,file"`
	HMACKey string `env:"PRIVACY_HMAC_KEY_FILE,file"`
}

func (c Config) Validate() error {
	for name, v := range map[string]string{"PRIVACY_ENC_KEY_FILE": c.EncKey, "PRIVACY_HMAC_KEY_FILE": c.HMACKey} {
		k, err := decodeKey(v)
		if err != nil || len(k) != 32 {
			return fmt.Errorf("privacy: %s must hold 32 bytes encoded in base64", name)
		}
	}
	return nil
}

func decodeKey(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(s))
}

type Cipher struct {
	aead cipher.AEAD
	mac  []byte
}

func New(cfg Config) (*Cipher, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	enc, _ := decodeKey(cfg.EncKey)
	mac, _ := decodeKey(cfg.HMACKey)
	return NewFromKeys(enc, mac)
}

// NewFromKeys cho test: hai khoá 32 byte
func NewFromKeys(enc, mac []byte) (*Cipher, error) {
	block, err := aes.NewCipher(enc)
	if err != nil {
		return nil, fmt.Errorf("privacy: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("privacy: %w", err)
	}
	return &Cipher{aead: aead, mac: mac}, nil
}

var ErrDecrypt = errors.New("privacy: cannot decrypt value")

// Encrypt trả nonce || ciphertext
func (c *Cipher) Encrypt(plain string) []byte {
	nonce := make([]byte, c.aead.NonceSize())
	_, _ = rand.Read(nonce)
	return c.aead.Seal(nonce, nonce, []byte(plain), nil)
}

func (c *Cipher) Decrypt(data []byte) (string, error) {
	n := c.aead.NonceSize()
	if len(data) < n {
		return "", ErrDecrypt
	}
	plain, err := c.aead.Open(nil, data[:n], data[n:], nil)
	if err != nil {
		return "", ErrDecrypt
	}
	return string(plain), nil
}

// Hash là HMAC-SHA-256, dùng làm khoá tra cứu cho giá trị đã mã hoá
func (c *Cipher) Hash(value string) []byte {
	h := hmac.New(sha256.New, c.mac)
	h.Write([]byte(value))
	return h.Sum(nil)
}

// Pseudonym là mã ẩn danh ổn định của một account trong nhật ký kiểm toán (FR-5.3):
// cùng account luôn cùng mã, nhưng không suy ngược ra account nếu không có khoá.
func (c *Cipher) Pseudonym(accountID uuid.UUID) string {
	return "u-" + hex.EncodeToString(c.Hash("audit:" + accountID.String())[:5])
}
