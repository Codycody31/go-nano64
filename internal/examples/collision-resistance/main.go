package main

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.codycody31.dev/nano64/v1"
)

// formatNumberWithCommas formats a number with comma separators for readability
func formatNumberWithCommas(n int64) string {
	s := strconv.FormatInt(n, 10)
	parts := []string{}
	for i := len(s); i > 0; i -= 3 {
		start := i - 3
		if start < 0 {
			start = 0
		}
		parts = append([]string{s[start:i]}, parts...)
	}
	return strings.Join(parts, ",")
}

func main() {
	fmt.Println("=== Nano64 Collision Resistance Demonstration ===")
	fmt.Println()

	// Test 1: Generate IDs at high speed and check for collisions
	fmt.Println("Test 1: High-Speed Generation (Single-threaded)")
	fmt.Println("Generating 5 million IDs as fast as possible...")
	testHighSpeedGeneration(5_000_000)
	fmt.Println()

	// Test 2: Generate IDs in concurrent goroutines
	fmt.Println("Test 2: Concurrent Generation")
	fmt.Println("Generating 5 million IDs across 10 goroutines...")
	testConcurrentGeneration(5_000_000, 10)
	fmt.Println()

	// Test 3: Target rate of 145k IDs/second to demonstrate 1% collision probability
	fmt.Println("Test 3: Sustained Rate at 145k IDs/second")
	fmt.Println("Generating IDs at ~145,000 per second for 10 seconds...")
	testSustainedRate(145_000, 10*time.Second)
	fmt.Println()

	// Test 4: Extreme stress test - generate as many as possible in 1 second
	fmt.Println("Test 4: Maximum Throughput (1 second burst)")
	testMaxThroughput(1 * time.Second)
}

// testHighSpeedGeneration generates a large number of IDs and checks for collisions
func testHighSpeedGeneration(count int) {
	seen := make(map[uint64]bool)
	collisions := 0
	start := time.Now()

	for i := 0; i < count; i++ {
		id, err := nano64.GenerateDefault()
		if err != nil {
			panic(err)
		}

		value := id.Uint64Value()
		if seen[value] {
			collisions++
		}
		seen[value] = true
	}

	elapsed := time.Since(start)
	rate := float64(count) / elapsed.Seconds()

	fmt.Printf("  Generated: %s IDs\n", formatNumberWithCommas(int64(count)))
	fmt.Printf("  Time: %v\n", elapsed)
	fmt.Printf("  Rate: %s IDs/second\n", formatNumberWithCommas(int64(rate)))
	fmt.Printf("  Collisions: %s (%.6f%%)\n", formatNumberWithCommas(int64(collisions)), float64(collisions)/float64(count)*100)
	fmt.Printf("  Unique IDs: %s\n", formatNumberWithCommas(int64(len(seen))))
}

// testConcurrentGeneration generates IDs across multiple goroutines
func testConcurrentGeneration(totalCount int, numGoroutines int) {
	seen := sync.Map{}
	var collisions atomic.Int64
	var wg sync.WaitGroup

	countPerGoroutine := totalCount / numGoroutines
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < countPerGoroutine; j++ {
				id, err := nano64.GenerateDefault()
				if err != nil {
					panic(err)
				}

				value := id.Uint64Value()
				if _, exists := seen.LoadOrStore(value, true); exists {
					collisions.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)
	rate := float64(totalCount) / elapsed.Seconds()

	// Count unique IDs
	uniqueCount := 0
	seen.Range(func(key, value interface{}) bool {
		uniqueCount++
		return true
	})

	fmt.Printf("  Generated: %s IDs across %d goroutines\n", formatNumberWithCommas(int64(totalCount)), numGoroutines)
	fmt.Printf("  Time: %v\n", elapsed)
	fmt.Printf("  Rate: %s IDs/second\n", formatNumberWithCommas(int64(rate)))
	fmt.Printf("  Collisions: %s (%.6f%%)\n", formatNumberWithCommas(collisions.Load()), float64(collisions.Load())/float64(totalCount)*100)
	fmt.Printf("  Unique IDs: %s\n", formatNumberWithCommas(int64(uniqueCount)))
}

// testSustainedRate generates IDs at a target rate for a duration
func testSustainedRate(targetRate int, duration time.Duration) {
	seen := make(map[uint64]bool)
	collisions := 0
	totalGenerated := 0

	start := time.Now()
	deadline := start.Add(duration)

	// Track per-millisecond statistics
	msStats := make(map[int64]int)
	maxPerMs := 0

	// Generate IDs in batches to achieve target rate
	batchSize := 1000
	batchInterval := time.Duration(float64(time.Second) / (float64(targetRate) / float64(batchSize)))
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		<-ticker.C

		// Generate a batch of IDs
		for i := 0; i < batchSize && time.Now().Before(deadline); i++ {
			id, err := nano64.GenerateDefault()
			if err != nil {
				continue
			}

			value := id.Uint64Value()
			if seen[value] {
				collisions++
			}
			seen[value] = true
			totalGenerated++

			// Track IDs per millisecond
			timestamp := id.GetTimestamp()
			msStats[timestamp]++
			if msStats[timestamp] > maxPerMs {
				maxPerMs = msStats[timestamp]
			}
		}
	}

	elapsed := time.Since(start)
	actualRate := float64(totalGenerated) / elapsed.Seconds()

	fmt.Printf("  Target Rate: %s IDs/second\n", formatNumberWithCommas(int64(targetRate)))
	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Generated: %s IDs\n", formatNumberWithCommas(int64(totalGenerated)))
	fmt.Printf("  Actual Rate: %s IDs/second\n", formatNumberWithCommas(int64(actualRate)))
	fmt.Printf("  Collisions: %s (%.6f%%)\n", formatNumberWithCommas(int64(collisions)), float64(collisions)/float64(totalGenerated)*100)
	fmt.Printf("  Unique IDs: %s\n", formatNumberWithCommas(int64(len(seen))))
	fmt.Printf("  Max IDs in single millisecond: %s\n", formatNumberWithCommas(int64(maxPerMs)))
	fmt.Printf("  Milliseconds with IDs: %s\n", formatNumberWithCommas(int64(len(msStats))))
}

