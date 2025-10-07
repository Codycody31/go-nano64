# Nano64 - 64‑bit Time‑Sortable Identifiers for Go

**Nano64** is a lightweight library for generating time-sortable, globally unique IDs that offer the same practical guarantees as ULID or UUID in half the storage footprint; reducing index and I/O overhead while preserving cryptographic-grade randomness. Includes optional monotonic sequencing and AES-GCM encryption.

[![Go Reference](https://pkg.go.dev/badge/go.codycody31.dev/nano64.svg)](https://pkg.go.dev/go.codycody31.dev/nano64)
[![Go Report Card](https://goreportcard.com/badge/go.codycody31.dev/nano64)](https://goreportcard.com/report/go.codycody31.dev/nano64)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

> **Note:** This is a Go port of the original [Nano64 TypeScript/JavaScript library](https://github.com/only-cliches/nano64) by [@only-cliches](https://github.com/only-cliches). All credit for the original concept, design, and implementation goes to the original author. This port aims to bring the same powerful, compact ID generation capabilities to the Go ecosystem.

---

## Features

* **Time‑sortable:** IDs order by creation time automatically.
* **Compact:** 8 bytes / 16 hex characters.
* **Deterministic format:** `[63‥20]=timestamp`, `[19‥0]=random`.
* **Collision‑resistant:** ~1% colllision risk at 145,000 IDs per second.
* **Cross‑database‑safe:** Big‑endian bytes preserve order in SQLite, Postgres, MySQL, etc.
* **AES-GCM encryption:** Optional encryption masks the embedded creation date.
* **Unsigned canonical form:** Single, portable representation (0..2⁶⁴‑1).

---

## Installation

```bash
go get go.codycody31.dev/nano64
```

---

## Usage

### Basic ID generation

```go
import (
    "fmt"
    "go.codycody31.dev/nano64"
)

func main() {
    id, err := nano64.GenerateDefault()
    if err != nil {
        panic(err)
    }
    
    fmt.Println(id.ToHex())        // 17‑char uppercase hex TIMESTAMP-RANDOM
    // 199C01B6659-5861C
    fmt.Println(id.ToBytes())      // [8]byte
    // [25 156 1 182 101 149 134 28]
    fmt.Println(id.GetTimestamp()) // ms since epoch
    // 1759864645209
}
```

### Monotonic generation

Ensures strictly increasing values even if created in the same millisecond.

```go
a, err := nano64.GenerateMonotonicDefault()
if err != nil {
    panic(err)
}

b, err := nano64.GenerateMonotonicDefault()
if err != nil {
    panic(err)
}

fmt.Println(nano64.Compare(a, b)) // -1
```

### AES‑GCM encryption

IDs can easily be encrypted and decrypted to mask their timestamp value from public view.

```go
import "go.codycody31.dev/nano64"

// Create encryption key (32 bytes for AES-256)
key := make([]byte, 32)
if _, err := rand.Read(key); err != nil {
    panic(err)
}

factory := nano64.NewEncryptedFactory(key)

// Generate and encrypt
wrapped, err := factory.GenerateEncrypted()
if err != nil {
    panic(err)
}

fmt.Println(wrapped.ID.ToHex())           // Unencrypted ID
// 199C01B66F8-CB911
fmt.Println(wrapped.ToEncryptedHex())     // 72‑char hex payload
// 2D5CEBF218C569DDE077C4C1F247C708063BAA93B4285CD67D53327EA4C374A64395CFF0

// Decrypt later
restored, err := factory.FromEncryptedHex(wrapped.ToEncryptedHex())
if err != nil {
    panic(err)
}

fmt.Println(restored.ID.Uint64Value() == wrapped.ID.Uint64Value()) // true
```

### Database primary key storage

The `Nano64` type implements `database/sql/driver.Valuer` and `sql.Scanner` interfaces for seamless database integration.

Store `id.ToBytes()` as an **8‑byte big‑endian binary** value, or use the built-in SQL support:

```go
import (
    "database/sql"
    "go.codycody31.dev/nano64"
)

type User struct {
    ID   nano64.Nano64
    Name string
}

// Insert
id, _ := nano64.GenerateDefault()
_, err := db.Exec("INSERT INTO users (id, name) VALUES (?, ?)", id, "Alice")

// Query
var user User
err = db.QueryRow("SELECT id, name FROM users WHERE id = ?", id).Scan(&user.ID, &user.Name)
```

**Database compatibility:**

| DBMS        | Column Type       | Preserves Order | Notes                                                                  |
| ----------- | ----------------- | --------------- | ---------------------------------------------------------------------- |
| SQLite      | `BLOB` (8 bytes)  | ✅              | Lexicographic byte order matches unsigned big-endian.                  |
| PostgreSQL  | `BYTEA` (8 bytes) | ✅              | `PRIMARY KEY` on `BYTEA` is fine.                                      |
| MySQL 8+    | `BINARY(8)`       | ✅              | Binary collation.                                                      |
| MariaDB     | `BINARY(8)`       | ✅              | Same as MySQL.                                                         |
| SQL Server  | `BINARY(8)`       | ✅              | Clustered index sorts by bytes.                                        |
| Oracle      | `RAW(8)`          | ✅              | RAW compares bytewise.                                                 |
| CockroachDB | `BYTES` (8)       | ✅              | Bytewise ordering.                                                     |
| DuckDB      | `BLOB` (8)        | ✅              | Bytewise ordering.                                                     |

---

## Comparison with other identifiers

| Property               | **Nano64**                                | **ULID**                    | **UUIDv4**              | **Snowflake ID**             |
| ---------------------- | ----------------------------------------- | --------------------------- | ----------------------- | ---------------------------- |
| Bits total             | 64                                        | 128                         | 128                     | 64                           |
| Encoded timestamp bits | 44                                        | 48                          | 0                       | 41                           |
| Random / entropy bits  | 20                                        | 80                          | 122                     | 22 (per-node sequence)       |
| Sortable by time       | ✅ Yes (lexicographic & numeric)           | ✅ Yes                       | ❌ No                    | ✅ Yes                        |
| Collision risk (1%)    | ~145 IDs/ms                               | ~26M/ms                     | Practically none        | None (central sequence)      |
| Typical string length  | 16 hex chars                              | 26 Crockford base32         | 36 hex+hyphens          | 18–20 decimal digits         |
| Encodes creation time  | ✅                                        | ✅                           | ❌                       | ✅                            |
| Can hide timestamp     | ✅ via AES-GCM encryption                  | ⚠️ Not built-in             | ✅ (no time field)       | ❌ Not by design              |
| Database sort order    | ✅ Stable with big-endian BLOB             | ✅ (lexical)                 | ❌ Random                | ✅ Numeric                    |
| Cryptographic strength | 20-bit random, optional AES               | 80-bit random               | 122-bit random          | None (deterministic)         |
| Dependencies           | None (crypto optional)                    | None                        | None                    | Central service or worker ID |
| Target use             | Compact, sortable, optionally private IDs | Human-readable sortable IDs | Pure random identifiers | Distributed service IDs      |

---

## API Summary

### Generation Functions

* **`Generate(timestamp int64, rng RNG) (Nano64, error)`** - Creates a new ID with specified timestamp and RNG
* **`GenerateNow(rng RNG) (Nano64, error)`** - Creates an ID with current timestamp
* **`GenerateDefault() (Nano64, error)`** - Creates an ID with current timestamp and default RNG
* **`GenerateMonotonic(timestamp int64, rng RNG) (Nano64, error)`** - Creates monotonic ID (strictly increasing)
* **`GenerateMonotonicNow(rng RNG) (Nano64, error)`** - Creates monotonic ID with current timestamp
* **`GenerateMonotonicDefault() (Nano64, error)`** - Creates monotonic ID with current timestamp and default RNG

### Parsing Functions

* **`FromHex(hex string) (Nano64, error)`** - Parse from 16-char hex string (with or without dash)
* **`FromBytes(bytes []byte) (Nano64, error)`** - Parse from 8 big-endian bytes
* **`FromUint64(value uint64) Nano64`** - Create from uint64 value
* **`New(value uint64) Nano64`** - Create from uint64 value (alias)

### ID Methods

* **`ToHex() string`** - Returns 17-char uppercase hex (TIMESTAMP-RANDOM)
* **`ToBytes() []byte`** - Returns 8-byte big-endian encoding
* **`ToDate() time.Time`** - Converts embedded timestamp to time.Time
* **`GetTimestamp() int64`** - Extracts embedded millisecond timestamp
* **`GetRandom() uint32`** - Extracts 20-bit random field
* **`Uint64Value() uint64`** - Returns raw uint64 value

### Comparison Functions

* **`Compare(a, b Nano64) int`** - Compare two IDs (-1, 0, 1)
* **`Equals(other Nano64) bool`** - Check equality

### Database Support

* **`Value() (driver.Value, error)`** - Implements `driver.Valuer` for SQL storage
* **`Scan(value interface{}) error`** - Implements `sql.Scanner` for SQL retrieval

### Encrypted IDs

* **`NewEncryptedFactory(key []byte) *EncryptedFactory`** - Create factory with 32-byte AES-256 key
* **`factory.GenerateEncrypted() (*EncryptedNano64, error)`** - Generate and encrypt ID
* **`factory.Encrypt(id Nano64) (*EncryptedNano64, error)`** - Encrypt existing ID
* **`factory.FromEncryptedHex(hex string) (*EncryptedNano64, error)`** - Decrypt from hex
* **`factory.FromEncryptedBytes(bytes []byte) (*EncryptedNano64, error)`** - Decrypt from bytes

---

## Design

| Bits | Field          | Purpose             | Range                 |
| ---- | -------------- | ------------------- | --------------------- |
| 44   | Timestamp (ms) | Chronological order | 1970–2527             |
| 20   | Random         | Collision avoidance | 1,048,576 patterns/ms |

Collision probability ≈ 1% if ~145 IDs generated in one millisecond.

---

## Examples

The [`internal/examples`](internal/examples/) directory contains comprehensive examples:

* **[Basic Usage](internal/examples/basic/)** - Simple ID generation and operations
* **[Monotonic Generation](internal/examples/monotonic/)** - Demonstrates strictly increasing IDs with per-millisecond sequencing
* **[Collision Resistance](internal/examples/collision-resistance/)** - Demonstrates collision resistance by generating millions of IDs at high speed (145k+ IDs/second)

Run the collision resistance demonstration:

```bash
go run ./internal/examples/collision-resistance/main.go
```

This will generate 5 million IDs at maximum speed, test concurrent generation across multiple goroutines, maintain a sustained rate of 145,000 IDs/second, and perform stress testing. The demonstration proves that even at very high generation rates, Nano64 maintains excellent collision resistance with typically < 0.2% collision rate.

---

## Tests

Run:

```bash
go test -v
```

All unit tests cover:

* Hex ↔ bytes conversions
* BigInt encoding
* Timestamp extraction and monotonic logic
* AES‑GCM encryption/decryption integrity
* Overflow edge cases
* Database driver.Valuer and sql.Scanner interfaces

---

## License

MIT License

---

## Keywords

nano64, ulid, time-sortable, 64-bit id, bigint, aes-gcm, uid, uuid alternative, distributed id, database key, monotonic id, sortable id, crypto id, go, golang, timestamp id
