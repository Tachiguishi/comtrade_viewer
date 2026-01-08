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

	"github.com/gin-gonic/gin"
)

type ChannelMeta struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Type  string  `json:"type"` // analog | digital
	Unit  *string `json:"unit"`
	Scale *struct {
		K float64 `json:"k"`
		B float64 `json:"b"`
	} `json:"scale"`
}

type Metadata struct {
	Station   string         `json:"station"`
	Recording map[string]any `json:"recording"`
	Sampling  struct {
		Rate float64 `json:"rate"`
	} `json:"sampling"`
	Channels  []ChannelMeta `json:"channels"`
	Timebase  float64       `json:"timebase"`
	StartTime int64         `json:"startTime"`
	EndTime   int64         `json:"endTime"`
}

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

func parseCfgMinimal(cfgPath string) (Metadata, error) {
	// Minimal, robust-ish parser for common ComTrade .cfg layout.
	// Fallbacks are used when fields are missing.
	var meta Metadata
	meta.Recording = map[string]any{}
	meta.Timebase = 1e-6
	meta.StartTime = time.Now().UnixMilli()
	meta.EndTime = meta.StartTime

	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return meta, err
	}
	lines := strings.Split(string(b), "\n")
	// Typical: first line station,device,id
	if len(lines) > 0 {
		parts := strings.Split(lines[0], ",")
		if len(parts) > 0 {
			meta.Station = strings.TrimSpace(parts[0])
		}
		if len(parts) > 1 {
			meta.Recording["device"] = strings.TrimSpace(parts[1])
		}
	}
	// Channel counts line e.g. "nA,nD"
	var nA, nD int
	if len(lines) > 1 {
		parts := strings.Split(lines[1], ",")
		if len(parts) >= 2 {
			nA, _ = strconv.Atoi(strings.TrimSpace(parts[0]))
			nD, _ = strconv.Atoi(strings.TrimSpace(parts[1]))
		}
	}
	// Read analog channel names (heuristic)
	idx := 2
	for i := 0; i < nA && idx < len(lines); i++ {
		line := strings.TrimSpace(lines[idx])
		idx++
		if line == "" {
			continue
		}
		ps := strings.Split(line, ",")
		name := ps[0]
		ch := ChannelMeta{ID: "A" + strconv.Itoa(i+1), Name: strings.TrimSpace(name), Type: "analog"}
		meta.Channels = append(meta.Channels, ch)
	}
	for i := 0; i < nD && idx < len(lines); i++ {
		line := strings.TrimSpace(lines[idx])
		idx++
		if line == "" {
			continue
		}
		ps := strings.Split(line, ",")
		name := ps[0]
		ch := ChannelMeta{ID: "D" + strconv.Itoa(i+1), Name: strings.TrimSpace(name), Type: "digital"}
		meta.Channels = append(meta.Channels, ch)
	}
	// Sampling rate line (heuristic)
	for _, l := range lines[idx:] {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if strings.Contains(l, ",") {
			ps := strings.Split(l, ",")
			if len(ps) >= 2 {
				if v, err := strconv.ParseFloat(strings.TrimSpace(ps[0]), 64); err == nil && v > 0 {
					meta.Sampling.Rate = v
					break
				}
			}
		}
	}
	if meta.Sampling.Rate == 0 {
		meta.Sampling.Rate = 1000
	}
	return meta, nil
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
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
		meta, err := parseCfgMinimal(filepath.Join(dp, "cfg"))
		if err == nil {
			_ = writeJSON(filepath.Join(dp, "meta.json"), meta)
		}
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
		var meta Metadata
		if b, err := os.ReadFile(mp); err == nil {
			if err := json.Unmarshal(b, &meta); err == nil {
				c.JSON(http.StatusOK, meta)
				return
			}
		}
		if m, err := parseCfgMinimal(filepath.Join(dp, "cfg")); err == nil {
			c.JSON(http.StatusOK, m)
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"error": "metadata not found"})
	})

	// Waveforms (placeholder synthetic)
	r.GET("/api/datasets/:id/waveforms", func(c *gin.Context) {
		chs := strings.Split(c.Query("channels"), ",")
		startMs, _ := strconv.ParseFloat(c.Query("start"), 64)
		endMs, _ := strconv.ParseFloat(c.Query("end"), 64)
		if endMs <= startMs {
			endMs = startMs + 500
		}
		points := 2000
		series := make([]map[string]any, 0, len(chs))
		// Synthetic sine for demo
		dur := endMs - startMs
		for i, ch := range chs {
			if ch == "" {
				continue
			}
			t := make([]float64, points)
			y := make([]float64, points)
			for k := 0; k < points; k++ {
				tt := startMs + (dur*float64(k))/float64(points-1)
				t[k] = tt / 1000.0
				y[k] = 0.5 * float64(i+1) *
					(0.8*mathSin(2*3.14159*(0.01*tt)) + 0.2*mathSin(2*3.14159*(0.002*tt)))
			}
			series = append(series, map[string]any{"channelId": ch, "t": t, "y": y})
		}
		c.JSON(http.StatusOK, gin.H{
			"series": series,
			"window": map[string]float64{"start": startMs / 1000.0, "end": endMs / 1000.0},
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

func mathSin(x float64) float64 { // small inline to avoid extra imports
	// Taylor approximation for simplicity of dependencies
	// Good enough for demo rendering
	x3 := x * x * x
	x5 := x3 * x * x
	x7 := x5 * x * x
	return x - (x3 / 6.0) + (x5 / 120.0) - (x7 / 5040.0)
}
