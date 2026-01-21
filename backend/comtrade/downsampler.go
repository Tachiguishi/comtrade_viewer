package comtrade

import "math"

const DefaultSampleRate = 50.0

// ComputeTimeAxisFromMeta builds a time axis for the given signal metadata.
// and nrates (SampleRates with SampRate and LastSampleNum). Returns microseconds.
// sampleLen must be the exact number of samples to compute, and in the timestamps-based
// branch len(timestamps) must be >= sampleLen; result is preallocated to this length.
// The returned slice has length sampleLen and contains time values in microseconds.
//
// Parameters:
//   - meta:        COMTRADE metadata. If meta.RatesNum > 0 and meta.SampleRates is
//     non-empty, the time axis is derived from the SampleRates (SampRate
//     and LastSampleNum) and timestamps is ignored.
//   - timestamps:  Optional raw timestamp values. These are only used when
//     meta.RatesNum == 0 or meta.SampleRates is empty. In that case
//     timestamps must be non-nil and have at least sampleLen entries,
//     otherwise the function will panic due to out-of-bounds access.
//     When timestamps are ignored (meta.RatesNum > 0 and SampleRates
//     non-empty), callers may pass nil or an empty slice.
//   - sampleLen:   Number of samples to produce in the time axis. This is the length
//     of the returned slice. When timestamps are used, sampleLen should
//     not exceed len(timestamps); typically it will equal len(timestamps).
func ComputeTimeAxisFromMeta(meta Metadata, timestamps []int32, sampleLen int) []float32 {
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
				rate = DefaultSampleRate
			}
			for i := prevLast; i < end; i++ {
				result[i] = elapsed + float32(i-prevLast)/rate*secondsToMicrosecondsMultiplier
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
				rate = DefaultSampleRate
			}
			for i := prevLast; i < sampleLen; i++ {
				result[i] = elapsed + float32(i-prevLast)/rate*secondsToMicrosecondsMultiplier
			}
		}
	} else {
		// Fallback: use raw timestamps with TimeMultiplier(microseconds)
		mul := float32(meta.TimeMultiplier)
		if mul == 0 {
			mul = 1.0
		}
		for i := range result {
			result[i] = float32(timestamps[i]) * mul
		}
	}

	return result
}


// downsampleLTTB applies Largest-Triangle-Three-Buckets downsampling algorithm
// Returns downsampled time and y arrays
func DownsampleLTTB(timestamps []float32, y []float64, targetPoints int) ([]float32, []float64) {
	n := len(y)
	if n <= targetPoints || targetPoints < 3 {
		return timestamps, y
	}

	downsampledT := make([]float32, 0, targetPoints)
	downsampledY := make([]float64, 0, targetPoints)

	// Always keep first point
	downsampledT = append(downsampledT, timestamps[0])
	downsampledY = append(downsampledY, y[0])

	bucketSize := float64(n-2) / float64(targetPoints-2)

	for i := 0; i < targetPoints-2; i++ {
		avgRangeStart := int(float64(i+1)*bucketSize) + 1
		avgRangeEnd := min(int(float64(i+2)*bucketSize) + 1, n)

		// Calculate average point in next bucket
		avgX := float64(0)
		avgY := float64(0)
		count := avgRangeEnd - avgRangeStart
		if count > 0 {
			for j := avgRangeStart; j < avgRangeEnd; j++ {
				avgX += float64(timestamps[j])
				avgY += y[j]
			}
			avgX /= float64(count)
			avgY /= float64(count)
		}

		// Find point in current bucket with largest triangle area
		rangeStart := int(float64(i)*bucketSize) + 1
		rangeEnd := min(avgRangeStart, n)

		maxArea := -1.0
		maxIdx := rangeStart

		lastX := float64(downsampledT[len(downsampledT)-1])
		lastY := downsampledY[len(downsampledY)-1]

		for j := rangeStart; j < rangeEnd; j++ {
			area := math.Abs((lastX-avgX)*(y[j]-lastY)-(lastX-float64(timestamps[j]))*(avgY-lastY)) * 0.5
			if area > maxArea {
				maxArea = area
				maxIdx = j
			}
		}

		downsampledT = append(downsampledT, timestamps[maxIdx])
		downsampledY = append(downsampledY, y[maxIdx])
	}

	// Always keep last point
	downsampledT = append(downsampledT, timestamps[n-1])
	downsampledY = append(downsampledY, y[n-1])

	return downsampledT, downsampledY
}

// downsampleDigital downsamples digital signals by keeping state changes
func DownsampleDigital(timestamps []float32, y []int8) ([]float32, []int8) {
	n := len(y)

	// Keep all state changes and uniformly sample the rest
	downsampledT := make([]float32, 0)
	downsampledY := make([]int8, 0)

	// Always keep first point
	downsampledT = append(downsampledT, timestamps[0])
	downsampledY = append(downsampledY, y[0])

	// Find all state changes
	for i := 1; i < n - 1; i++ {
		// If current point differs from either neighbor, it's a state change
		if y[i] != y[i+1] || y[i] != y[i-1] {
			downsampledT = append(downsampledT, timestamps[i])
			downsampledY = append(downsampledY, y[i])
		}
	}

	// Always keep last point
	downsampledT = append(downsampledT, timestamps[n-1])
	downsampledY = append(downsampledY, y[n-1])

	return downsampledT, downsampledY
}
