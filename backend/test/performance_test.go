package test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"comtradeviewer/comtrade"
)

func BenchmarkDATFileParsing(b *testing.B) {
	// Test data file path
	datPath := filepath.Join(".", "data", "test", "dat")
	cfgPath := filepath.Join(".", "data", "test", "cfg")

	// Check if test files exist, skip if not
	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		b.Skip("Test files not found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfgData, err := os.ReadFile(cfgPath)
		if err != nil {
			b.Fatalf("Failed to read CFG file: %v", err)
		}
		datData, err := os.ReadFile(datPath)
		if err != nil {
			b.Fatalf("Failed to read DAT file: %v", err)
		}
		_, _, err = comtrade.ParseComtradeFromBytes(cfgData, datData)
		if err != nil {
			b.Fatalf("Failed to parse: %v", err)
		}
	}
}

// TestCachingVsReparsing compares the performance of:
// 1. Parsing from disk each time
// 2. Parsing and caching in memory
func TestCachingPerformance(t *testing.T) {
	datPath := filepath.Join(".", "data", "test", "dat")
	cfgPath := filepath.Join(".", "data", "test", "cfg")

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Skip("Test files not found")
	}

	numRequests := 10

	// Test 1: Direct parsing each time
	t.Run("DirectParsing", func(t *testing.T) {
		start := time.Now()
		for range numRequests {
			cfgData, err := os.ReadFile(cfgPath)
			if err != nil {
				t.Fatalf("Failed to read CFG file: %v", err)
			}
			datData, err := os.ReadFile(datPath)
			if err != nil {
				t.Fatalf("Failed to read DAT file: %v", err)
			}
			_, _, err = comtrade.ParseComtradeFromBytes(cfgData, datData)
			if err != nil {
				t.Fatalf("Failed to parse: %v", err)
			}
		}
		duration := time.Since(start)
		t.Logf("Direct parsing %d times: %v (avg: %v per request)", numRequests, duration, duration/time.Duration(numRequests))
	})

	// Test 2: Using memory cache
	t.Run("MemoryCache", func(t *testing.T) {
		cache := comtrade.NewDatasetCache(10)

		// First request: parse and cache
		start := time.Now()
		cfgData, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("Failed to read CFG file: %v", err)
		}
		datData, err := os.ReadFile(datPath)
		if err != nil {
			t.Fatalf("Failed to read DAT file: %v", err)
		}
		meta, dat, err := comtrade.ParseComtradeFromBytes(cfgData, datData)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}
		cache.Set("test", meta, dat)
		firstTime := time.Since(start)
		t.Logf("First request (parse+cache): %v", firstTime)

		// Subsequent requests: from cache
		start = time.Now()
		for i := 1; i < numRequests; i++ {
			_, _, ok := cache.Get("test")
			if !ok {
				t.Fatal("Expected cache hit")
			}
		}
		subsequentTime := time.Since(start)
		t.Logf("Subsequent %d requests (from cache): %v (avg: %v per request)",
			numRequests-1, subsequentTime, subsequentTime/time.Duration(numRequests-1))

		totalTime := firstTime + subsequentTime
		t.Logf("Total for %d requests: %v (avg: %v per request)", numRequests, totalTime, totalTime/time.Duration(numRequests))
	})

	// Test 3: JSON serialization overhead
	t.Run("JSONSerialization", func(t *testing.T) {
		cfgData, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("Failed to read CFG file: %v", err)
		}
		datData, err := os.ReadFile(datPath)
		if err != nil {
			t.Fatalf("Failed to read DAT file: %v", err)
		}
		meta, dat, err := comtrade.ParseComtradeFromBytes(cfgData, datData)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		// Measure serialization time
		start := time.Now()
		data := map[string]any{"meta": meta, "dat": dat}
		b, _ := json.MarshalIndent(data, "", "  ")
		serializeTime := time.Since(start)

		t.Logf("JSON serialization: %v, size: %d bytes (compression: %.2f%%)",
			serializeTime, len(b), float64(len(b))/float64(1024*1024)*100)

		// Measure deserialization time
		start = time.Now()
		var decoded map[string]any
		_ = json.Unmarshal(b, &decoded)
		deserializeTime := time.Since(start)

		t.Logf("JSON deserialization: %v", deserializeTime)
		t.Logf("Total JSON roundtrip: %v", serializeTime+deserializeTime)
	})

	// Test 4: Downsampling performance
	t.Run("Downsampling", func(t *testing.T) {
		cfgData, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("Failed to read CFG file: %v", err)
		}
		datData, err := os.ReadFile(datPath)
		if err != nil {
			t.Fatalf("Failed to read DAT file: %v", err)
		}
		meta, dat, err := comtrade.ParseComtradeFromBytes(cfgData, datData)
		if err != nil {
			t.Fatalf("Failed to parse: %v", err)
		}

		if len(dat.AnalogChannels) == 0 {
			t.Skip("No analog channels in test data")
		}

		chData := dat.AnalogChannels[0]
		timestamps := comtrade.ComputeTimeAxisFromMeta(*meta, dat.Timestamps, len(dat.Timestamps))

		// Create test y data
		y := make([]float64, len(chData.RawData))
		for i, d := range chData.RawData {
			y[i] = float64(d)
		}

		originalPoints := len(y)
		targetPoints := 5000

		// Benchmark LTTB downsampling
		start := time.Now()
		_, newY := comtrade.DownsampleLTTB(timestamps, y, targetPoints)
		downsampleTime := time.Since(start)

		t.Logf("LTTB downsampling: %v, %d -> %d points (%.1f%% reduction)",
			downsampleTime, originalPoints, len(newY), float64(originalPoints-len(newY))/float64(originalPoints)*100)

		// Estimate network transfer time (assuming 100Mbps = 12.5MB/s)
		originalSize := originalPoints * 16 // 8 bytes per float64 + ~8 for timestamp
		downsampledSize := len(newY) * 16
		networkSpeed := 12.5 * 1024 * 1024 // bytes per second

		t.Logf("Estimated network transfer (100Mbps):")
		t.Logf("  Original: %d bytes = %.3fs", originalSize, float64(originalSize)/networkSpeed)
		t.Logf("  Downsampled: %d bytes = %.3fs", downsampledSize, float64(downsampledSize)/networkSpeed)
	})
}

