# Nano64 Examples

This directory contains example programs demonstrating various features of the Nano64 library.

## Examples

### Basic Usage (`basic/`)

A simple example showing basic ID generation and common operations.

```bash
go run internal/examples/basic/main.go
```

### Monotonic Generation (`monotonic/`)

Demonstrates monotonic ID generation which ensures strictly increasing IDs even when generated in the same millisecond. This example shows:

- Basic monotonic generation with sequential IDs
- Strict ordering guarantees across 1000+ IDs
- Comparison between monotonic and non-monotonic generation
- Per-millisecond sequence behavior
- Automatic timestamp transitions

```bash
go run internal/examples/monotonic/main.go
```

### Collision Resistance Demonstration (`collision-resistance/`)

A comprehensive demonstration of Nano64's collision resistance properties. This program:

- Generates **5 million IDs** in single-threaded mode at maximum speed
- Generates **5 million IDs** across 10 concurrent goroutines
- Maintains a sustained rate of **145,000 IDs/second** for 10 seconds
- Performs a maximum throughput stress test

The program tracks:

- Total IDs generated
- Generation rate (IDs per second)
- Number of collisions
- Collision percentage
- Maximum IDs generated in a single millisecond
- Per-millisecond statistics

```bash
go run internal/examples/collision-resistance/main.go
```

**Expected Results:**

At rates around 145,000 IDs/second (the theoretical 1% collision threshold), you should observe:

- Very low collision rates (typically < 0.2%)
- Excellent performance (millions of IDs per second possible)
- Collision probability increases with IDs per millisecond

The 20-bit random field provides 1,048,576 unique values per millisecond, making collisions extremely rare under normal load.

## Running All Examples

You can run all examples from the repository root:

```bash
# Basic example
go run ./internal/examples/basic/main.go

# Monotonic generation
go run ./internal/examples/monotonic/main.go

# Collision resistance test
go run ./internal/examples/collision-resistance/main.go
```