// testMaxThroughput generates as many IDs as possible in a time window
func testMaxThroughput(duration time.Duration) {
	seen := make(map[uint64]bool)
	collisions := 0
	totalGenerated := 0

	// Track per-millisecond statistics
	msStats := make(map[int64]int)
	maxPerMs := 0
	collisionsPerMs := make(map[int64]int)

	start := time.Now()
	deadline := start.Add(duration)

	for time.Now().Before(deadline) {
		id, err := nano64.GenerateDefault()
		if err != nil {
			continue
		}

		value := id.Uint64Value()
		timestamp := id.GetTimestamp()

		if seen[value] {
			collisions++
			collisionsPerMs[timestamp]++
		}
		seen[value] = true
		totalGenerated++

		// Track IDs per millisecond
		msStats[timestamp]++
		if msStats[timestamp] > maxPerMs {
			maxPerMs = msStats[timestamp]
		}
	}

	elapsed := time.Since(start)
	rate := float64(totalGenerated) / elapsed.Seconds()

	// Find milliseconds with highest collision rates
	var maxCollisionMs int64
	maxCollisions := 0
	for ms, count := range collisionsPerMs {
		if count > maxCollisions {
			maxCollisions = count
			maxCollisionMs = ms
		}
	}

	fmt.Printf("  Duration: %v\n", duration)
	fmt.Printf("  Generated: %s IDs\n", formatNumberWithCommas(int64(totalGenerated)))
	fmt.Printf("  Rate: %s IDs/second\n", formatNumberWithCommas(int64(rate)))
	fmt.Printf("  Collisions: %s (%.6f%%)\n", formatNumberWithCommas(int64(collisions)), float64(collisions)/float64(totalGenerated)*100)
	fmt.Printf("  Unique IDs: %s\n", formatNumberWithCommas(int64(len(seen))))
	fmt.Printf("  Max IDs in single millisecond: %s\n", formatNumberWithCommas(int64(maxPerMs)))
	fmt.Printf("  Milliseconds with IDs: %s\n", formatNumberWithCommas(int64(len(msStats))))
	if maxCollisions > 0 {
		fmt.Printf("  Max collisions in single ms: %s (at timestamp %s, had %s IDs)\n",
			formatNumberWithCommas(int64(maxCollisions)), formatNumberWithCommas(maxCollisionMs), formatNumberWithCommas(int64(msStats[maxCollisionMs])))
	}

	// Calculate theoretical collision statistics for max throughput ms
	if maxPerMs > 0 {
		fmt.Println("\n  === Analysis of Peak Millisecond ===")
		fmt.Printf("  At peak rate of %s IDs/millisecond:\n", formatNumberWithCommas(int64(maxPerMs)))

		// With 20-bit random field (1,048,576 possible values)
		R := 1048576.0
		n := float64(maxPerMs)

		// Expected number of collisions using birthday paradox
		// E[collisions] ≈ n²/(2*R)
		expectedCollisions := (n * n) / (2 * R)

		// Probability that at least one collision occurs
		// P(at least 1) = 1 - e^(-n²/(2*R))
		// But we'll use a simpler approximation for small probabilities
		probAtLeastOne := 1.0 - (1.0 - 1.0/R)
		for i := 1; i < int(n); i++ {
			probAtLeastOne *= 1.0 - float64(i)/R
		}
		probAtLeastOne = 1.0 - probAtLeastOne

		// What rate would give us ~1% collision probability?
		// Solving: 1 - e^(-n²/(2*R)) = 0.01
		// n ≈ sqrt(2*R*0.01) ≈ sqrt(2 * 1048576 * 0.01) ≈ 145
		safeRate := 145.0

		fmt.Printf("    • Expected collisions: %.2f\n", expectedCollisions)
		fmt.Printf("    • Actual collisions observed: %s\n", formatNumberWithCommas(int64(maxCollisions)))
		if maxCollisions > 0 {
			accuracy := (float64(maxCollisions) / expectedCollisions) * 100
			fmt.Printf("    • Prediction accuracy: %.1f%%\n", accuracy)
		}
		fmt.Printf("    • This is %.1fx the safe rate (~%s IDs/ms for 1%% risk)\n",
			n/safeRate, formatNumberWithCommas(int64(safeRate)))

		if probAtLeastOne < 0.99 {
			fmt.Printf("    • Probability of at least one collision: %.2f%%\n", probAtLeastOne*100)
		} else {
			fmt.Printf("    • Probability of at least one collision: >99%% (virtually certain)\n")
		}
	}
}
