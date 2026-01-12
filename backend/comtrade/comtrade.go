package comtrade

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func findFileCaseInsensitive(directory, base, ext string) (string, error) {
	ext = strings.TrimPrefix(ext, ".")
	candidate := filepath.Join(directory, base+"."+ext)
	if _, err := os.Stat(candidate); err == nil {
		return candidate, nil
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("read dir %s: %w", directory, err)
	}
	target := strings.ToLower(base + "." + ext)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.ToLower(e.Name()) == target {
			return filepath.Join(directory, e.Name()), nil
		}
	}
	return "", fmt.Errorf("file not found (case-insensitive): %s", candidate)
}

func ParseComtrade(cfgPath string, datPath string) (*Metadata, *ChannelData, error) {
	cfgFile, err := os.Open(cfgPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open CFG file: %w", err)
	}
	defer cfgFile.Close()

	cfg, err := ParseCFGFile(cfgFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse CFG file: %w", err)
	}

	datFile, err := os.Open(datPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open DAT file: %w", err)
	}
	defer datFile.Close()

	dat, err := ParseDATFile(datFile, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse DAT file: %w", err)
	}

	return cfg, dat, nil
}

// ComputeTimeAxisFromMeta builds a time axis for the given signal metadata.
// and nrates (SampleRates with SampRate and LastSampleNum). Returns microseconds.
// sampleLen must be the exact number of samples to compute, and in the timestamps-based
// branch len(timestamps) must be >= sampleLen; result is preallocated to this length.
// The returned slice has length sampleLen and contains time values in microseconds.
//
// Parameters:
//   - meta:        COMTRADE metadata. If meta.RatesNum > 0 and meta.SampleRates is
//                 non-empty, the time axis is derived from the SampleRates (SampRate
//                 and LastSampleNum) and timestamps is ignored.
//   - timestamps:  Optional raw timestamp values. These are only used when
//                 meta.RatesNum == 0 or meta.SampleRates is empty. In that case
//                 timestamps must be non-nil and have at least sampleLen entries,
//                 otherwise the function will panic due to out-of-bounds access.
//                 When timestamps are ignored (meta.RatesNum > 0 and SampleRates
//                 non-empty), callers may pass nil or an empty slice.
//   - sampleLen:   Number of samples to produce in the time axis. This is the length
//                 of the returned slice. When timestamps are used, sampleLen should
//                 not exceed len(timestamps); typically it will equal len(timestamps).
func ComputeTimeAxisFromMeta(meta Metadata, timestamps []uint32, sampleLen int) []float32 {
	result := make([]float32, sampleLen)

	secondsToMicrosecondsMultiplier := float32(1e6)
	// Prefer Δt(N) = Σ(samples_in_segment / sample_rate) when nrates available
	if meta.RatesNum > 0 && len(meta.SampleRates) > 0 {
		elapsed := float32(0)
		prevLast := 0
		for _, sr := range meta.SampleRates {
			end := min(sr.LastSampleNum, sampleLen)
			rate := float32(sr.SampRate)
			if rate <= 0 {
				rate = 1000.0
			}
			for i := prevLast; i < end; i++ {
				result[i] = elapsed + float32(i-prevLast)/rate * secondsToMicrosecondsMultiplier
			}
			elapsed += float32(end-prevLast) / rate * secondsToMicrosecondsMultiplier
			prevLast = end
			if prevLast >= sampleLen {
				break
			}
		}
		if prevLast < sampleLen {
			rate := float32(meta.SampleRates[len(meta.SampleRates)-1].SampRate)
			if rate <= 0 {
				rate = 1000.0
			}
			for i := prevLast; i < sampleLen; i++ {
				result[i] = elapsed + float32(i-prevLast)/rate * secondsToMicrosecondsMultiplier
			}
		}
	} else {
		mul := float32(meta.TimeMultiplier)
		if mul == 0 {
			// Default to microseconds→seconds if not set
			mul = 1e-6
		}
		for i := range result {
			result[i] = float32(timestamps[i]) * mul * secondsToMicrosecondsMultiplier
		}
	}

	return result
}
