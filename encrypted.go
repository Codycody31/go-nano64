package nano64

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
)

const (
	// IVLength is the length of the initialization vector for AES-GCM (96 bits).
	IVLength = 12

	// PayloadLength is the total length of encrypted payload: IV + ciphertext + tag.
	PayloadLength = IVLength + 8 + 16 // 36 bytes total
)

// EncryptedNano64 represents an authenticated encrypted wrapper for a Nano64 ID.
// Payload layout: 12-byte IV || 8-byte ciphertext || 16-byte GCM tag (36 bytes).
type EncryptedNano64 struct {
	// The decrypted original Nano64 ID.
	ID Nano64

	// The raw encrypted payload (IV ‖ cipher+tag).
	payload []byte

	// The AES-GCM cipher used for encryption/decryption.
	gcm cipher.AEAD
}

// ToEncryptedHex returns the 36-byte payload as 72-char uppercase hex.
func (e EncryptedNano64) ToEncryptedHex() string {
	return Hex.FromBytes(e.payload)
}

// ToEncryptedBytes returns a defensive copy of the raw payload bytes.
func (e EncryptedNano64) ToEncryptedBytes() []byte {
	result := make([]byte, len(e.payload))
	copy(result, e.payload)
	return result
}

// EncryptedIDConfig holds configuration for encrypted Nano64 operations.
type EncryptedIDConfig struct {
	gcm   cipher.AEAD
	clock Clock
	rng   RNG
}

// NewEncryptedIDConfig creates a new configuration for encrypted Nano64 operations.
// The aesKey must be 16, 24, or 32 bytes for AES-128, AES-192, or AES-256 respectively.
func NewEncryptedIDConfig(aesKey []byte, clock Clock, rng RNG) (*EncryptedIDConfig, error) {
	if clock == nil {
		clock = DefaultClock
	}
	if rng == nil {
		rng = DefaultRNG
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create AES cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &EncryptedIDConfig{
		gcm:   gcm,
		clock: clock,
		rng:   rng,
	}, nil
}

// generateIV generates a fresh 96-bit random IV.
func (c *EncryptedIDConfig) generateIV() ([]byte, error) {
	iv := make([]byte, IVLength)
	if _, err := rand.Read(iv); err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}
	return iv, nil
}

// Encrypt encrypts an existing Nano64 into an authenticated payload.
func (c *EncryptedIDConfig) Encrypt(id Nano64) (*EncryptedNano64, error) {
	iv, err := c.generateIV()
	if err != nil {
		return nil, err
	}

	plaintext := BigIntHelpers.ToBytesBE(id.value)
	ciphertext := c.gcm.Seal(nil, iv, plaintext, nil)

	if len(ciphertext) != 8+16 {
		return nil, fmt.Errorf("unexpected AES-GCM output length: %d", len(ciphertext))
	}

	// Construct payload: IV + ciphertext (includes tag)
	payload := make([]byte, PayloadLength)
	copy(payload[0:], iv)
	copy(payload[IVLength:], ciphertext)

	return &EncryptedNano64{
		ID:      id,
		payload: payload,
		gcm:     c.gcm,
	}, nil
}

// GenerateEncrypted generates a new Nano64, then encrypts it.
func (c *EncryptedIDConfig) GenerateEncrypted(timestamp int64) (*EncryptedNano64, error) {
	if timestamp == 0 {
		timestamp = c.clock()
	}

	id, err := Generate(timestamp, c.rng)
	if err != nil {
		return nil, fmt.Errorf("failed to generate ID: %w", err)
	}

	return c.Encrypt(id)
}

// GenerateEncryptedNow generates a new Nano64 with current timestamp, then encrypts it.
func (c *EncryptedIDConfig) GenerateEncryptedNow() (*EncryptedNano64, error) {
	return c.GenerateEncrypted(c.clock())
}

// FromEncryptedBytes decrypts from raw 36-byte payload.
func (c *EncryptedIDConfig) FromEncryptedBytes(bytes []byte) (*EncryptedNano64, error) {
	if len(bytes) != PayloadLength {
		return nil, fmt.Errorf("encrypted payload must be %d bytes, got %d", PayloadLength, len(bytes))
	}

	iv := bytes[:IVLength]
	ciphertext := bytes[IVLength:]

	plaintext, err := c.gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed: %w", err)
	}

	if len(plaintext) != 8 {
		return nil, fmt.Errorf("decryption yielded invalid length: %d", len(plaintext))
	}

	value, err := BigIntHelpers.FromBytesBE(plaintext)
	if err != nil {
		return nil, fmt.Errorf("failed to parse decrypted bytes: %w", err)
	}

	id := Nano64{value: value}

	// Make a defensive copy of the payload
	payload := make([]byte, len(bytes))
	copy(payload, bytes)

	return &EncryptedNano64{
		ID:      id,
		payload: payload,
		gcm:     c.gcm,
	}, nil
}

// FromEncryptedHex decrypts from 72-char hex payload.
func (c *EncryptedIDConfig) FromEncryptedHex(encHex string) (*EncryptedNano64, error) {
	bytes, err := Hex.ToBytes(encHex)
	if err != nil {
		return nil, fmt.Errorf("invalid hex: %w", err)
	}

	if len(bytes) != PayloadLength {
		return nil, fmt.Errorf("encrypted payload must be %d bytes, got %d", PayloadLength, len(bytes))
	}

	return c.FromEncryptedBytes(bytes)
}
