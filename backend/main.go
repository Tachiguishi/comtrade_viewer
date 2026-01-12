package main

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"comtradeviewer/comtrade"

	"github.com/gin-gonic/gin"
)

type DatasetInfo struct {
	DatasetID string `json:"datasetId"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	SizeBytes int64  `json:"sizeBytes"`
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func saveUploadedFile(fh *multipart.Form, field string, dest string) error {
	files := fh.File[field]
	if len(files) == 0 {
		return os.ErrNotExist
	}
	src, err := files[0].Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	return err
}

func listDatasets(root string) ([]DatasetInfo, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []DatasetInfo{}, nil
		}
		return nil, err
	}
	out := []DatasetInfo{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		id := e.Name()
		dp := filepath.Join(root, id)
		var size int64
		for _, fn := range []string{"cfg", "dat"} {
			p := filepath.Join(dp, fn)
			if st, err := os.Stat(p); err == nil {
				size += st.Size()
			}
		}
		out = append(out, DatasetInfo{DatasetID: id, Name: id, CreatedAt: 0, SizeBytes: size})
	}
	return out, nil
}

func main() {
	r := gin.Default()
	r.MaxMultipartMemory = 128 << 20 // 128MB

	dataRoot := filepath.Join(".", "data")
	_ = ensureDir(dataRoot)

	// Upload
	r.POST("/api/datasets/import", func(c *gin.Context) {
		if err := c.Request.ParseMultipartForm(256 << 20); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid form"})
			return
		}
		fh := c.Request.MultipartForm
		datasetID := strconv.FormatInt(time.Now().UnixNano(), 10)
		dp := filepath.Join(dataRoot, datasetID)
		if err := ensureDir(dp); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "storage error"})
			return
		}
		if err := saveUploadedFile(fh, "cfg", filepath.Join(dp, "cfg")); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cfg file required"})
			return
		}
		if err := saveUploadedFile(fh, "dat", filepath.Join(dp, "dat")); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "dat file required"})
			return
		}
		meta, dat, err := comtrade.ParseComtrade(filepath.Join(dp, "cfg"), filepath.Join(dp, "dat"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "parse error: " + err.Error()})
			return
		}
		_ = writeJSON(filepath.Join(dp, "meta.json"), meta)
		_ = writeJSON(filepath.Join(dp, "data.json"), dat)
		c.JSON(http.StatusOK, gin.H{"datasetId": datasetID, "name": datasetID})
	})

	// List datasets
	r.GET("/api/datasets", func(c *gin.Context) {
		lst, err := listDatasets(dataRoot)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "list error"})
			return
		}
		c.JSON(http.StatusOK, lst)
	})

	// Metadata
	r.GET("/api/datasets/:id/metadata", func(c *gin.Context) {
		id := c.Param("id")
		dp := filepath.Join(dataRoot, id)
		mp := filepath.Join(dp, "meta.json")
		var meta comtrade.Metadata
		if b, err := os.ReadFile(mp); err == nil {
			if err := json.Unmarshal(b, &meta); err == nil {
				c.JSON(http.StatusOK, meta)
				return
			}
		}
		if m, _, err := comtrade.ParseComtrade(filepath.Join(dp, "cfg"), filepath.Join(dp, "dat")); err == nil {
			c.JSON(http.StatusOK, m)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
	})

	// Waveforms (real data from parsed ComTrade)
	r.GET("/api/datasets/:id/waveforms", func(c *gin.Context) {
		id := c.Param("id")
		dp := filepath.Join(dataRoot, id)
		
		// Load parsed data
		var meta comtrade.Metadata
		var dat comtrade.ChannelData
		
		metaPath := filepath.Join(dp, "meta.json")
		dataPath := filepath.Join(dp, "data.json")
		
		// Try to load from cache first
		if b, err := os.ReadFile(metaPath); err == nil {
			json.Unmarshal(b, &meta)
		}
		if b, err := os.ReadFile(dataPath); err == nil {
			json.Unmarshal(b, &dat)
		}
		
		// If cache doesn't exist, parse now
		if len(dat.Timestamps) == 0 {
			m, d, err := comtrade.ParseComtrade(filepath.Join(dp, "cfg"), filepath.Join(dp, "dat"))
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse ComTrade: " + err.Error()})
				return
			}
			meta = *m
			dat = *d
		}

		if len(dat.Timestamps) == 0 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "no channel data"})
			return
		}
		
		// Parse requested channels
		chsStr := c.Query("channels")
		if chsStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "channels parameter required"})
			return
		}
		chs := strings.Split(chsStr, ",")
		
		// Build series response
		series := make([]map[string]any, 0, len(chs))
		timestamps := comtrade.ComputeTimeAxisFromMeta(meta, dat.Timestamps, len(dat.Timestamps))

		for _, chID := range chs {
			chID = strings.TrimSpace(chID)
			if chID == "" {
				continue
			}
			
			// Check if it's analog (A1, A2, etc) or digital (D1, D2, etc)
			if after, ok :=strings.CutPrefix(chID, "A"); ok  {
				// Analog channel
				chNum, err := strconv.Atoi(after)
				if err != nil {
					continue
				}
				
				// Find the channel data
				for _, chData := range dat.AnalogChannels {
					if chData.ChannelNumber == chNum {
						sampleLen := max(len(chData.RawData), len(chData.RawDataFloat))
						y := make([]float64, sampleLen)
						
						// Get scaling factors from metadata
						var multiplier, offset float64 = 1.0, 0.0
						if chNum-1 < len(meta.AnalogChannels) {
							ch := meta.AnalogChannels[chNum-1]
							multiplier = ch.Multiplier
							offset = ch.Offset
						}
						
						if len(chData.RawDataFloat) == sampleLen {
							// Use float data if available
							for i, d := range chData.RawDataFloat {
								// Apply scaling: physical_value = raw * multiplier + offset
								y[i] = float64(d)*multiplier + offset
							}
						} else  {
							// Fallback to int data
							for i, d := range chData.RawData {
								// Apply scaling: physical_value = raw * multiplier + offset
								y[i] = float64(d)*multiplier + offset
							}
						}

						series = append(series, map[string]any{
							"channel": chID,
							"name":    meta.AnalogChannels[chNum-1].ChannelName,
							"unit":    meta.AnalogChannels[chNum-1].Unit,
							"y":       y,
						})
						break
					}
				}
			} else if after0, ok0 :=strings.CutPrefix(chID, "D"); ok0  {
				// Digital channel
				chNum, err := strconv.Atoi(after0)
				if err != nil {
					continue
				}
				
				for _, chData := range dat.DigitalChannels {
					if chData.ChannelNumber == chNum {
						sampleLen := len(chData.RawData)
						y := make([]int8, sampleLen)

						copy(y, chData.RawData)
						
						series = append(series, map[string]any{
							"channel": chID,
							"name":    meta.DigitalChannels[chNum-1].ChannelName,
							"y":       y,
						})
						break
					}
				}
			}
		}
		
		c.JSON(http.StatusOK, gin.H{
			"series": series,
			"times":   timestamps,
			"window": map[string]float32{"start": timestamps[0], "end": timestamps[len(timestamps)-1]},
		})
	})

	// Annotations (file-backed JSON)
	r.GET("/api/datasets/:id/annotations", func(c *gin.Context) {
		id := c.Param("id")
		p := filepath.Join(dataRoot, id, "annotations.json")
		f, err := os.Open(p)
		if err != nil {
			c.JSON(http.StatusOK, []any{})
			return
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		var out []map[string]any
		_ = json.Unmarshal(b, &out)
		c.JSON(http.StatusOK, out)
	})

	r.POST("/api/datasets/:id/annotations", func(c *gin.Context) {
		id := c.Param("id")
		p := filepath.Join(dataRoot, id, "annotations.json")
		var ann map[string]any
		if err := c.BindJSON(&ann); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "bad json"})
			return
		}
		var out []map[string]any
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &out)
		}
		ann["id"] = strconv.FormatInt(time.Now().UnixNano(), 10)
		out = append(out, ann)
		_ = writeJSON(p, out)
		c.JSON(http.StatusOK, gin.H{"id": ann["id"]})
	})

	r.DELETE("/api/datasets/:id/annotations/:annId", func(c *gin.Context) {
		id := c.Param("id")
		annID := c.Param("annId")
		p := filepath.Join(dataRoot, id, "annotations.json")
		var out []map[string]any
		if b, err := os.ReadFile(p); err == nil {
			_ = json.Unmarshal(b, &out)
		}
		kept := make([]map[string]any, 0, len(out))
		for _, a := range out {
			if v, ok := a["id"].(string); ok && v == annID {
				continue
			}
			kept = append(kept, a)
		}
		_ = writeJSON(p, kept)
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.Run(":8080")
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func mathSin(x float64) float64 { // small inline to avoid extra imports
	// Taylor approximation for simplicity of dependencies
	// Good enough for demo rendering
	x3 := x * x * x
	x5 := x3 * x * x
	x7 := x5 * x * x
	return x - (x3 / 6.0) + (x5 / 120.0) - (x7 / 5040.0)
}
