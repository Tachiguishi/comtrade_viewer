package main

import (
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
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
			writeError(c, http.StatusBadRequest, "INVALID_FORM", "无效的表单数据", gin.H{"hint": "请通过multipart/form-data提交.cfg与.dat文件"})
			return
		}
		fh := c.Request.MultipartForm
		datasetID := strconv.FormatInt(time.Now().UnixNano(), 10)
		dp := filepath.Join(dataRoot, datasetID)
		if err := ensureDir(dp); err != nil {
			writeError(c, http.StatusInternalServerError, "STORAGE_ERROR", "服务器存储异常", gin.H{"detail": err.Error()})
			return
		}
		// Validate presence and extensions before saving
		if !hasFileField(fh, "cfg") {
			writeError(c, http.StatusBadRequest, "CFG_MISSING", ".cfg文件缺失", gin.H{"hint": "请选择配置文件(.cfg)"})
			return
		}
		if !hasFileField(fh, "dat") {
			writeError(c, http.StatusBadRequest, "DAT_MISSING", ".dat文件缺失", gin.H{"hint": "请选择数据文件(.dat)"})
			return
		}
		// Extension checks (case-insensitive)
		if !hasFileExt(fh, "cfg", ".cfg") {
			writeError(c, http.StatusBadRequest, "CFG_EXT_INVALID", "配置文件扩展名无效", gin.H{"expected": ".cfg"})
			return
		}
		if !hasFileExt(fh, "dat", ".dat") {
			writeError(c, http.StatusBadRequest, "DAT_EXT_INVALID", "数据文件扩展名无效", gin.H{"expected": ".dat"})
			return
		}
		if err := saveUploadedFile(fh, "cfg", filepath.Join(dp, "cfg")); err != nil {
			writeError(c, http.StatusBadRequest, "CFG_SAVE_FAILED", "保存配置文件失败", gin.H{"detail": err.Error()})
			return
		}
		if err := saveUploadedFile(fh, "dat", filepath.Join(dp, "dat")); err != nil {
			writeError(c, http.StatusBadRequest, "DAT_SAVE_FAILED", "保存数据文件失败", gin.H{"detail": err.Error()})
			return
		}
		meta, dat, err := comtrade.ParseComtrade(filepath.Join(dp, "cfg"), filepath.Join(dp, "dat"))
		if err != nil {
			code, msg, details := toFriendlyParseError(err)
			writeError(c, http.StatusBadRequest, code, msg, details)
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
			writeError(c, http.StatusInternalServerError, "LIST_ERROR", "获取数据集列表失败", gin.H{"detail": err.Error()})
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
		writeError(c, http.StatusNotFound, "METADATA_NOT_FOUND", "未找到元数据", gin.H{"id": id})
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
				code, msg, details := toFriendlyParseError(err)
				writeError(c, http.StatusInternalServerError, code, msg, details)
				return
			}
			meta = *m
			dat = *d
		}

		if len(dat.Timestamps) == 0 {
			writeError(c, http.StatusInternalServerError, "NO_DATA", "未找到通道数据", gin.H{"id": id})
			return
		}
		
		// Parse requested channels
		chsStr := c.Query("channels")
		if chsStr == "" {
			writeError(c, http.StatusBadRequest, "CHANNELS_REQUIRED", "缺少channels参数", gin.H{"format": "A1,A2,D1,..."})
			return
		}
		chs := strings.Split(chsStr, ",")
		
		// Build series response
		series := make([]map[string]any, 0, len(chs))
		timestamps := comtrade.ComputeTimeAxisFromMeta(meta, dat.Timestamps, len(dat.Timestamps))

		analogChannels := make([]int, 0)
		digitalChannels := make([]int, 0)
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

				analogChannels = append(analogChannels, chNum)
			} else if after0, ok0 :=strings.CutPrefix(chID, "D"); ok0  {
				// Digital channel
				chNum, err := strconv.Atoi(after0)
				if err != nil {
					continue
				}

				digitalChannels = append(digitalChannels, chNum)
			}
		}

		sort.Ints(analogChannels)
		sort.Ints(digitalChannels)

		// Find the channel data
		for _, chData := range dat.AnalogChannels {
			if len(analogChannels) == 0 {
				break
			}

			found := false
			for _, chNum := range analogChannels {
				if chData.ChannelNumber == chNum {
					found = true
					// remove from list to avoid duplicate processing
					analogChannels = removeInt(analogChannels, chNum)
					break
				}
			}
			if !found {
				continue
			}

			chNum := chData.ChannelNumber

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
				"channel": chNum,
				"type":   "analog",
				"name":    meta.AnalogChannels[chNum-1].ChannelName,
				"unit":    meta.AnalogChannels[chNum-1].Unit,
				"y":       y,
			})
		}

		for _, chData := range dat.DigitalChannels {
			if len(digitalChannels) == 0 {
				break
			}

			found := false
			for _, chNum := range digitalChannels {
				if chData.ChannelNumber == chNum {
					found = true
					// remove from list to avoid duplicate processing
					digitalChannels = removeInt(digitalChannels, chNum)
					break
				}
			}
			if !found {
				continue
			}

			chNum := chData.ChannelNumber
			sampleLen := len(chData.RawData)
			y := make([]int8, sampleLen)

			copy(y, chData.RawData)
			
			series = append(series, map[string]any{
				"channel": chNum,
				"type":   "digital",
				"name":    meta.DigitalChannels[chNum-1].ChannelName,
				"y":       y,
			})
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
			// Return empty list on absent file, else error
			if os.IsNotExist(err) {
				c.JSON(http.StatusOK, []any{})
				return
			}
			writeError(c, http.StatusInternalServerError, "ANNOTATIONS_READ_ERROR", "读取标注失败", gin.H{"detail": err.Error()})
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
			writeError(c, http.StatusBadRequest, "BAD_JSON", "JSON格式错误", gin.H{"detail": err.Error()})
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
		if err := writeJSON(p, kept); err != nil {
			writeError(c, http.StatusInternalServerError, "ANNOTATIONS_WRITE_ERROR", "写入标注失败", gin.H{"detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	r.Run(":8080")
}

func removeInt(source []int, target int) []int {
	for i, v := range source {
		if v == target {
			return append(source[:i], source[i+1:]...)
		}
	}
	return source
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

// --- Error handling helpers ---

type apiError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func writeError(c *gin.Context, status int, code string, message string, details interface{}) {
	c.JSON(status, gin.H{"error": apiError{Code: code, Message: message, Details: details}})
}

func hasFileField(fh *multipart.Form, field string) bool {
	files := fh.File[field]
	return len(files) > 0
}

func hasFileExt(fh *multipart.Form, field string, want string) bool {
	files := fh.File[field]
	if len(files) == 0 {
		return false
	}
	name := files[0].Filename
	return strings.EqualFold(filepath.Ext(name), want)
}

// toFriendlyParseError maps internal parse errors to user-friendly messages
func toFriendlyParseError(err error) (string, string, gin.H) {
	s := err.Error()
	// Generic fallback
	code := "PARSE_ERROR"
	msg := "解析COMTRADE文件失败"
	details := gin.H{"error": s}

	// Specific mappings
	switch {
	case strings.Contains(s, "failed to open CFG"):
		code = "CFG_OPEN_FAILED"
		msg = "无法打开配置文件(.cfg)"
	case strings.Contains(s, "failed to parse CFG"):
		code = "CFG_PARSE_FAILED"
		msg = "配置文件(.cfg)解析失败，请检查格式"
	case strings.Contains(s, "failed to open DAT"):
		code = "DAT_OPEN_FAILED"
		msg = "无法打开数据文件(.dat)"
	case strings.Contains(s, "failed to parse DAT"):
		code = "DAT_PARSE_FAILED"
		msg = "数据文件(.dat)解析失败，请检查格式与版本"
	case strings.Contains(s, "unsupported COMTRADE version"):
		code = "VERSION_UNSUPPORTED"
		msg = "不支持的COMTRADE版本"
	case strings.Contains(s, "unsupported data file type") || strings.Contains(s, "unsupported analog data type"):
		code = "DATA_TYPE_UNSUPPORTED"
		msg = "不支持的数据文件类型，请检查cfg中的data_file_type"
	case strings.Contains(s, "invalid "):
		code = "FORMAT_INVALID"
		msg = "文件内容格式不合法，请检查字段"
	}
	return code, msg, details
}
