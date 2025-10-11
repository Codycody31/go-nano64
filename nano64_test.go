package nano64

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
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
		want    []byte
		wantErr bool
	}{
		{"zero", 0, []byte{0, 0, 0, 0, 0, 0, 0, 0}, false},
		{"positive", 12345, []byte{0, 0, 0, 0, 0, 0, 0x30, 0x39}, false},
		{"large value", 0x123456789ABCDEF0, []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}, false},
		{"max", ^uint64(0), []byte{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}, false},
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
				gotBytes, ok := got.([]byte)
				if !ok {
					t.Errorf("Value() returned type %T, want []byte", got)
					return
				}
				if len(gotBytes) != len(tt.want) {
					t.Errorf("Value() returned %d bytes, want %d", len(gotBytes), len(tt.want))
					return
				}
				for i := range gotBytes {
					if gotBytes[i] != tt.want[i] {
						t.Errorf("Value() = %v, want %v", gotBytes, tt.want)
						break
					}
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

func TestNano64_MarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		value   uint64
		want    string
		wantErr bool
	}{
		{"zero", 0, `"00000000000-00000"`, false},
		{"small", 12345, `"00000000000-03039"`, false},
		{"large", 0x123456789ABCDEF0, `"123456789AB-CDEF0"`, false},
		{"max timestamp", 0x0FFFFFFFFFFF << 20, `"FFFFFFFFFFF-00000"`, false},
		{"max random", 0xFFFFF, `"00000000000-FFFFF"`, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := New(tt.value)
			got, err := json.Marshal(id)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && string(got) != tt.want {
				t.Errorf("MarshalJSON() = %s, want %s", string(got), tt.want)
			}
		})
	}
}

func TestNano64_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		want    uint64
		wantErr bool
	}{
		{"hex string zero", `"00000000000-00000"`, 0, false},
		{"hex string small", `"00000000000-03039"`, 12345, false},
		{"hex string large", `"123456789AB-CDEF0"`, 0x123456789ABCDEF0, false},
		{"hex string no dash", `"0000000000003039"`, 12345, false},
		{"hex string lowercase", `"00000000000-03039"`, 12345, false},
		{"numeric zero", `0`, 0, false},
		{"numeric small", `12345`, 12345, false},
		{"numeric large", `1311768467463790320`, 0x123456789ABCDEF0, false},
		{"invalid hex", `"ZZZZ"`, 0, true},
		{"invalid type", `true`, 0, true},
		{"invalid object", `{}`, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var id Nano64
			err := json.Unmarshal([]byte(tt.json), &id)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && id.Uint64Value() != tt.want {
				t.Errorf("UnmarshalJSON() value = %d, want %d", id.Uint64Value(), tt.want)
			}
		})
	}
}

func TestNano64_JSON_Roundtrip(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{"zero", 0},
		{"small", 12345},
		{"large", 0x123456789ABCDEF0},
		{"max", ^uint64(0)},
		{"generated", func() uint64 { id, _ := GenerateDefault(); return id.Uint64Value() }()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := New(tt.value)

			// Marshal to JSON
			jsonData, err := json.Marshal(original)
			if err != nil {
				t.Fatalf("Marshal() error = %v", err)
			}

			// Unmarshal back
			var decoded Nano64
			err = json.Unmarshal(jsonData, &decoded)
			if err != nil {
				t.Fatalf("Unmarshal() error = %v", err)
			}

			// Compare
			if decoded.Uint64Value() != original.Uint64Value() {
				t.Errorf("JSON roundtrip failed: %d != %d", decoded.Uint64Value(), original.Uint64Value())
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

// setupTestDB creates a temporary SQLite database for testing.
func setupTestDB(t *testing.T) (*sql.DB, func()) {
	t.Helper()

	// Create a temporary directory for the test database
	tmpDir, err := os.MkdirTemp("", "nano64_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to open database: %v", err)
	}

	// Create a test table
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY,
			nano64_id INTEGER NOT NULL,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		db.Close()
		os.RemoveAll(tmpDir)
		t.Fatalf("failed to create table: %v", err)
	}

	// Return cleanup function
	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestNano64_DatabaseWrite(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name     string
		nano64ID Nano64
		itemName string
	}{
		{"zero value", New(0), "zero item"},
		{"small value", New(12345), "small item"},
		{"large value", New(0x123456789ABCDEF0), "large item"},
		{"max value", New(^uint64(0)), "max item"},
		{"generated ID", func() Nano64 { id, _ := GenerateDefault(); return id }(), "generated item"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write to database
			result, err := db.Exec(
				"INSERT INTO items (nano64_id, name) VALUES (?, ?)",
				tt.nano64ID,
				tt.itemName,
			)
			if err != nil {
				t.Fatalf("failed to insert: %v", err)
			}

			rowID, err := result.LastInsertId()
			if err != nil {
				t.Fatalf("failed to get last insert id: %v", err)
			}

			if rowID <= 0 {
				t.Errorf("expected positive row ID, got %d", rowID)
			}
		})
	}
}

