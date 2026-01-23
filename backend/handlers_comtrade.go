package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"comtradeviewer/comtrade"
	"comtradeviewer/storage"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

// DatasetInfo 表示数据集列表中的简要信息
type DatasetInfo struct {
	DatasetID string `json:"datasetId"`
	Name      string `json:"name"`
	CreatedAt int64  `json:"createdAt"`
	SizeBytes int64  `json:"sizeBytes"`
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

// listDatasets 列出存储中的所有数据集目录
func listDatasets(stor storage.Storage) ([]DatasetInfo, error) {
	ctx := context.Background()
	files, err := stor.ListFiles(ctx, "")
	if err != nil {
		return []DatasetInfo{}, nil
	}

	// 按数据集ID分组
	datasets := make(map[string]DatasetInfo)
	for _, file := range files {
		// 文件格式: datasetID/cfg 或 datasetID/dat
		parts := strings.Split(file, "/")
		if len(parts) >= 2 {
			id := parts[0]
			if id == "" || id == "." {
				continue
			}
			size, _ := stor.GetFileSize(ctx, file)
			info := datasets[id]
			info.DatasetID = id
			info.Name = parts[1]
			info.SizeBytes += size
			datasets[id] = info
		}
	}

	out := []DatasetInfo{}
	for _, info := range datasets {
		out = append(out, info)
	}
	return out, nil
}

// saveUploadedFileToStorage 从 multipart 表单中保存指定字段的文件到存储
func saveUploadedFileToStorage(ctx context.Context, stor storage.Storage, fh *multipart.Form, field string, prefix string) error {
	files := fh.File[field]
	if len(files) == 0 {
		return fmt.Errorf("file not found")
	}

	src, err := files[0].Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dest := filepath.Join(prefix, files[0].Filename)
	return stor.SaveFile(ctx, dest, src)
}

// readComtradeFile 从存储读取COMTRADE文件
func readComtradeFile(ctx context.Context, stor storage.Storage, prefix string, ext string) ([]byte, error) {
	entries, err := stor.ListFiles(ctx, prefix)
	if err != nil {
		return nil, err
	}
	
	var path string
	for _, entry := range entries {
		if strings.HasSuffix(strings.ToLower(entry), strings.ToLower(ext)) {
			path = entry
			break
		}
	}

	if path == "" {
		return nil, fmt.Errorf("file with extension %s not found in %s", ext, prefix)
	}

	reader, err := stor.ReadFile(ctx, path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
}

// writeComtradeFile 写入数据到存储
func writeComtradeFile(ctx context.Context, stor storage.Storage, path string, data []byte) error {
	return stor.SaveFile(ctx, path, bytes.NewReader(data))
}

func parseComtrade(cache *comtrade.DatasetCache, stor storage.Storage, id string, ctx context.Context, c *gin.Context) (*comtrade.Metadata, *comtrade.ChannelData, error) {
	var meta *comtrade.Metadata
	var dat *comtrade.ChannelData

	// read from cache first
	if cachedMeta, cachedDat, ok := cache.Get(id); ok {
		meta = cachedMeta
		dat = cachedDat
		fmt.Printf("Cache hit for dataset %s\n", id)
	} else {
		fmt.Printf("Cache miss for dataset %s, parsing from storage\n", id)

		cfgBytes, err := readComtradeFile(ctx, stor, id, "cfg")
		if err != nil {
			return nil, nil, err
		}

		datBytes, err := readComtradeFile(ctx, stor, id, "dat")
		if err != nil {
			return nil, nil, err
		}

		m, d, err := comtrade.ParseComtradeFromBytes(cfgBytes, datBytes)
		if err != nil {
			return nil, nil, err
		}
		meta = m
		dat = d

		cache.Set(id, meta, dat)
	}

	return meta, dat, nil
}

// registerComtradeRoutes 注册与 COMTRADE 相关的所有接口
func registerComtradeRoutes(r *gin.Engine, stor storage.Storage) {
	// Initialize LRU cache: keep last 10 datasets in memory
	cache := comtrade.NewDatasetCache(10)

	// 上传
	r.POST("/api/datasets/import", func(c *gin.Context) {
		if err := c.Request.ParseMultipartForm(256 << 20); err != nil {
			writeError(c, http.StatusBadRequest, "INVALID_FORM", "无效的表单数据", gin.H{"hint": "请通过multipart/form-data提交.cfg与.dat文件"})
			return
		}

		ctx := c.Request.Context()
		fh := c.Request.MultipartForm
		datasetID := strconv.FormatInt(time.Now().UnixNano(), 10)

		// 校验文件存在与扩展名
		if !hasFileField(fh, "cfg") {
			writeError(c, http.StatusBadRequest, "CFG_MISSING", ".cfg文件缺失", gin.H{"hint": "请选择配置文件(.cfg)"})
			return
		}
		if !hasFileField(fh, "dat") {
			writeError(c, http.StatusBadRequest, "DAT_MISSING", ".dat文件缺失", gin.H{"hint": "请选择数据文件(.dat)"})
			return
		}
		if !hasFileExt(fh, "cfg", ".cfg") {
			writeError(c, http.StatusBadRequest, "CFG_EXT_INVALID", "配置文件扩展名无效", gin.H{"expected": ".cfg"})
			return
		}
		if !hasFileExt(fh, "dat", ".dat") {
			writeError(c, http.StatusBadRequest, "DAT_EXT_INVALID", "数据文件扩展名无效", gin.H{"expected": ".dat"})
			return
		}

		if err := saveUploadedFileToStorage(ctx, stor, fh, "cfg", datasetID); err != nil {
			writeError(c, http.StatusBadRequest, "CFG_SAVE_FAILED", "保存配置文件失败", gin.H{"detail": err.Error()})
			return
		}
		if err := saveUploadedFileToStorage(ctx, stor, fh, "dat", datasetID); err != nil {
			writeError(c, http.StatusBadRequest, "DAT_SAVE_FAILED", "保存数据文件失败", gin.H{"detail": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"datasetId": datasetID, "name": datasetID})
	})

	// 数据集列表
	r.GET("/api/datasets", func(c *gin.Context) {
		lst, err := listDatasets(stor)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "LIST_ERROR", "获取数据集列表失败", gin.H{"detail": err.Error()})
			return
		}
		c.JSON(http.StatusOK, lst)
	})

	// 元数据
	r.GET("/api/datasets/:id/metadata", func(c *gin.Context) {
		id := c.Param("id")
		ctx := c.Request.Context()

		var meta *comtrade.Metadata

		// 从cfg文件解析
		cfgData, err := readComtradeFile(ctx, stor, id, "cfg")
		if err != nil {
			writeError(c, http.StatusNotFound, "METADATA_NOT_FOUND", "未找到元数据", gin.H{"id": id})
			return
		}

		meta, err = comtrade.ParseComtradeCFGFromBytes(cfgData)
		if err != nil {
			writeError(c, http.StatusNotFound, "METADATA_NOT_FOUND", "未找到元数据", gin.H{"id": id})
			return
		}

		c.JSON(http.StatusOK, meta)
	})

	// 波形数据
	r.GET("/api/datasets/:id/waveforms", gzip.Gzip(gzip.BestCompression), func(c *gin.Context) {
		id := c.Param("id")
		ctx := c.Request.Context()

		lastTime := time.Now()

		meta, dat, err := parseComtrade(cache, stor, id, ctx, c)
		if err != nil {
			code, msg, details := toFriendlyParseError(err)
			writeError(c, http.StatusInternalServerError, code, msg, details)
			return
		}

		currentTime := time.Now()
		fmt.Printf("Time taken to load data: %v\n", currentTime.Sub(lastTime))

		if len(dat.Timestamps) == 0 {
			writeError(c, http.StatusInternalServerError, "NO_DATA", "未找到通道数据", gin.H{"id": id})
			return
		}

		// 解析通道参数
		analogChsStr := c.Query("A")
		analogChannels := make([]int, 0)
		if analogChsStr != "" {
			chs := strings.SplitSeq(analogChsStr, ",")
			for chID := range chs {
				chID = strings.TrimSpace(chID)
				if chID == "" {
					continue
				}
				chNum, err := strconv.Atoi(chID)
				if err != nil {
					continue
				}
				analogChannels = append(analogChannels, chNum)
			}
			sort.Ints(analogChannels)
		}

		digitalChsStr := c.Query("D")
		digitalChannels := make([]int, 0)
		if digitalChsStr != "" {
			chs := strings.SplitSeq(digitalChsStr, ",")
			for chID := range chs {
				chID = strings.TrimSpace(chID)
				if chID == "" {
					continue
				}
				chNum, err := strconv.Atoi(chID)
				if err != nil {
					continue
				}
				digitalChannels = append(digitalChannels, chNum)
			}
			sort.Ints(digitalChannels)
		}

		if len(analogChannels) == 0 && len(digitalChannels) == 0 {
			writeError(c, http.StatusBadRequest, "NO_CHANNELS_SPECIFIED", "no channel specified", gin.H{"hint": "请通过查询参数A和D指定所需的模拟和数字通道, 例如?A=1,2,3&D=1,2"})
			return
		}

		// 下采样参数
		targetPoints := 5000 // 默认
		if tp := c.Query("targetPoints"); tp != "" {
			if v, err := strconv.Atoi(tp); err == nil && v > 0 {
				targetPoints = v
			}
		}
		downsampleMethod := c.DefaultQuery("downsample", "auto") // auto, none, lttb, minmax

		// 时间轴
		timestamps := comtrade.ComputeTimeAxisFromMeta(*meta, dat.Timestamps, len(dat.Timestamps))

		// 时间范围: 默认显示10%数据点的范围
		startTime := timestamps[0]
		endTimeIndex := int(math.Max(5000, float64(len(timestamps)/20)))
		if endTimeIndex >= len(timestamps) {
			endTimeIndex = len(timestamps) - 1
		}
		endTime := timestamps[endTimeIndex]
		if st := c.Query("startTime"); st != "" {
			if v, err := strconv.ParseFloat(st, 32); err == nil {
				startTime = float32(v)
			}
		}
		if et := c.Query("endTime"); et != "" {
			if v, err := strconv.ParseFloat(et, 32); err == nil {
				endTime = float32(v)
			}
		}

		// 过滤时间范围
		var filteredTimestamps []float32
		var timeIndices []int
		if (startTime != 0 || endTime != 0) && startTime < endTime {
			for i, t := range timestamps {
				if t >= startTime && t <= endTime {
					timeIndices = append(timeIndices, i)
					filteredTimestamps = append(filteredTimestamps, t)
				}
			}
		} else {
			filteredTimestamps = timestamps
			timeIndices = make([]int, len(timestamps))
			for i := range timestamps {
				timeIndices[i] = i
			}
		}

		// 是否需要下采样
		needDownsample := false
		switch downsampleMethod {
		case "auto":
			needDownsample = len(filteredTimestamps) > targetPoints*2
		case "lttb", "minmax":
			needDownsample = len(filteredTimestamps) > targetPoints
		case "none":
			needDownsample = false
			downsampleMethod = "none"
		}

		// 构造返回数据
		series := make([]map[string]any, 0, len(analogChannels)+len(digitalChannels))

		// 模拟量
		for _, chData := range dat.AnalogChannels {
			if len(analogChannels) == 0 {
				break
			}

			found := false
			for _, chNum := range analogChannels {
				if chData.ChannelNumber == chNum {
					found = true
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

			var multiplier, offset float64 = 1.0, 0.0
			if chNum-1 < len(meta.AnalogChannels) {
				ch := meta.AnalogChannels[chNum-1]
				multiplier = ch.Multiplier
				offset = ch.Offset
			}

			if len(chData.RawDataFloat) == sampleLen {
				for i, d := range chData.RawDataFloat {
					y[i] = float64(d)*multiplier + offset
				}
			} else {
				for i, d := range chData.RawData {
					y[i] = float64(d)*multiplier + offset
				}
			}

			var rangeTimestamps []float32
			var rangeY []float64
			for _, idx := range timeIndices {
				rangeTimestamps = append(rangeTimestamps, timestamps[idx])
				rangeY = append(rangeY, y[idx])
			}

			returnTimes := rangeTimestamps
			returnY := rangeY
			if needDownsample && len(rangeTimestamps) > 0 {
				returnTimes, returnY = comtrade.DownsampleLTTB(rangeTimestamps, rangeY, targetPoints)
			}

			series = append(series, map[string]any{
				"channel": chNum,
				"type":    "analog",
				"name":    meta.AnalogChannels[chNum-1].ChannelName,
				"unit":    meta.AnalogChannels[chNum-1].Unit,
				"times":   returnTimes,
				"y":       returnY,
			})
		}

		// 开关量
		for _, chData := range dat.DigitalChannels {
			if len(digitalChannels) == 0 {
				break
			}

			found := false
			for _, chNum := range digitalChannels {
				if chData.ChannelNumber == chNum {
					found = true
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

			var rangeTimestamps []float32
			var rangeY []int8
			for _, idx := range timeIndices {
				if idx < len(y) {
					rangeTimestamps = append(rangeTimestamps, timestamps[idx])
					rangeY = append(rangeY, y[idx])
				}
			}

			returnTimes := rangeTimestamps
			returnY := rangeY
			if needDownsample && len(rangeTimestamps) > 0 {
				returnTimes, returnY = comtrade.DownsampleDigital(rangeTimestamps, rangeY)
			}

			series = append(series, map[string]any{
				"channel": chNum,
				"type":    "digital",
				"name":    meta.DigitalChannels[chNum-1].ChannelName,
				"times":   returnTimes,
				"y":       returnY,
			})
		}

		response := gin.H{
			"series": series,
			"window": map[string]float32{"start": timestamps[0], "end": timestamps[len(timestamps)-1]},
			"timeRange": map[string]float32{"start": startTime, "end": endTime},
		}

		response["downsample"] = map[string]any{
			"method":         downsampleMethod,
			"targetPoints":   targetPoints,
			"originalPoints": len(timestamps),
		}

		c.JSON(http.StatusOK, response)
	})

	// WaveCanvas 数据
	r.GET("/api/datasets/:id/wavecanvas", gzip.Gzip(gzip.BestSpeed), func(c *gin.Context) {
		id := c.Param("id")
		ctx := c.Request.Context()

		meta, dat, err := parseComtrade(cache, stor, id, ctx, c)
		if err != nil {
			code, msg, details := toFriendlyParseError(err)
			writeError(c, http.StatusInternalServerError, code, msg, details)
			return
		}

		sampleInfo := make([]map[string]any, 0)
		for _, sample := range meta.SampleRates {
			sampleInfo = append(sampleInfo, map[string]any{
				"samp":    sample.SampRate,
				"endsamp": sample.LastSampleNum,
			})
		}

		selecters := make([]map[string]any, 0)
		channels := make([]map[string]any, 0)
		for _, ch := range meta.AnalogChannels {
			selecters = append(selecters, map[string]any{
				"channel":   ch.ChannelNumber,
				"groupName": ch.ChannelName,
				"phase":     ch.Phase,
				"AD":        "A",
			})

			channels = append(channels, map[string]any{
				"name":    ch.ChannelName,
				"uu":      ch.Unit,
				"a":       ch.Multiplier,
				"b":       ch.Offset,
				"ptct":    ch.Primary / ch.Secondary,
				"ps":      ch.PS,
				"max":     ch.MaxValue,
				"min":     ch.MinValue,
				"analyse": 1,
				"y":       dat.AnalogChannels[ch.ChannelNumber-1].RawData,
				"skew":    ch.Skew,
			})
		}
		for _, ch := range meta.DigitalChannels {
			selecters = append(selecters, map[string]any{
				"channel":   ch.ChannelNumber,
				"groupName": ch.ChannelName,
				"AD":        "D",
			})
			channels = append(channels, map[string]any{
				"name":    ch.ChannelName,
				"uu":      "",
				"a":       0,
				"b":       0,
				"ptct":    0,
				"ps":      "",
				"max":     1,
				"min":     1,
				"analyse": 0,
				"y":       dat.DigitalChannels[ch.ChannelNumber-1].RawData,
				"skew":    0,
			})
		}

		response := gin.H{
			"beginTime":  meta.StartTime,
			"sampleInfo": sampleInfo,
			"ts":         dat.Timestamps,
			"allSelector": selecters,
			"chns":       channels,
		}

		c.JSON(http.StatusOK, response)
	})

	// 标注（文件持久化）
	r.GET("/api/datasets/:id/annotations", func(c *gin.Context) {
		id := c.Param("id")
		ctx := c.Request.Context()

		data, err := readComtradeFile(ctx, stor, id, "annotations.json")
		if err != nil {
			c.JSON(http.StatusOK, []any{})
			return
		}

		var out []map[string]any
		_ = json.Unmarshal(data, &out)
		c.JSON(http.StatusOK, out)
	})

	r.POST("/api/datasets/:id/annotations", func(c *gin.Context) {
		id := c.Param("id")
		ctx := c.Request.Context()

		var ann map[string]any
		if err := c.BindJSON(&ann); err != nil {
			writeError(c, http.StatusBadRequest, "BAD_JSON", "JSON格式错误", gin.H{"detail": err.Error()})
			return
		}

		var out []map[string]any
		if data, err := readComtradeFile(ctx, stor, id, "annotations.json"); err == nil {
			_ = json.Unmarshal(data, &out)
		}

		ann["id"] = strconv.FormatInt(time.Now().UnixNano(), 10)
		out = append(out, ann)

		if b, err := json.MarshalIndent(out, "", "  "); err == nil {
			path := filepath.Join(id, "annotations.json")
			_ = writeComtradeFile(ctx, stor, path, b)
		}

		c.JSON(http.StatusOK, gin.H{"id": ann["id"]})
	})

	r.DELETE("/api/datasets/:id/annotations/:annId", func(c *gin.Context) {
		id := c.Param("id")
		annID := c.Param("annId")
		ctx := c.Request.Context()

		var out []map[string]any
		if data, err := readComtradeFile(ctx, stor, id, "annotations.json"); err == nil {
			_ = json.Unmarshal(data, &out)
		}

		kept := make([]map[string]any, 0, len(out))
		for _, a := range out {
			if v, ok := a["id"].(string); ok && v == annID {
				continue
			}
			kept = append(kept, a)
		}

		if b, err := json.MarshalIndent(kept, "", "  "); err == nil {
			p := filepath.Join(id, "annotations.json")
			if err := writeComtradeFile(ctx, stor, p, b); err != nil {
				writeError(c, http.StatusInternalServerError, "ANNOTATIONS_WRITE_ERROR", "写入标注失败", gin.H{"detail": err.Error()})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
}

func removeInt(source []int, target int) []int {
	for i, v := range source {
		if v == target {
			return append(source[:i], source[i+1:]...)
		}
	}
	return source
}