// TestDownsamplingCorrectness verifies LTTB produces correct output
func TestLTTBDownsampling(t *testing.T) {
	datPath := filepath.Join(".", "data", "test", "dat")
	cfgPath := filepath.Join(".", "data", "test", "cfg")

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		t.Skip("Test files not found")
	}

	cfgData, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("Failed to read CFG file: %v", err)
	}
	datData, err := os.ReadFile(datPath)
	if err != nil {
		t.Fatalf("Failed to read DAT file: %v", err)
	}
	meta, dat, err := comtrade.ParseComtradeFromBytes(cfgData, datData)
	if err != nil {
		t.Fatalf("Failed to parse: %v", err)
	}

	if len(dat.AnalogChannels) == 0 {
		t.Skip("No analog channels")
	}

	chData := dat.AnalogChannels[0]
	timestamps := comtrade.ComputeTimeAxisFromMeta(*meta, dat.Timestamps, len(dat.Timestamps))

	y := make([]float64, len(chData.RawData))
	for i, d := range chData.RawData {
		y[i] = float64(d)
	}

	// Test various target point counts
	targetCounts := []int{100, 500, 1000, 5000}
	for _, target := range targetCounts {
		newT, newY := comtrade.DownsampleLTTB(timestamps, y, target)

		if len(newT) != len(newY) {
			t.Errorf("Times and Y mismatch: %d vs %d", len(newT), len(newY))
		}

		if len(newY) > target {
			t.Errorf("Downsampling failed: wanted <= %d points, got %d", target, len(newY))
		}

		// Verify first and last points are preserved
		if newT[0] != timestamps[0] || newY[0] != y[0] {
			t.Errorf("First point not preserved")
		}
		if newT[len(newT)-1] != timestamps[len(timestamps)-1] || newY[len(newY)-1] != y[len(y)-1] {
			t.Errorf("Last point not preserved")
		}

		t.Logf("LTTB(%d): %d points -> %d points (%.1f%%)",
			target, len(y), len(newY), float64(len(y)-len(newY))/float64(len(y))*100)
	}
}

// TestMemoryCacheEviction verifies LRU eviction works correctly
func TestMemoryCacheEviction(t *testing.T) {
	cache := comtrade.NewDatasetCache(3)

	// Add 5 entries to a cache with size 3
	for i := 1; i <= 5; i++ {
		meta := &comtrade.Metadata{Station: fmt.Sprintf("Station%d", i)}
		dat := &comtrade.ChannelData{}
		cache.Set(fmt.Sprintf("id%d", i), meta, dat)
	}

	// Verify only the last 3 are in cache
	if cache.Size() != 3 {
		t.Errorf("Expected cache size 3, got %d", cache.Size())
	}

	// Verify oldest entries were evicted
	if _, _, ok := cache.Get("id1"); ok {
		t.Error("id1 should have been evicted")
	}
	if _, _, ok := cache.Get("id2"); ok {
		t.Error("id2 should have been evicted")
	}

	// Verify newest entries are present
	if _, _, ok := cache.Get("id3"); !ok {
		t.Error("id3 should be in cache")
	}
	if _, _, ok := cache.Get("id4"); !ok {
		t.Error("id4 should be in cache")
	}
	if _, _, ok := cache.Get("id5"); !ok {
		t.Error("id5 should be in cache")
	}
}