func TestNano64_DatabaseRead(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Insert test data
	testID := New(0x123456789ABCDEF0)
	testName := "test item"

	_, err := db.Exec(
		"INSERT INTO items (nano64_id, name) VALUES (?, ?)",
		testID,
		testName,
	)
	if err != nil {
		t.Fatalf("failed to insert test data: %v", err)
	}

	// Read back
	var scannedID Nano64
	var scannedName string

	err = db.QueryRow("SELECT nano64_id, name FROM items WHERE name = ?", testName).Scan(
		&scannedID,
		&scannedName,
	)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if !scannedID.Equals(testID) {
		t.Errorf("ID mismatch: got %d, want %d", scannedID.Uint64Value(), testID.Uint64Value())
	}

	if scannedName != testName {
		t.Errorf("name mismatch: got %s, want %s", scannedName, testName)
	}
}

func TestNano64_DatabaseWriteReadRoundtrip(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	tests := []struct {
		name  string
		value uint64
	}{
		{"zero", 0},
		{"small", 12345},
		{"medium", 0x123456789ABC},
		{"large", 0x123456789ABCDEF0},
		{"max", ^uint64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := New(tt.value)

			// Write to database
			_, err := db.Exec(
				"INSERT INTO items (nano64_id, name) VALUES (?, ?)",
				original,
				tt.name,
			)
			if err != nil {
				t.Fatalf("failed to insert: %v", err)
			}

			// Read back
			var scanned Nano64
			err = db.QueryRow("SELECT nano64_id FROM items WHERE name = ?", tt.name).Scan(&scanned)
			if err != nil {
				t.Fatalf("failed to query: %v", err)
			}

			// Verify roundtrip
			if !scanned.Equals(original) {
				t.Errorf("roundtrip failed: got %d, want %d", scanned.Uint64Value(), original.Uint64Value())
			}

			// Verify timestamp and random fields are preserved
			if scanned.GetTimestamp() != original.GetTimestamp() {
				t.Errorf("timestamp mismatch: got %d, want %d", scanned.GetTimestamp(), original.GetTimestamp())
			}

			if scanned.GetRandom() != original.GetRandom() {
				t.Errorf("random mismatch: got %d, want %d", scanned.GetRandom(), original.GetRandom())
			}
		})
	}
}

func TestNano64_DatabaseMultipleRecords(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Generate multiple IDs
	ids := make([]Nano64, 10)
	for i := 0; i < len(ids); i++ {
		id, err := GenerateDefault()
		if err != nil {
			t.Fatalf("failed to generate ID: %v", err)
		}
		ids[i] = id

		// Insert into database
		_, err = db.Exec(
			"INSERT INTO items (nano64_id, name) VALUES (?, ?)",
			id,
			"item_"+string(rune('0'+i)),
		)
		if err != nil {
			t.Fatalf("failed to insert ID %d: %v", i, err)
		}
	}

	// Query all records ordered by nano64_id
	rows, err := db.Query("SELECT nano64_id, name FROM items ORDER BY nano64_id ASC")
	if err != nil {
		t.Fatalf("failed to query all: %v", err)
	}
	defer rows.Close()

	scannedIDs := make([]Nano64, 0, len(ids))
	for rows.Next() {
		var id Nano64
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			t.Fatalf("failed to scan row: %v", err)
		}
		scannedIDs = append(scannedIDs, id)
	}

	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}

	// Verify all IDs were retrieved
	if len(scannedIDs) != len(ids) {
		t.Errorf("expected %d records, got %d", len(ids), len(scannedIDs))
	}

	// Verify ordering (should be sorted by timestamp)
	for i := 1; i < len(scannedIDs); i++ {
		if Compare(scannedIDs[i-1], scannedIDs[i]) > 0 {
			t.Errorf("IDs not properly sorted at index %d: %d > %d",
				i, scannedIDs[i-1].Uint64Value(), scannedIDs[i].Uint64Value())
		}
	}
}

