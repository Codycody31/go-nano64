package nano64

import (
	"testing"
	"time"
)

func TestNano64_New(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		want  uint64
	}{
		{"zero", 0, 0},
		{"max", ^uint64(0), ^uint64(0)},
		{"random", 0x123456789ABCDEF0, 0x123456789ABCDEF0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := New(tt.value)
			if got := id.Uint64Value(); got != tt.want {
				t.Errorf("New(%d).Uint64Value() = %d, want %d", tt.value, got, tt.want)
			}
		})
	}
}

func TestNano64_Generate(t *testing.T) {
	timestamp := int64(1234567890123)

	rng := func(bits int) (uint32, error) {
		return 0x12345, nil // Fixed value for testing
	}

	id, err := Generate(timestamp, rng)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if got := id.GetTimestamp(); got != timestamp {
		t.Errorf("GetTimestamp() = %d, want %d", got, timestamp)
	}

	expectedRandom := uint32(0x12345)
	if got := id.GetRandom(); got != expectedRandom {
		t.Errorf("GetRandom() = %d, want %d", got, expectedRandom)
	}
}

func TestNano64_GenerateDefault(t *testing.T) {
	id, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}

	// Check that timestamp is recent (within last minute)
	now := time.Now().UnixMilli()
	ts := id.GetTimestamp()
	if ts < now-60000 || ts > now+1000 {
		t.Errorf("timestamp %d is not recent (now: %d)", ts, now)
	}

	// Check that random field is within valid range
	random := id.GetRandom()
	if random >= (1 << RandomBits) {
		t.Errorf("random value %d exceeds maximum %d", random, (1<<RandomBits)-1)
	}
}

func TestNano64_GenerateMonotonic(t *testing.T) {
	timestamp := int64(1234567890123)

	rng := func(bits int) (uint32, error) {
		return 0x12345, nil
	}

	// Generate first ID
	id1, err := GenerateMonotonic(timestamp, rng)
	if err != nil {
		t.Fatalf("GenerateMonotonic() error = %v", err)
	}

	// Generate second ID with same timestamp
	id2, err := GenerateMonotonic(timestamp, rng)
	if err != nil {
		t.Fatalf("GenerateMonotonic() error = %v", err)
	}

	// Second ID should be greater than first
	if Compare(id2, id1) <= 0 {
		t.Errorf("monotonic IDs not increasing: %d <= %d", id2.Uint64Value(), id1.Uint64Value())
	}

	// Both should have same timestamp
	if id1.GetTimestamp() != id2.GetTimestamp() {
		t.Errorf("timestamps differ: %d != %d", id1.GetTimestamp(), id2.GetTimestamp())
	}
}

