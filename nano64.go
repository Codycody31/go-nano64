package nano64

import (
	"crypto/rand"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	// TimestampBits is the number of bits allocated to the millisecond timestamp (0..2^44-1).
	TimestampBits = 44

	// RandomBits is the number of bits allocated to the random field per millisecond (0..2^20-1).
	RandomBits = 20

	// timestampShift is the bit shift used to position the timestamp above the random field.
	timestampShift = RandomBits

	// timestampMask is the mask for extracting the 44-bit timestamp from a u64 value.
	timestampMask = (1 << TimestampBits) - 1

	// randomMask is the mask for the 20-bit random field.
	randomMask = (1 << RandomBits) - 1

	// maxTimestamp is the maximum timestamp value (2^44 - 1).
	maxTimestamp = timestampMask
)

// RNG is a function type for entropy source that returns `bits` random bits (1..32).
type RNG func(bits int) (uint32, error)

// Clock is a function type for a clock that returns epoch milliseconds.
type Clock func() int64

// Nano64 represents a 64-bit time-sortable identifier with 44-bit timestamp and 20-bit random field.
// Canonical representation is an unsigned 64-bit integer (0..2^64-1).
type Nano64 struct {
	value uint64
}

var (
	// lastTimestamp is used by GenerateMonotonic to track the last used timestamp.
	lastTimestamp int64 = -1

	// lastRandom is used by GenerateMonotonic to track the last used random value.
	lastRandom uint64

	// monotonicMutex protects the monotonic generation state.
	monotonicMutex sync.Mutex
)

// DefaultRNG provides a cryptographically-secure RNG using crypto/rand.
// Returns an unsigned integer with exactly `bits` bits of entropy.
func DefaultRNG(bits int) (uint32, error) {
	if bits <= 0 || bits > 32 {
		return 0, fmt.Errorf("bits must be 1-32, got %d", bits)
	}

	// Generate 4 bytes for simplicity
	buf := make([]byte, 4)
	if _, err := rand.Read(buf); err != nil {
		return 0, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	// Convert to uint32 and mask to requested bits
	val := binary.BigEndian.Uint32(buf)
	if bits == 32 {
		return val, nil
	}

	mask := uint32((1 << bits) - 1)
	return val & mask, nil
}

// DefaultClock returns the current time in Unix milliseconds.
func DefaultClock() int64 {
	return time.Now().UnixMilli()
}

// New creates a new Nano64 from a uint64 value.
func New(value uint64) Nano64 {
	return Nano64{value: value}
}

// Uint64Value returns the unsigned 64-bit integer value.
func (n Nano64) Uint64Value() uint64 {
	return n.value
}

// Value implements the driver.Valuer interface for SQL database support.
// Returns the ID as a byte slice for storage in SQL databases as BYTEA.
func (n Nano64) Value() (driver.Value, error) {
	return n.ToBytes(), nil
}

// Scan implements the sql.Scanner interface for SQL database support.
// Accepts int64 or uint64 values from SQL databases.
func (n *Nano64) Scan(value interface{}) error {
	if value == nil {
		n.value = 0
		return nil
	}

	switch v := value.(type) {
	case int64:
		n.value = uint64(v)
		return nil
	case uint64:
		n.value = v
		return nil
	case []byte:
		if len(v) != 8 {
			return fmt.Errorf("invalid byte length for Nano64: %d", len(v))
		}
		parsed, err := BigIntHelpers.FromBytesBE(v)
		if err != nil {
			return fmt.Errorf("failed to scan bytes: %w", err)
		}
		n.value = parsed
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Nano64", value)
	}
}

// MarshalJSON implements the json.Marshaler interface.
// Encodes the Nano64 as a hex string in JSON.
func (n Nano64) MarshalJSON() ([]byte, error) {
	return json.Marshal(n.ToHex())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
// Accepts either a hex string or a numeric value from JSON.
func (n *Nano64) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as string first (hex format)
	var hexStr string
	if err := json.Unmarshal(data, &hexStr); err == nil {
		parsed, err := FromHex(hexStr)
		if err != nil {
			return fmt.Errorf("failed to parse hex string: %w", err)
		}
		*n = parsed
		return nil
	}

	// Try to unmarshal as number
	var num uint64
	if err := json.Unmarshal(data, &num); err != nil {
		return fmt.Errorf("failed to unmarshal Nano64: expected hex string or number")
	}
	*n = Nano64{value: num}
	return nil
}

// GetTimestamp extracts the embedded UNIX-epoch milliseconds from the ID.
// Returns integer milliseconds in range [0, 2^44-1].
func (n Nano64) GetTimestamp() int64 {
	return int64((n.value >> timestampShift) & timestampMask)
}

// GetRandom extracts the 20-bit random field from the ID.
func (n Nano64) GetRandom() uint32 {
	return uint32(n.value & randomMask)
}

// ToDate builds a time.Time from the embedded timestamp.
func (n Nano64) ToDate() time.Time {
	return time.UnixMilli(n.GetTimestamp())
}

// Generate creates an ID with a given or current timestamp.
// Random field is filled with DefaultRNG(20) bits of entropy.
func Generate(timestamp int64, rng RNG) (Nano64, error) {
	if timestamp < 0 {
		return Nano64{}, fmt.Errorf("timestamp cannot be negative: %d", timestamp)
	}
	if timestamp > maxTimestamp {
		return Nano64{}, fmt.Errorf("timestamp exceeds 44-bit range: %d > %d", timestamp, maxTimestamp)
	}

	if rng == nil {
		rng = DefaultRNG
	}

	randVal, err := rng(RandomBits)
	if err != nil {
		return Nano64{}, fmt.Errorf("failed to generate random value: %w", err)
	}

	ms := uint64(timestamp) & timestampMask
	random := uint64(randVal) & randomMask
	value := (ms << timestampShift) | random

	return Nano64{value: value}, nil
}

// GenerateNow creates an ID with the current timestamp using DefaultClock.
func GenerateNow(rng RNG) (Nano64, error) {
	return Generate(DefaultClock(), rng)
}

// GenerateDefault creates an ID with the current timestamp and default RNG.
func GenerateDefault() (Nano64, error) {
	return GenerateNow(DefaultRNG)
}

// GenerateMonotonic creates monotonic IDs. Nondecreasing across calls in one process.
// If the per-ms sequence wraps, the timestamp is bumped by 1 ms and the random field resets to 0.
func GenerateMonotonic(timestamp int64, rng RNG) (Nano64, error) {
	if timestamp < 0 {
		return Nano64{}, fmt.Errorf("timestamp cannot be negative: %d", timestamp)
	}
	if timestamp > maxTimestamp {
		return Nano64{}, fmt.Errorf("timestamp exceeds 44-bit range: %d > %d", timestamp, maxTimestamp)
	}

	if rng == nil {
		rng = DefaultRNG
	}

	monotonicMutex.Lock()
	defer monotonicMutex.Unlock()

	// Enforce nondecreasing time
	t := timestamp
	if t < lastTimestamp {
		t = lastTimestamp
	}

	var random uint64
	if t == lastTimestamp {
		// Same ms → increment
		random = (lastRandom + 1) & randomMask
		if random == 0 {
			// Per-ms space exhausted → move to next ms and start at 0
			t++
			if t > maxTimestamp {
				return Nano64{}, fmt.Errorf("timestamp overflow after incrementing for monotonic generation")
			}
			lastTimestamp = t
			lastRandom = 0
			ms := uint64(t) & timestampMask
			value := ms << timestampShift
			return Nano64{value: value}, nil
		}
	} else {
		// First ID in this newer ms
		randVal, err := rng(RandomBits)
		if err != nil {
			return Nano64{}, fmt.Errorf("failed to generate random value: %w", err)
		}
		random = uint64(randVal) & randomMask
	}

	lastTimestamp = t
	lastRandom = random

	ms := uint64(t) & timestampMask
	value := (ms << timestampShift) | random
	return Nano64{value: value}, nil
}

// GenerateMonotonicNow creates a monotonic ID with the current timestamp.
func GenerateMonotonicNow(rng RNG) (Nano64, error) {
	return GenerateMonotonic(DefaultClock(), rng)
}

// GenerateMonotonicDefault creates a monotonic ID with current timestamp and default RNG.
func GenerateMonotonicDefault() (Nano64, error) {
	return GenerateMonotonicNow(DefaultRNG)
}

// Compare compares two IDs as unsigned 64-bit numbers.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func Compare(a, b Nano64) int {
	if a.value < b.value {
		return -1
	} else if a.value > b.value {
		return 1
	}
	return 0
}

