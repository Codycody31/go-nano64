package nano64

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// Hex provides hex encoding/decoding helpers with strict validation.
var Hex = hexHelpers{}

type hexHelpers struct{}

// FromBytes converts bytes to uppercase hex string.
func (hexHelpers) FromBytes(bytes []byte) string {
	return strings.ToUpper(hex.EncodeToString(bytes))
}

// ToBytes parses hex string into bytes.
// Accepts optional "0x" prefix and is case-insensitive.
// Returns error if length is odd or non-hex chars are present.
func (hexHelpers) ToBytes(hexStr string) ([]byte, error) {
	h := hexStr
	if strings.HasPrefix(h, "0x") || strings.HasPrefix(h, "0X") {
		h = h[2:]
	}

	if len(h)%2 != 0 {
		return nil, fmt.Errorf("hex length must be even, got %d", len(h))
	}

	// Validate hex characters
	for i, r := range h {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return nil, fmt.Errorf("hex contains non-hex character '%c' at position %d", r, i)
		}
	}

	bytes, err := hex.DecodeString(h)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex: %w", err)
	}

	return bytes, nil
}