func TestNano64_ToHex(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
		want  string
	}{
		{"zero", 0, "00000000000-00000"},
		{"max", ^uint64(0), "FFFFFFFFFFF-FFFFF"},
		{"example", 0x123456789ABCDEF0, "123456789AB-CDEF0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := New(tt.value)
			if got := id.ToHex(); got != tt.want {
				t.Errorf("ToHex() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNano64_FromHex(t *testing.T) {
	tests := []struct {
		name    string
		hex     string
		want    uint64
		wantErr bool
	}{
		{"zero", "00000000000-00000", 0, false},
		{"max", "FFFFFFFFFFF-FFFFF", ^uint64(0), false},
		{"example", "123456789AB-CDEF0", 0x123456789ABCDEF0, false},
		{"no dash", "123456789ABCDEF0", 0x123456789ABCDEF0, false},
		{"lowercase", "123456789ab-cdef0", 0x123456789ABCDEF0, false},
		{"0x prefix", "0x123456789ABCDEF0", 0x123456789ABCDEF0, false},
		{"invalid length", "123", 0, true},
		{"invalid char", "123456789AB-CDEFG", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FromHex(tt.hex)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromHex() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Uint64Value() != tt.want {
				t.Errorf("FromHex() = %d, want %d", got.Uint64Value(), tt.want)
			}
		})
	}
}

func TestNano64_ToBytes_FromBytes(t *testing.T) {
	original := New(0x123456789ABCDEF0)

	bytes := original.ToBytes()
	if len(bytes) != 8 {
		t.Errorf("ToBytes() length = %d, want 8", len(bytes))
	}

	parsed, err := FromBytes(bytes)
	if err != nil {
		t.Fatalf("FromBytes() error = %v", err)
	}

	if parsed.Uint64Value() != original.Uint64Value() {
		t.Errorf("roundtrip failed: %d != %d", parsed.Uint64Value(), original.Uint64Value())
	}
}

func TestNano64_Compare(t *testing.T) {
	id1 := New(100)
	id2 := New(200)
	id3 := New(100)

	if got := Compare(id1, id2); got != -1 {
		t.Errorf("Compare(100, 200) = %d, want -1", got)
	}

	if got := Compare(id2, id1); got != 1 {
		t.Errorf("Compare(200, 100) = %d, want 1", got)
	}

	if got := Compare(id1, id3); got != 0 {
		t.Errorf("Compare(100, 100) = %d, want 0", got)
	}
}

func TestNano64_Equals(t *testing.T) {
	id1 := New(100)
	id2 := New(200)
	id3 := New(100)

	if id1.Equals(id2) {
		t.Error("id1.Equals(id2) = true, want false")
	}

	if !id1.Equals(id3) {
		t.Error("id1.Equals(id3) = false, want true")
	}
}

func TestNano64_ToDate(t *testing.T) {
	timestamp := int64(1234567890123)

	rng := func(bits int) (uint32, error) {
		return 0, nil
	}

	id, err := Generate(timestamp, rng)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	date := id.ToDate()
	if date.UnixMilli() != timestamp {
		t.Errorf("ToDate().UnixMilli() = %d, want %d", date.UnixMilli(), timestamp)
	}
}

func TestDefaultRNG(t *testing.T) {
	tests := []struct {
		name    string
		bits    int
		wantErr bool
	}{
		{"valid 1 bit", 1, false},
		{"valid 20 bits", 20, false},
		{"valid 32 bits", 32, false},
		{"invalid 0 bits", 0, true},
		{"invalid 33 bits", 33, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DefaultRNG(tt.bits)
			if (err != nil) != tt.wantErr {
				t.Errorf("DefaultRNG() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				maxVal := uint32((1 << tt.bits) - 1)
				if got > maxVal {
					t.Errorf("DefaultRNG(%d) = %d, exceeds max %d", tt.bits, got, maxVal)
				}
			}
		})
	}
}

func TestGenerate_Errors(t *testing.T) {
	tests := []struct {
		name      string
		timestamp int64
		wantErr   bool
	}{
		{"negative timestamp", -1, true},
		{"valid timestamp", 1234567890123, false},
		{"max timestamp", (1 << TimestampBits) - 1, false},
		{"overflow timestamp", 1 << TimestampBits, true},
	}

	rng := func(bits int) (uint32, error) {
		return 0, nil
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Generate(tt.timestamp, rng)
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNano64_Value(t *testing.T) {
	tests := []struct {
		name    string
		value   uint64
		want    int64
		wantErr bool
	}{
		{"zero", 0, 0, false},
		{"positive", 12345, 12345, false},
		{"max int64", 0x7FFFFFFFFFFFFFFF, 0x7FFFFFFFFFFFFFFF, false},
		{"large value", 0x123456789ABCDEF0, int64(0x123456789ABCDEF0), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := New(tt.value)
			got, err := id.Value()
			if (err != nil) != tt.wantErr {
				t.Errorf("Value() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				gotInt64, ok := got.(int64)
				if !ok {
					t.Errorf("Value() returned type %T, want int64", got)
					return
				}
				if gotInt64 != tt.want {
					t.Errorf("Value() = %d, want %d", gotInt64, tt.want)
				}
			}
		})
	}
}

func TestNano64_Scan(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		want    uint64
		wantErr bool
	}{
		{"nil", nil, 0, false},
		{"int64 zero", int64(0), 0, false},
		{"int64 positive", int64(12345), 12345, false},
		{"int64 large", int64(0x123456789ABCDEF0), 0x123456789ABCDEF0, false},
		{"uint64 zero", uint64(0), 0, false},
		{"uint64 positive", uint64(12345), 12345, false},
		{"uint64 max", ^uint64(0), ^uint64(0), false},
		{"bytes 8 bytes", []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}, 0x123456789ABCDEF0, false},
		{"bytes zero", []byte{0, 0, 0, 0, 0, 0, 0, 0}, 0, false},
		{"bytes wrong length", []byte{1, 2, 3}, 0, true},
		{"string invalid type", "invalid", 0, true},
		{"float invalid type", 3.14, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id Nano64
			err := id.Scan(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id.Uint64Value() != tt.want {
				t.Errorf("Scan() resulted in value %d, want %d", id.Uint64Value(), tt.want)
			}
		})
	}
}

func TestNano64_ValueScan_Roundtrip(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{"zero", 0},
		{"small", 12345},
		{"large", 0x123456789ABCDEF0},
		{"max", ^uint64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := New(tt.value)

			// Convert to driver.Value
			driverValue, err := original.Value()
			if err != nil {
				t.Fatalf("Value() error = %v", err)
			}

			// Scan back
			var scanned Nano64
			err = scanned.Scan(driverValue)
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}

			// Compare
			if scanned.Uint64Value() != original.Uint64Value() {
				t.Errorf("roundtrip failed: %d != %d", scanned.Uint64Value(), original.Uint64Value())
			}
		})
	}
}

func TestNano64_Scan_BytesRoundtrip(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{"zero", 0},
		{"small", 12345},
		{"large", 0x123456789ABCDEF0},
		{"max", ^uint64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := New(tt.value)

			// Convert to bytes
			bytes := original.ToBytes()

			// Scan from bytes
			var scanned Nano64
			err := scanned.Scan(bytes)
			if err != nil {
				t.Fatalf("Scan() error = %v", err)
			}

			// Compare
			if scanned.Uint64Value() != original.Uint64Value() {
				t.Errorf("bytes roundtrip failed: %d != %d", scanned.Uint64Value(), original.Uint64Value())
			}
		})
	}
}

func BenchmarkGenerate(b *testing.B) {
	timestamp := time.Now().UnixMilli()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Generate(timestamp, DefaultRNG)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGenerateMonotonic(b *testing.B) {
	timestamp := time.Now().UnixMilli()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateMonotonic(timestamp, DefaultRNG)
		if err != nil {
			b.Fatal(err)
		}
	}
}