// Equals checks equality by unsigned value.
func (n Nano64) Equals(other Nano64) bool {
	return Compare(n, other) == 0
}

// String returns a string representation for debugging.
func (n Nano64) String() string {
	return fmt.Sprintf("Nano64{value: %d, timestamp: %d, random: %d}",
		n.value, n.GetTimestamp(), n.GetRandom())
}

// ToHex returns uppercase 16-char hex encoding of the u64, with a dash between timestamp and random parts.
func (n Nano64) ToHex() string {
	full := fmt.Sprintf("%016X", n.value)
	// Split 44-bit (11 hex digits) timestamp + 20-bit (5 hex digits) random = 16 hex total
	const split = 11 // ceil(44 / 4)
	return full[:split] + "-" + full[split:]
}

// ToBytes returns 8-byte big-endian encoding of the u64.
func (n Nano64) ToBytes() []byte {
	return BigIntHelpers.ToBytesBE(n.value)
}

// FromHex parses from 17-char dashed hex (timestamp-random) or plain 16-char hex.
// Accepts uppercase or lowercase, optional `0x` prefix.
func FromHex(hexStr string) (Nano64, error) {
	clean := strings.ReplaceAll(hexStr, "-", "")
	if strings.HasPrefix(clean, "0x") || strings.HasPrefix(clean, "0X") {
		clean = clean[2:]
	}

	if len(clean) != 16 {
		return Nano64{}, fmt.Errorf("hex must be 16 chars after removing dash, got %d", len(clean))
	}

	bytes, err := Hex.ToBytes(clean)
	if err != nil {
		return Nano64{}, fmt.Errorf("invalid hex: %w", err)
	}

	if len(bytes) != 8 {
		return Nano64{}, fmt.Errorf("hex must decode to 8 bytes, got %d", len(bytes))
	}

	value, err := BigIntHelpers.FromBytesBE(bytes)
	if err != nil {
		return Nano64{}, fmt.Errorf("failed to parse bytes: %w", err)
	}

	return Nano64{value: value}, nil
}

// FromBytes parses from 8 big-endian bytes.
func FromBytes(bytes []byte) (Nano64, error) {
	value, err := BigIntHelpers.FromBytesBE(bytes)
	if err != nil {
		return Nano64{}, fmt.Errorf("failed to parse bytes: %w", err)
	}
	return Nano64{value: value}, nil
}

// FromUint64 creates a Nano64 from a uint64 value.
func FromUint64(value uint64) Nano64 {
	return Nano64{value: value}
}
