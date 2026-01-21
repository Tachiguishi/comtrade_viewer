package main

import (
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func main() {
	r := gin.Default()
	r.MaxMultipartMemory = 128 << 20 // 128MB

	dataRoot := filepath.Join(".", "data")
	_ = ensureDir(dataRoot)

	// 登录接口无需鉴权，需在中间件前注册
	jwtSecret := registerAuthRoutes(r)

	// 全局鉴权
	r.Use(authMiddleware(jwtSecret))

	// 注册 COMTRADE 相关路由
	registerComtradeRoutes(r, dataRoot)

	r.Run(":8080")
}

// --- Error handling helpers ---

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func writeError(c *gin.Context, status int, code string, message string, details any) {
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
