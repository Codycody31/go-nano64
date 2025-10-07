package nano64

import (
	"encoding/binary"
	"fmt"
)

// BigIntHelpers provides big-endian ⇄ uint64 conversions for fixed 8-byte unsigned integers.
var BigIntHelpers = bigIntHelpers{}

type bigIntHelpers struct{}

// FromBytesBE reads a uint64 from 8 big-endian bytes.
func (bigIntHelpers) FromBytesBE(bytes []byte) (uint64, error) {
	if len(bytes) != 8 {
		return 0, fmt.Errorf("must be 8 bytes, got %d", len(bytes))
	}
	return binary.BigEndian.Uint64(bytes), nil
}

// ToBytesBE writes a uint64 to 8 big-endian bytes.
func (bigIntHelpers) ToBytesBE(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes
}
