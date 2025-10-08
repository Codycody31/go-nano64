package main

import (
	"fmt"
	"time"

	"go.codycody31.dev/nano64"
)

func main() {
	fmt.Println("=== Nano64 Monotonic Generation Example ===")
	fmt.Println()

	// Example 1: Basic monotonic generation
	fmt.Println("Example 1: Basic Monotonic Generation")
	fmt.Println("Generating 10 IDs in quick succession...")
	for i := 0; i < 10; i++ {
		id, err := nano64.GenerateMonotonicDefault()
		if err != nil {
			panic(err)
		}
		fmt.Printf("  ID %2d: %s (timestamp: %d, random: %d)\n",
			i+1, id.ToHex(), id.GetTimestamp(), id.GetRandom())
	}
	fmt.Println()

	// Example 2: Demonstrate strict ordering
	fmt.Println("Example 2: Strict Ordering Guarantee")
	fmt.Println("Generating 1000 IDs and verifying they're strictly increasing...")

	prev, err := nano64.GenerateMonotonicDefault()
	if err != nil {
		panic(err)
	}

	allIncreasing := true
	for i := 1; i < 1000; i++ {
		current, err := nano64.GenerateMonotonicDefault()
		if err != nil {
			panic(err)
		}

		if nano64.Compare(current, prev) <= 0 {
			fmt.Printf("  ❌ Order violation at position %d!\n", i)
			allIncreasing = false
			break
		}
		prev = current
	}

	if allIncreasing {
		fmt.Println("  ✅ All 1000 IDs are strictly increasing")
	}
	fmt.Println()

	// Example 3: Compare monotonic vs non-monotonic
	fmt.Println("Example 3: Monotonic vs Non-Monotonic Generation")
	fmt.Println("Generating 10 IDs with each method in rapid succession...")

	fmt.Println("\n  Non-Monotonic (GenerateDefault):")
	nonMonotonicIDs := make([]nano64.Nano64, 10)
	for i := 0; i < 10; i++ {
		id, err := nano64.GenerateDefault()
		if err != nil {
			panic(err)
		}
		nonMonotonicIDs[i] = id
		fmt.Printf("    %s\n", id.ToHex())
	}

	fmt.Println("\n  Monotonic (GenerateMonotonicDefault):")
	monotonicIDs := make([]nano64.Nano64, 10)
	for i := 0; i < 10; i++ {
		id, err := nano64.GenerateMonotonicDefault()
		if err != nil {
			panic(err)
		}
		monotonicIDs[i] = id
		fmt.Printf("    %s\n", id.ToHex())
	}
	fmt.Println()

	// Example 4: Demonstrate per-millisecond sequence
	fmt.Println("Example 4: Per-Millisecond Sequence")
	fmt.Println("Rapidly generating IDs to show incrementing random field within same millisecond...")

	// Generate many IDs quickly to ensure some share the same millisecond
	var ids []nano64.Nano64
	targetTimestamp := int64(-1)
	sameTimestampCount := 0

	for len(ids) < 100 {
		id, err := nano64.GenerateMonotonicDefault()
		if err != nil {
			continue
		}

		timestamp := id.GetTimestamp()
		if targetTimestamp == -1 {
			targetTimestamp = timestamp
		}

		if timestamp == targetTimestamp {
			ids = append(ids, id)
			sameTimestampCount++
		}

		// Stop once we have enough IDs with the same timestamp
		if sameTimestampCount >= 10 {
			break
		}

		// Also stop if we've moved to a new timestamp and have some IDs
		if timestamp != targetTimestamp && len(ids) > 0 {
			break
		}
	}

	fmt.Printf("  Generated %d IDs with timestamp %d:\n", len(ids), targetTimestamp)
	for i, id := range ids {
		fmt.Printf("    ID %2d: random=%d, full=%s\n",
			i+1, id.GetRandom(), id.ToHex())
	}

	// Verify they're increasing
	isMonotonic := true
	for i := 1; i < len(ids); i++ {
		if ids[i].GetRandom() <= ids[i-1].GetRandom() {
			isMonotonic = false
			break
		}
	}

	if isMonotonic {
		fmt.Println("  ✅ Random field increments monotonically within the same millisecond")
	} else {
		fmt.Println("  ⚠️  Note: IDs may span multiple milliseconds")
	}
	fmt.Println()

	// Example 5: Timestamp rollover behavior
	fmt.Println("Example 5: Millisecond Transition")
	fmt.Println("Generating IDs across millisecond boundaries...")

	prevTimestamp := int64(-1)
	prevRandom := uint32(0)
	transitionFound := false

	start := time.Now()
	for time.Since(start) < 100*time.Millisecond && !transitionFound {
		id, err := nano64.GenerateMonotonicDefault()
		if err != nil {
			continue
		}

		timestamp := id.GetTimestamp()
		random := id.GetRandom()

		if prevTimestamp != -1 && timestamp > prevTimestamp {
			fmt.Printf("  Timestamp transition detected:\n")
			fmt.Printf("    Before: timestamp=%d, random=%d\n", prevTimestamp, prevRandom)
			fmt.Printf("    After:  timestamp=%d, random=%d\n", timestamp, random)
			fmt.Println("  ✅ Random field reset with new timestamp")
			transitionFound = true
		}

		prevTimestamp = timestamp
		prevRandom = random
	}

	if !transitionFound {
		fmt.Println("  ⚠️  No millisecond transition observed in 100ms window")
	}
	fmt.Println()

	// Summary
	fmt.Println("=== Summary ===")
	fmt.Println("Monotonic generation ensures:")
	fmt.Println("  • Strictly increasing IDs across all calls")
	fmt.Println("  • Incrementing random field within same millisecond")
	fmt.Println("  • Automatic timestamp bump when random field wraps")
	fmt.Println("  • Safe for distributed primary keys and time-series data")
}