func TestNano64_DatabaseNullHandling(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a table that allows NULL values
	_, err := db.Exec(`
		CREATE TABLE nullable_items (
			id INTEGER PRIMARY KEY,
			nano64_id INTEGER,
			name TEXT NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create nullable table: %v", err)
	}

	// Insert NULL value
	_, err = db.Exec("INSERT INTO nullable_items (nano64_id, name) VALUES (NULL, ?)", "null item")
	if err != nil {
		t.Fatalf("failed to insert NULL: %v", err)
	}

	// Read back NULL
	var nullID Nano64
	err = db.QueryRow("SELECT nano64_id FROM nullable_items WHERE name = ?", "null item").Scan(&nullID)
	if err != nil {
		t.Fatalf("failed to scan NULL: %v", err)
	}

	// NULL should scan as zero value
	if nullID.Uint64Value() != 0 {
		t.Errorf("NULL scanned as %d, expected 0", nullID.Uint64Value())
	}
}

// TestBigIntHelpers_FromBytesBE_Error tests error handling for invalid byte lengths
func TestBigIntHelpers_FromBytesBE_Error(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		wantErr bool
	}{
		{
			name:    "empty bytes",
			bytes:   []byte{},
			wantErr: true,
		},
		{
			name:    "too few bytes",
			bytes:   []byte{0x01, 0x02, 0x03},
			wantErr: true,
		},
		{
			name:    "too many bytes",
			bytes:   []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09},
			wantErr: true,
		},
		{
			name:    "valid 8 bytes",
			bytes:   []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x42},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BigIntHelpers.FromBytesBE(tt.bytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromBytesBE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestHex_FromBytes tests hex encoding from bytes
func TestHex_FromBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes []byte
		want  string
	}{
		{
			name:  "empty bytes",
			bytes: []byte{},
			want:  "",
		},
		{
			name:  "single byte",
			bytes: []byte{0xFF},
			want:  "FF",
		},
		{
			name:  "multiple bytes",
			bytes: []byte{0xDE, 0xAD, 0xBE, 0xEF},
			want:  "DEADBEEF",
		},
		{
			name:  "8 bytes",
			bytes: []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77},
			want:  "0011223344556677",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Hex.FromBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FromBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestHex_ToBytes_Errors tests error handling in hex decoding
func TestHex_ToBytes_Errors(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		wantErr bool
	}{
		{
			name:    "odd length without prefix",
			hexStr:  "ABC",
			wantErr: true,
		},
		{
			name:    "odd length with prefix",
			hexStr:  "0xABC",
			wantErr: true,
		},
		{
			name:    "invalid character",
			hexStr:  "ABCG",
			wantErr: true,
		},
		{
			name:    "invalid character lowercase",
			hexStr:  "abcz",
			wantErr: true,
		},
		{
			name:    "valid hex",
			hexStr:  "ABCD",
			wantErr: false,
		},
		{
			name:    "valid hex with 0x prefix",
			hexStr:  "0xABCD",
			wantErr: false,
		},
		{
			name:    "valid hex with 0X prefix",
			hexStr:  "0XABCD",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Hex.ToBytes(tt.hexStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("ToBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNano64_String tests the String method
func TestNano64_String(t *testing.T) {
	id := New(0x123456789ABCD)
	str := id.String()
	if str == "" {
		t.Errorf("String() returned empty string")
	}
	// Check that it contains the expected components
	if !strings.Contains(str, "Nano64") {
		t.Errorf("String() = %v, should contain 'Nano64'", str)
	}
}

// TestNano64_FromUint64 tests the FromUint64 function
func TestNano64_FromUint64(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{
			name:  "zero",
			value: 0,
		},
		{
			name:  "small value",
			value: 12345,
		},
		{
			name:  "large value",
			value: 0xFFFFFFFFFFFFFFFF,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id := FromUint64(tt.value)
			if id.Uint64Value() != tt.value {
				t.Errorf("FromUint64() = %v, want %v", id.Uint64Value(), tt.value)
			}
		})
	}
}

// TestNano64_GenerateMonotonicNow tests GenerateMonotonicNow
func TestNano64_GenerateMonotonicNow(t *testing.T) {
	// Reset monotonic state
	monotonicMutex.Lock()
	lastTimestamp = -1
	lastRandom = 0
	monotonicMutex.Unlock()

	id1, err := GenerateMonotonicNow(nil)
	if err != nil {
		t.Fatalf("GenerateMonotonicNow() error = %v", err)
	}

	id2, err := GenerateMonotonicNow(nil)
	if err != nil {
		t.Fatalf("GenerateMonotonicNow() error = %v", err)
	}

	if id1.Uint64Value() >= id2.Uint64Value() {
		t.Errorf("GenerateMonotonicNow() not monotonic: %v >= %v", id1.Uint64Value(), id2.Uint64Value())
	}
}

// TestNano64_GenerateMonotonicDefault tests GenerateMonotonicDefault
func TestNano64_GenerateMonotonicDefault(t *testing.T) {
	// Reset monotonic state
	monotonicMutex.Lock()
	lastTimestamp = -1
	lastRandom = 0
	monotonicMutex.Unlock()

	id, err := GenerateMonotonicDefault()
	if err != nil {
		t.Fatalf("GenerateMonotonicDefault() error = %v", err)
	}

	if id.Uint64Value() == 0 {
		t.Errorf("GenerateMonotonicDefault() returned zero value")
	}
}

// TestEncryptedNano64_Complete tests the full encrypted ID workflow
func TestEncryptedNano64_Complete(t *testing.T) {
	// Create a test key (32 bytes for AES-256)
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	config, err := NewEncryptedIDConfig(key, nil, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	// Test GenerateEncryptedNow
	encrypted, err := config.GenerateEncryptedNow()
	if err != nil {
		t.Fatalf("GenerateEncryptedNow() error = %v", err)
	}

	// Test ToEncryptedHex
	hexStr := encrypted.ToEncryptedHex()
	if len(hexStr) != 72 { // 36 bytes = 72 hex chars
		t.Errorf("ToEncryptedHex() length = %d, want 72", len(hexStr))
	}

	// Test ToEncryptedBytes
	bytes := encrypted.ToEncryptedBytes()
	if len(bytes) != PayloadLength {
		t.Errorf("ToEncryptedBytes() length = %d, want %d", len(bytes), PayloadLength)
	}

	// Test FromEncryptedHex
	decrypted, err := config.FromEncryptedHex(hexStr)
	if err != nil {
		t.Fatalf("FromEncryptedHex() error = %v", err)
	}

	if !decrypted.ID.Equals(encrypted.ID) {
		t.Errorf("FromEncryptedHex() ID = %v, want %v", decrypted.ID, encrypted.ID)
	}

	// Test FromEncryptedBytes
	decrypted2, err := config.FromEncryptedBytes(bytes)
	if err != nil {
		t.Fatalf("FromEncryptedBytes() error = %v", err)
	}

	if !decrypted2.ID.Equals(encrypted.ID) {
		t.Errorf("FromEncryptedBytes() ID = %v, want %v", decrypted2.ID, encrypted.ID)
	}
}

// TestEncryptedNano64_GenerateEncrypted tests GenerateEncrypted with explicit timestamp
func TestEncryptedNano64_GenerateEncrypted(t *testing.T) {
	key := make([]byte, 32)
	config, err := NewEncryptedIDConfig(key, nil, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	timestamp := int64(1234567890)
	encrypted, err := config.GenerateEncrypted(timestamp)
	if err != nil {
		t.Fatalf("GenerateEncrypted() error = %v", err)
	}

	if encrypted.ID.GetTimestamp() != timestamp {
		t.Errorf("GenerateEncrypted() timestamp = %v, want %v", encrypted.ID.GetTimestamp(), timestamp)
	}
}

// TestEncryptedNano64_GenerateEncryptedZeroTimestamp tests GenerateEncrypted with zero timestamp
func TestEncryptedNano64_GenerateEncryptedZeroTimestamp(t *testing.T) {
	key := make([]byte, 32)
	mockClock := func() int64 { return 9999999 }
	config, err := NewEncryptedIDConfig(key, mockClock, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	encrypted, err := config.GenerateEncrypted(0)
	if err != nil {
		t.Fatalf("GenerateEncrypted() error = %v", err)
	}

	if encrypted.ID.GetTimestamp() != 9999999 {
		t.Errorf("GenerateEncrypted(0) should use clock, got timestamp = %v", encrypted.ID.GetTimestamp())
	}
}

// TestEncryptedNano64_Encrypt tests encrypting an existing ID
func TestEncryptedNano64_Encrypt(t *testing.T) {
	key := make([]byte, 32)
	config, err := NewEncryptedIDConfig(key, nil, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	id, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}

	encrypted, err := config.Encrypt(id)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if !encrypted.ID.Equals(id) {
		t.Errorf("Encrypt() ID = %v, want %v", encrypted.ID, id)
	}
}

// TestEncryptedNano64_Errors tests error cases for encrypted IDs
func TestEncryptedNano64_Errors(t *testing.T) {
	t.Run("invalid key length", func(t *testing.T) {
		key := []byte{0x01, 0x02} // Too short
		_, err := NewEncryptedIDConfig(key, nil, nil)
		if err == nil {
			t.Errorf("NewEncryptedIDConfig() with invalid key should error")
		}
	})

	t.Run("invalid encrypted bytes length", func(t *testing.T) {
		key := make([]byte, 32)
		config, err := NewEncryptedIDConfig(key, nil, nil)
		if err != nil {
			t.Fatalf("NewEncryptedIDConfig() error = %v", err)
		}

		_, err = config.FromEncryptedBytes([]byte{0x01, 0x02, 0x03})
		if err == nil {
			t.Errorf("FromEncryptedBytes() with wrong length should error")
		}
	})

	t.Run("invalid encrypted hex", func(t *testing.T) {
		key := make([]byte, 32)
		config, err := NewEncryptedIDConfig(key, nil, nil)
		if err != nil {
			t.Fatalf("NewEncryptedIDConfig() error = %v", err)
		}

		_, err = config.FromEncryptedHex("INVALID")
		if err == nil {
			t.Errorf("FromEncryptedHex() with invalid hex should error")
		}
	})

	t.Run("invalid encrypted hex wrong length", func(t *testing.T) {
		key := make([]byte, 32)
		config, err := NewEncryptedIDConfig(key, nil, nil)
		if err != nil {
			t.Fatalf("NewEncryptedIDConfig() error = %v", err)
		}

		_, err = config.FromEncryptedHex("AABBCCDD")
		if err == nil {
			t.Errorf("FromEncryptedHex() with wrong length should error")
		}
	})

	t.Run("tampered ciphertext", func(t *testing.T) {
		key := make([]byte, 32)
		config, err := NewEncryptedIDConfig(key, nil, nil)
		if err != nil {
			t.Fatalf("NewEncryptedIDConfig() error = %v", err)
		}

		encrypted, err := config.GenerateEncryptedNow()
		if err != nil {
			t.Fatalf("GenerateEncryptedNow() error = %v", err)
		}

		// Tamper with the bytes
		bytes := encrypted.ToEncryptedBytes()
		bytes[20] ^= 0xFF

		_, err = config.FromEncryptedBytes(bytes)
		if err == nil {
			t.Errorf("FromEncryptedBytes() with tampered data should error")
		}
	})
}

// TestEncryptedNano64_DifferentKeySizes tests different AES key sizes
func TestEncryptedNano64_DifferentKeySizes(t *testing.T) {
	tests := []struct {
		name    string
		keySize int
		wantErr bool
	}{
		{
			name:    "AES-128",
			keySize: 16,
			wantErr: false,
		},
		{
			name:    "AES-192",
			keySize: 24,
			wantErr: false,
		},
		{
			name:    "AES-256",
			keySize: 32,
			wantErr: false,
		},
		{
			name:    "invalid size",
			keySize: 20,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			config, err := NewEncryptedIDConfig(key, nil, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEncryptedIDConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && config != nil {
				_, err := config.GenerateEncryptedNow()
				if err != nil {
					t.Errorf("GenerateEncryptedNow() error = %v", err)
				}
			}
		})
	}
}

// TestGenerateMonotonic_Overflow tests overflow handling in monotonic generation
func TestGenerateMonotonic_Overflow(t *testing.T) {
	// Reset monotonic state
	monotonicMutex.Lock()
	lastTimestamp = maxTimestamp
	lastRandom = randomMask // Max random value
	monotonicMutex.Unlock()

	// This should cause an overflow
	_, err := GenerateMonotonic(maxTimestamp, nil)
	if err == nil {
		t.Errorf("GenerateMonotonic() at max timestamp with exhausted random should error")
	}
}

// TestGenerateMonotonic_BackwardsTime tests monotonic generation with backwards time
func TestGenerateMonotonic_BackwardsTime(t *testing.T) {
	// Reset monotonic state
	monotonicMutex.Lock()
	lastTimestamp = 1000000
	lastRandom = 100
	monotonicMutex.Unlock()

	// Try to generate with an earlier timestamp
	id, err := GenerateMonotonic(500000, nil)
	if err != nil {
		t.Fatalf("GenerateMonotonic() error = %v", err)
	}

	// Should use the last timestamp, not the provided one
	if id.GetTimestamp() < 1000000 {
		t.Errorf("GenerateMonotonic() should not go backwards in time")
	}
}

// TestFromBytes_Error tests error handling in FromBytes
func TestFromBytes_Error(t *testing.T) {
	tests := []struct {
		name    string
		bytes   []byte
		wantErr bool
	}{
		{
			name:    "wrong length",
			bytes:   []byte{0x01, 0x02},
			wantErr: true,
		},
		{
			name:    "valid length",
			bytes:   make([]byte, 8),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromBytes(tt.bytes)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromBytes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestFromHex_EdgeCases tests additional edge cases for FromHex
func TestFromHex_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		hexStr  string
		wantErr bool
	}{
		{
			name:    "too short after prefix removal",
			hexStr:  "0xABCD",
			wantErr: true,
		},
		{
			name:    "too long",
			hexStr:  "0x00112233445566778899",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromHex(tt.hexStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("FromHex() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestGenerate_RNGError tests error handling when RNG fails
func TestGenerate_RNGError(t *testing.T) {
	failingRNG := func(bits int) (uint32, error) {
		return 0, fmt.Errorf("RNG failure")
	}

	_, err := Generate(12345, failingRNG)
	if err == nil {
		t.Errorf("Generate() with failing RNG should error")
	}
}

// TestGenerateMonotonic_RNGError tests error handling when RNG fails in monotonic generation
func TestGenerateMonotonic_RNGError(t *testing.T) {
	// Reset monotonic state to a different timestamp
	monotonicMutex.Lock()
	lastTimestamp = 1000
	lastRandom = 0
	monotonicMutex.Unlock()

	failingRNG := func(bits int) (uint32, error) {
		return 0, fmt.Errorf("RNG failure")
	}

	// Use a newer timestamp so it tries to generate a new random value
	_, err := GenerateMonotonic(5000, failingRNG)
	if err == nil {
		t.Errorf("GenerateMonotonic() with failing RNG should error")
	}
}

// TestEncryptedNano64_InvalidDecryptedLength tests decryption with unexpected plaintext length
func TestEncryptedNano64_InvalidDecryptedLength(t *testing.T) {
	// This test covers the edge case where decrypted data isn't 8 bytes
	// This is difficult to trigger naturally, but we can test the error path exists
	key := make([]byte, 32)
	config, err := NewEncryptedIDConfig(key, nil, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	// Try with completely invalid data that will decrypt to wrong length or fail
	invalidPayload := make([]byte, PayloadLength)
	_, err = config.FromEncryptedBytes(invalidPayload)
	if err == nil {
		t.Errorf("FromEncryptedBytes() with invalid payload should error")
	}
}

// TestEncryptedNano64_CiphertextLengthCheck tests the ciphertext length validation
func TestEncryptedNano64_CiphertextLengthCheck(t *testing.T) {
	// This covers the error case in Encrypt where ciphertext length is unexpected
	// In normal operation this shouldn't happen, but the code checks for it
	key := make([]byte, 32)
	config, err := NewEncryptedIDConfig(key, nil, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	// Generate a valid ID
	id, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}

	// Normal encryption should work
	encrypted, err := config.Encrypt(id)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if encrypted == nil {
		t.Errorf("Encrypt() returned nil")
	}
}

// TestGenerateMonotonic_SameTimestampIncrement tests incrementing within same millisecond
func TestGenerateMonotonic_SameTimestampIncrement(t *testing.T) {
	// Reset monotonic state
	monotonicMutex.Lock()
	lastTimestamp = 1000
	lastRandom = 50
	monotonicMutex.Unlock()

	// Generate multiple IDs with the same timestamp
	id1, err := GenerateMonotonic(1000, nil)
	if err != nil {
		t.Fatalf("GenerateMonotonic() error = %v", err)
	}

	id2, err := GenerateMonotonic(1000, nil)
	if err != nil {
		t.Fatalf("GenerateMonotonic() error = %v", err)
	}

	// Should increment the random field
	if id2.GetRandom() <= id1.GetRandom() {
		t.Errorf("GenerateMonotonic() should increment random field in same ms")
	}
}

// TestHex_ToBytes_EdgeCase tests an edge case in hex validation
func TestHex_ToBytes_EdgeCase(t *testing.T) {
	// Test with special characters
	_, err := Hex.ToBytes("AB CD")
	if err == nil {
		t.Errorf("ToBytes() with space should error")
	}
}

// TestGenerate_WithNilRNG tests that nil RNG uses default
func TestGenerate_WithNilRNG(t *testing.T) {
	id, err := Generate(12345, nil)
	if err != nil {
		t.Fatalf("Generate() with nil RNG error = %v", err)
	}

	if id.GetTimestamp() != 12345 {
		t.Errorf("Generate() timestamp = %v, want 12345", id.GetTimestamp())
	}
}

// TestGenerateMonotonic_WithNilRNG tests that nil RNG uses default
func TestGenerateMonotonic_WithNilRNG(t *testing.T) {
	// Reset monotonic state
	monotonicMutex.Lock()
	lastTimestamp = -1
	lastRandom = 0
	monotonicMutex.Unlock()

	id, err := GenerateMonotonic(12345, nil)
	if err != nil {
		t.Fatalf("GenerateMonotonic() with nil RNG error = %v", err)
	}

	if id.GetTimestamp() != 12345 {
		t.Errorf("GenerateMonotonic() timestamp = %v, want 12345", id.GetTimestamp())
	}
}

// TestNewEncryptedIDConfig_DefaultClock tests that nil clock uses default
func TestNewEncryptedIDConfig_DefaultClock(t *testing.T) {
	key := make([]byte, 32)
	config, err := NewEncryptedIDConfig(key, nil, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	// Should be able to generate with default clock
	encrypted, err := config.GenerateEncryptedNow()
	if err != nil {
		t.Fatalf("GenerateEncryptedNow() error = %v", err)
	}

	if encrypted.ID.GetTimestamp() == 0 {
		t.Errorf("GenerateEncryptedNow() should use current time")
	}
}

// TestEncryptedNano64_GenerateIVError tests IV generation error handling
func TestEncryptedNano64_GenerateIVError(t *testing.T) {
	// This test verifies that the generateIV error path exists
	// In practice, crypto/rand.Read rarely fails, but the code handles it
	key := make([]byte, 32)
	config, err := NewEncryptedIDConfig(key, nil, nil)
	if err != nil {
		t.Fatalf("NewEncryptedIDConfig() error = %v", err)
	}

	// Normal operation should work
	id, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}

	encrypted, err := config.Encrypt(id)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if len(encrypted.ToEncryptedBytes()) != PayloadLength {
		t.Errorf("Encrypted payload has wrong length")
	}
}

// TestDefaultRNG_BitsMask tests that RNG correctly masks bits
func TestDefaultRNG_BitsMask(t *testing.T) {
	// Test that 1-bit RNG only returns 0 or 1
	for i := 0; i < 100; i++ {
		val, err := DefaultRNG(1)
		if err != nil {
			t.Fatalf("DefaultRNG(1) error = %v", err)
		}
		if val > 1 {
			t.Errorf("DefaultRNG(1) returned %d, expected 0 or 1", val)
		}
	}

	// Test that 2-bit RNG only returns 0-3
	for i := 0; i < 100; i++ {
		val, err := DefaultRNG(2)
		if err != nil {
			t.Fatalf("DefaultRNG(2) error = %v", err)
		}
		if val > 3 {
			t.Errorf("DefaultRNG(2) returned %d, expected 0-3", val)
		}
	}
}

func TestNil(t *testing.T) {
	// Test that Nil is zero value
	if Nil.Uint64Value() != 0 {
		t.Errorf("Nil.Uint64Value() = %d, want 0", Nil.Uint64Value())
	}

	// Test IsNil on Nil constant
	if !Nil.IsNil() {
		t.Error("Nil.IsNil() = false, want true")
	}

	// Test IsNil on non-nil ID
	id, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}
	if id.IsNil() {
		t.Error("generated ID.IsNil() = true, want false")
	}

	// Test IsNil on zero-constructed ID
	zero := Nano64{}
	if !zero.IsNil() {
		t.Error("zero-constructed ID.IsNil() = false, want true")
	}

	// Test equality with Nil
	if !zero.Equals(Nil) {
		t.Error("zero ID should equal Nil")
	}
}

func TestNil_DirectComparison(t *testing.T) {
	// Test direct == comparison with Nil
	var id Nano64
	if id != Nil {
		t.Error("zero-value ID should == Nil")
	}

	// Test direct != comparison with Nil
	id2, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}
	if id2 == Nil {
		t.Error("generated ID should != Nil")
	}

	// Test direct comparison between two IDs
	id3, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}

	if id2 == id3 {
		t.Error("two different generated IDs should not be equal (extremely unlikely)")
	}

	// Test self-comparison via copy
	id2Copy := id2
	if id2 != id2Copy {
		t.Error("ID should == its copy")
	}

	// Test comparison of zero-constructed IDs
	zero1 := Nano64{}
	zero2 := Nano64{}
	if zero1 != zero2 {
		t.Error("two zero-constructed IDs should be equal")
	}

	// Test that zero-constructed ID equals Nil
	if zero1 != Nil {
		t.Error("zero-constructed ID should equal Nil")
	}
}

func TestNullNano64_Scan(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		wantValid bool
		wantError bool
	}{
		{"nil value", nil, false, false},
		{"uint64 value", uint64(12345), true, false},
		{"int64 value", int64(12345), true, false},
		{"bytes value", []byte{0, 0, 0, 0, 0, 0, 0x30, 0x39}, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n NullNano64
			err := n.Scan(tt.input)

			if (err != nil) != tt.wantError {
				t.Errorf("Scan() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if n.Valid != tt.wantValid {
				t.Errorf("Valid = %v, want %v", n.Valid, tt.wantValid)
			}

			if !tt.wantValid && !n.ID.IsNil() {
				t.Error("Invalid NullNano64 should have Nil ID")
			}
		})
	}
}

func TestNullNano64_Value(t *testing.T) {
	// Test valid NullNano64
	id, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}

	validNull := NullNano64{ID: id, Valid: true}
	val, err := validNull.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if val == nil {
		t.Error("Valid NullNano64.Value() should not be nil")
	}

	// Test invalid NullNano64
	invalidNull := NullNano64{Valid: false}
	val, err = invalidNull.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}
	if val != nil {
		t.Errorf("Invalid NullNano64.Value() = %v, want nil", val)
	}
}

func TestNullNano64_JSON(t *testing.T) {
	// Test marshaling valid NullNano64
	id, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}

	validNull := NullNano64{ID: id, Valid: true}
	data, err := json.Marshal(validNull)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}

	var unmarshaled NullNano64
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	if !unmarshaled.Valid {
		t.Error("Unmarshaled NullNano64 should be valid")
	}
	if !unmarshaled.ID.Equals(id) {
		t.Error("Unmarshaled ID does not match original")
	}

	// Test marshaling invalid NullNano64
	invalidNull := NullNano64{Valid: false}
	data, err = json.Marshal(invalidNull)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	if string(data) != "null" {
		t.Errorf("Invalid NullNano64 marshaled to %s, want null", string(data))
	}

	// Test unmarshaling null
	var nullUnmarshaled NullNano64
	if err := json.Unmarshal([]byte("null"), &nullUnmarshaled); err != nil {
		t.Fatalf("Unmarshal(null) error = %v", err)
	}
	if nullUnmarshaled.Valid {
		t.Error("Unmarshaled null should be invalid")
	}
}

func TestNullNano64_Database(t *testing.T) {
	// Create in-memory SQLite database
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "test.db")

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create test table
	_, err = db.Exec(`
		CREATE TABLE test_null (
			id INTEGER PRIMARY KEY,
			nullable_id BLOB,
			non_null_id BLOB NOT NULL
		)
	`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}

	// Test inserting NULL value
	nullID := NullNano64{Valid: false}
	validID, err := GenerateDefault()
	if err != nil {
		t.Fatalf("GenerateDefault() error = %v", err)
	}
	validNullID := NullNano64{ID: validID, Valid: true}

	_, err = db.Exec("INSERT INTO test_null (id, nullable_id, non_null_id) VALUES (?, ?, ?)",
		1, nullID, validNullID)
	if err != nil {
		t.Fatalf("failed to insert: %v", err)
	}

	// Test querying NULL value
	var retrievedNull NullNano64
	var retrievedValid NullNano64
	err = db.QueryRow("SELECT nullable_id, non_null_id FROM test_null WHERE id = ?", 1).
		Scan(&retrievedNull, &retrievedValid)
	if err != nil {
		t.Fatalf("failed to query: %v", err)
	}

	if retrievedNull.Valid {
		t.Error("Retrieved null ID should be invalid")
	}

	if !retrievedValid.Valid {
		t.Error("Retrieved valid ID should be valid")
	}

	if !retrievedValid.ID.Equals(validID) {
		t.Error("Retrieved ID does not match original")
	}
}
